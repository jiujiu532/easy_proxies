package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"


	"easy_proxies/internal/boxmgr"
	"easy_proxies/internal/config"
	"easy_proxies/internal/monitor"
	"easy_proxies/internal/node"
	"easy_proxies/internal/proxypool"
	"easy_proxies/internal/store"
	"easy_proxies/internal/subscription"
)

// Run builds the runtime components from config and blocks until shutdown.
func Run(ctx context.Context, cfg *config.Config) error {
	// Initialize store for enhanced features
	dataDir := filepath.Join(filepath.Dir(cfg.FilePath()), "data")
	st, err := store.NewStore(dataDir)
	if err != nil {
		fmt.Printf("⚠️  Failed to initialize store: %v (continuing without persistence)\n", err)
		st, _ = store.NewStore("") // In-memory store
	}

	// Configure latency thresholds from config
	if cfg.LatencyGroups.LowThreshold > 0 || cfg.LatencyGroups.MediumThreshold > 0 {
		latencyCfg := store.LatencyConfig{
			LowThreshold:    cfg.LatencyGroups.LowThreshold,
			MediumThreshold: cfg.LatencyGroups.MediumThreshold,
		}
		if latencyCfg.LowThreshold <= 0 {
			latencyCfg.LowThreshold = 100
		}
		if latencyCfg.MediumThreshold <= 0 {
			latencyCfg.MediumThreshold = 300
		}
		st.SetLatencyConfig(latencyCfg)
	}

	// Load subscription configs into store
	for _, subURL := range cfg.Subscriptions {
		sub := &store.Subscription{
			URL:     subURL,
			Enabled: true,
		}
		st.AddSubscription(sub)
	}
	for _, subCfg := range cfg.SubscriptionConfigs {
		sub := &store.Subscription{
			ID:              subCfg.ID,
			Name:            subCfg.Name,
			URL:             subCfg.URL,
			Enabled:         subCfg.Enabled,
			RefreshInterval: subCfg.RefreshInterval,
		}
		st.AddSubscription(sub)
	}

	// Clean up duplicate subscriptions (same URL)
	if removed := st.DeduplicateSubscriptions(); removed > 0 {
		fmt.Printf("🧹 Cleaned up %d duplicate subscriptions\n", removed)
	}

	proxyUsername := cfg.Listener.Username
	proxyPassword := cfg.Listener.Password
	if cfg.Mode == "multi-port" || cfg.Mode == "hybrid" {
		proxyUsername = cfg.MultiPort.Username
		proxyPassword = cfg.MultiPort.Password
	}

	monitorCfg := monitor.Config{
		Enabled:        cfg.ManagementEnabled(),
		Listen:         cfg.Management.Listen,
		ProbeTarget:    cfg.Management.ProbeTarget,
		Password:       cfg.Management.Password,
		ProxyUsername:  proxyUsername,
		ProxyPassword:  proxyPassword,
		ExternalIP:     cfg.ExternalIP,
		SkipCertVerify: cfg.SkipCertVerify,
	}

	// Create and start BoxManager
	boxMgr := boxmgr.New(cfg, monitorCfg)
	if err := boxMgr.Start(ctx); err != nil {
		return fmt.Errorf("start box manager: %w", err)
	}
	defer boxMgr.Close()

	// Initialize proxy pool with rotation mode from config
	poolMode := store.PoolModeSequential
	switch cfg.Pool.Mode {
	case "random":
		poolMode = store.PoolModeRandom
	case "latency_first":
		poolMode = store.PoolModeLatencyFirst
	case "weighted":
		poolMode = store.PoolModeWeighted
	}

	pool := proxypool.NewProxyPool(st, proxypool.Config{
		Mode:            poolMode,
		FallbackEnabled: true,
		APIKey:          cfg.APIAuth.Key,
	})

	// Sync nodes from boxMgr to store with region detection
	syncNodesToStore(boxMgr, st, cfg)

	// Refresh pool nodes
	pool.RefreshNodes()

	// Wire up config to monitor server for settings API
	if server := boxMgr.MonitorServer(); server != nil {
		server.SetConfig(cfg)

		// Register new API handlers
		poolHandler := monitor.NewProxyPoolHandler(pool, st)
		poolHandler.SetConfig(cfg)
		poolHandler.SetNodeManager(boxMgr)
		poolHandler.SetMonitorManager(boxMgr.MonitorManager())
		poolHandler.RegisterRoutes(server.Mux(), server.WithAuth)
	}

	// Create and start SubscriptionManager if enabled
	var subMgr *subscription.Manager
	if cfg.SubscriptionRefresh.Enabled && (len(cfg.Subscriptions) > 0 || len(cfg.SubscriptionConfigs) > 0) {
		subMgr = subscription.New(cfg, boxMgr)
		subMgr.Start()
		defer subMgr.Stop()

		// Wire up subscription manager to monitor server for API endpoints
		if server := boxMgr.MonitorServer(); server != nil {
			server.SetSubscriptionRefresher(subMgr)
		}
	}

	// Start auto speedtest if enabled
	if cfg.AutoSpeedtest.Enabled {
		interval := parseInterval(cfg.AutoSpeedtest.Interval, 30*time.Minute)
		go runAutoSpeedtest(ctx, boxMgr, st, pool, interval)
		fmt.Printf("✅ Auto speedtest enabled, interval: %v\n", interval)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
	case sig := <-sigCh:
		fmt.Printf("received %s, shutting down\n", sig)
	}

	// Save store data before shutdown
	if err := st.Save(); err != nil {
		fmt.Printf("⚠️  Failed to save store: %v\n", err)
	}

	return nil
}

// detectNodeType returns the protocol type from URI
func detectNodeType(uri string) string {
	if strings.Contains(uri, "://") {
		parts := strings.SplitN(uri, "://", 2)
		return parts[0]
	}
	return "unknown"
}

// syncNodesToStore syncs nodes from boxMgr to store with region detection
func syncNodesToStore(boxMgr *boxmgr.Manager, st *store.Store, cfg *config.Config) {
	for _, nodeCfg := range cfg.Nodes {
		// Detect region from node name
		regionInfo := node.DetectRegion(nodeCfg.Name)

		enhancedNode := &store.EnhancedNode{
			Name:       nodeCfg.Name,
			URI:        nodeCfg.URI,
			Port:       nodeCfg.Port,
			Type:       detectNodeType(nodeCfg.URI),
			Region:     regionInfo.Code,
			RegionName: regionInfo.Name,
			Status:     store.NodeStatusEnabled,
			Available:  true,
			Latency:    -1, // Unknown until tested
			LatencyLevel: store.LatencyLevelUnknown,
		}

		// Set subscription info if available
		if nodeCfg.Source == config.NodeSourceSubscription {
			enhancedNode.SubscriptionName = "subscription"
		}

		st.UpdateNodeState(enhancedNode)
	}
}


// runAutoSpeedtest runs periodic speed tests and updates latency groups
func runAutoSpeedtest(ctx context.Context, boxMgr *boxmgr.Manager, st *store.Store, pool *proxypool.ProxyPool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get all node snapshots from monitor
			if mgr := boxMgr.MonitorManager(); mgr != nil {
				snapshots := mgr.Snapshot()
				for _, snap := range snapshots {
					if nodeState, ok := st.GetNodeState(snap.Name); ok {
						// Update latency
						nodeState.Latency = snap.LastLatencyMs
						nodeState.LatencyLevel = st.CalculateLatencyLevel(snap.LastLatencyMs)
						nodeState.Available = snap.Available
						nodeState.FailureCount = snap.FailureCount
						nodeState.SuccessCount = snap.SuccessCount
						nodeState.LastCheckAt = time.Now()
						st.UpdateNodeState(nodeState)
					}
				}
			}
			// Refresh pool after speedtest
			pool.RefreshNodes()
			fmt.Println("✅ Auto speedtest completed, pool refreshed")
		}
	}
}

// parseInterval parses duration string, returns default if invalid
func parseInterval(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

