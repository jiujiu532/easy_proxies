package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"easy_proxies/internal/config"
	"easy_proxies/internal/proxypool"
	"easy_proxies/internal/store"
)

// ProxyPoolHandler handles proxy pool API requests
type ProxyPoolHandler struct {
	pool       *proxypool.ProxyPool
	store      *store.Store
	cfg        *config.Config
	nodeMgr    NodeManager
	monitorMgr *Manager
}

// NewProxyPoolHandler creates a new handler
func NewProxyPoolHandler(pool *proxypool.ProxyPool, st *store.Store) *ProxyPoolHandler {
	return &ProxyPoolHandler{
		pool:  pool,
		store: st,
	}
}

// SetConfig sets the configuration for subscription updates
func (h *ProxyPoolHandler) SetConfig(cfg *config.Config) {
	h.cfg = cfg
}

// SetNodeManager sets the node manager for triggering reloads
func (h *ProxyPoolHandler) SetNodeManager(nm NodeManager) {
	h.nodeMgr = nm
}

// SetMonitorManager sets the monitor manager for unified node data
func (h *ProxyPoolHandler) SetMonitorManager(mgr *Manager) {
	h.monitorMgr = mgr
}

// RegisterRoutes registers proxy pool API routes
func (h *ProxyPoolHandler) RegisterRoutes(mux *http.ServeMux, withAuth func(http.HandlerFunc) http.HandlerFunc) {
	// Proxy Pool API (public or with optional API key auth)
	mux.HandleFunc("/api/proxy/get", h.handleGetProxy)
	mux.HandleFunc("/api/proxy/list", h.handleListProxies)
	mux.HandleFunc("/api/proxy/stats", h.handleStats)

	// Subscription Management API (requires auth)
	mux.HandleFunc("/api/subscriptions", withAuth(h.handleSubscriptions))
	mux.HandleFunc("/api/subscriptions/", withAuth(h.handleSubscriptionItem))

	// Node Status API (requires auth)
	mux.HandleFunc("/api/nodes/status/", withAuth(h.handleNodeStatus))

	// Group API (requires auth)
	mux.HandleFunc("/api/groups/latency", withAuth(h.handleGroupsByLatency))
	mux.HandleFunc("/api/groups/region", withAuth(h.handleGroupsByRegion))
	mux.HandleFunc("/api/groups/subscription", withAuth(h.handleGroupsBySubscription))
}

// --- Proxy Pool API ---

// handleGetProxy returns a single proxy based on filters
// GET /api/proxy/get?latency=low&region=US&sub=xxx&key=apikey
func (h *ProxyPoolHandler) handleGetProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key if configured
	apiKey := r.URL.Query().Get("key")
	if !h.pool.ValidateAPIKey(apiKey) {
		w.WriteHeader(http.StatusUnauthorized)
		writePoolJSON(w, map[string]any{"error": "Invalid API key"})
		return
	}

	// Parse filters
	regionFilter := strings.ToUpper(r.URL.Query().Get("region"))
	latencyFilter := r.URL.Query().Get("latency")

	// Get available nodes from monitor manager (single source of truth)
	var selectedNode *Snapshot
	if h.monitorMgr != nil {
		snapshots := h.monitorMgr.Snapshot()
		var candidates []Snapshot
		
		for _, snap := range snapshots {
			// Only include available nodes
			if !snap.Available {
				continue
			}
			
			// Apply region filter
			if regionFilter != "" && snap.Region != regionFilter {
				continue
			}
			
			// Apply latency filter
			if latencyFilter != "" {
				latencyLevel := h.classifyLatency(snap.LastLatencyMs)
				if string(latencyLevel) != latencyFilter {
					continue
				}
			}
			
			candidates = append(candidates, snap)
		}
		
		// Select best node (lowest latency with valid latency > 0)
		if len(candidates) > 0 {
			bestIdx := 0
			bestLatency := int64(999999)
			for i, c := range candidates {
				lat := c.LastLatencyMs
				if lat > 0 && lat < bestLatency {
					bestLatency = lat
					bestIdx = i
				}
			}
			selectedNode = &candidates[bestIdx]
		}
	}

	if selectedNode == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		writePoolJSON(w, map[string]any{"error": "no available proxy"})
		return
	}

	// Return proxy URL
	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", selectedNode.Port)
	if selectedNode.Port == 0 {
		proxyURL = selectedNode.URI
	}

	// Check response format
	format := r.URL.Query().Get("format")
	if format == "json" {
		writePoolJSON(w, map[string]any{
			"proxy":         proxyURL,
			"name":          selectedNode.Tag,
			"region":        selectedNode.Region,
			"latency":       selectedNode.LastLatencyMs,
			"latency_level": h.classifyLatency(selectedNode.LastLatencyMs),
		})
		return
	}

	// Default: return plain text proxy URL
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(proxyURL))
}

// handleListProxies returns list of proxies
// GET /api/proxy/list?latency=low&region=US&limit=10
func (h *ProxyPoolHandler) handleListProxies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key
	apiKey := r.URL.Query().Get("key")
	if !h.pool.ValidateAPIKey(apiKey) {
		w.WriteHeader(http.StatusUnauthorized)
		writePoolJSON(w, map[string]any{"error": "Invalid API key"})
		return
	}

	// Parse filters
	regionFilter := strings.ToUpper(r.URL.Query().Get("region"))
	latencyFilter := r.URL.Query().Get("latency")
	limit := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Build response from monitor manager (single source of truth for latency)
	var proxies []map[string]any
	
	if h.monitorMgr != nil {
		snapshots := h.monitorMgr.Snapshot()
		for _, snap := range snapshots {
			// Apply region filter
			if regionFilter != "" && snap.Region != regionFilter {
				continue
			}
			
			// Apply latency filter
			if latencyFilter != "" {
				latencyLevel := h.classifyLatency(snap.LastLatencyMs)
				if string(latencyLevel) != latencyFilter {
					continue
				}
			}
			
			// Only include available nodes
			if !snap.Available {
				continue
			}
			
			proxies = append(proxies, map[string]any{
				"proxy":         fmt.Sprintf("http://127.0.0.1:%d", snap.Port),
				"name":          snap.Tag,
				"uri":           snap.URI,
				"type":          snap.Mode, // Use Mode as type
				"region":        snap.Region,
				"region_name":   snap.RegionName,
				"latency":       snap.LastLatencyMs,
				"latency_level": h.classifyLatency(snap.LastLatencyMs),
				"subscription":  "", // Not available in Snapshot
				"failure_count": snap.FailureCount,
				"status":        "online",
			})
		}
		
		// Apply limit
		if limit > 0 && len(proxies) > limit {
			proxies = proxies[:limit]
		}
	}
	
	if proxies == nil {
		proxies = []map[string]any{}
	}

	writePoolJSON(w, map[string]any{
		"count":   len(proxies),
		"proxies": proxies,
	})
}

// classifyLatency returns latency level based on ms value
func (h *ProxyPoolHandler) classifyLatency(ms int64) store.LatencyLevel {
	if ms <= 0 {
		return store.LatencyLevelUnknown
	}
	if ms <= 100 {
		return store.LatencyLevelLow
	}
	if ms <= 300 {
		return store.LatencyLevelMedium
	}
	return store.LatencyLevelHigh
}


// handleStats returns pool statistics from monitor manager (single source of truth)
func (h *ProxyPoolHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If monitor manager is available, use it as single source of truth
	if h.monitorMgr != nil {
		snapshots := h.monitorMgr.Snapshot()
		stats := struct {
			TotalNodes     int            `json:"total_nodes"`
			AvailableNodes int            `json:"available_nodes"`
			Mode           string         `json:"mode"`
			ByLatency      map[string]int `json:"by_latency"`
			ByRegion       map[string]int `json:"by_region"`
		}{
			TotalNodes:     len(snapshots),
			AvailableNodes: 0,
			Mode:           "monitor",
			ByLatency:      make(map[string]int),
			ByRegion:       make(map[string]int),
		}

		for _, snap := range snapshots {
			if snap.Available && !snap.Blacklisted {
				stats.AvailableNodes++
				// Categorize by latency
				latencyMs := snap.LastLatencyMs
				if latencyMs <= 0 {
					stats.ByLatency["unknown"]++
				} else if latencyMs <= 100 {
					stats.ByLatency["low"]++
				} else if latencyMs <= 300 {
					stats.ByLatency["medium"]++
				} else {
					stats.ByLatency["high"]++
				}
			}
		}

		writePoolJSON(w, stats)
		return
	}

	// Fallback to pool stats
	stats := h.pool.Stats()
	writePoolJSON(w, stats)
}

// --- Subscription Management API ---

// handleSubscriptions handles list and create operations
func (h *ProxyPoolHandler) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		subs := h.store.ListSubscriptions()
		writePoolJSON(w, map[string]any{"subscriptions": subs})

	case http.MethodPost:
		var sub store.Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writePoolJSON(w, map[string]any{"error": "Invalid request body"})
			return
		}

		if sub.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			writePoolJSON(w, map[string]any{"error": "URL is required"})
			return
		}

		if sub.Name == "" {
			// Extract name from URL
			u, _ := url.Parse(sub.URL)
			if u != nil {
				sub.Name = u.Host
			} else {
				sub.Name = "Subscription"
			}
		}

		sub.Enabled = true
		if err := h.store.AddSubscription(&sub); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writePoolJSON(w, map[string]any{"error": err.Error()})
			return
		}

		// Also add to config file for reload to pick up
		configUpdated := false
		if h.cfg != nil {
			// Check if URL already exists
			exists := false
			for _, existingURL := range h.cfg.Subscriptions {
				if existingURL == sub.URL {
					exists = true
					break
				}
			}
			if !exists {
				h.cfg.Subscriptions = append(h.cfg.Subscriptions, sub.URL)
				if err := h.cfg.SaveSubscriptions(); err == nil {
					configUpdated = true
				}
			}
		}

		// Auto trigger reload after adding subscription (async with delay to avoid conflicts)
		reloadTriggered := false
		if h.nodeMgr != nil && configUpdated {
			reloadTriggered = true
			go func() {
				// Delay a bit to ensure all cleanup is done
				time.Sleep(500 * time.Millisecond)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_ = h.nodeMgr.TriggerReload(ctx)
			}()
		}

		writePoolJSON(w, map[string]any{
			"message":          "Subscription added",
			"subscription":     sub,
			"config_updated":   configUpdated,
			"reload_triggered": reloadTriggered,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSubscriptionItem handles single subscription operations
func (h *ProxyPoolHandler) handleSubscriptionItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		writePoolJSON(w, map[string]any{"error": "Subscription ID required"})
		return
	}

	// Check for action suffix
	parts := strings.Split(id, "/")
	id = parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch r.Method {
	case http.MethodGet:
		sub, err := h.store.GetSubscription(id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			writePoolJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writePoolJSON(w, sub)

	case http.MethodPut:
		var sub store.Subscription
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writePoolJSON(w, map[string]any{"error": "Invalid request body"})
			return
		}
		sub.ID = id
		if err := h.store.UpdateSubscription(&sub); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writePoolJSON(w, map[string]any{"error": err.Error()})
			return
		}
		writePoolJSON(w, map[string]any{"message": "Subscription updated", "subscription": sub})

	case http.MethodDelete:
		// Get subscription URL before deleting
		sub, _ := h.store.GetSubscription(id)
		subURL := ""
		if sub != nil {
			subURL = sub.URL
		}

		if err := h.store.DeleteSubscription(id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			writePoolJSON(w, map[string]any{"error": err.Error()})
			return
		}

		// Also remove from config file
		configUpdated := false
		if h.cfg != nil && subURL != "" {
			newSubs := make([]string, 0, len(h.cfg.Subscriptions))
			for _, u := range h.cfg.Subscriptions {
				if u != subURL {
					newSubs = append(newSubs, u)
				}
			}
			if len(newSubs) < len(h.cfg.Subscriptions) {
				h.cfg.Subscriptions = newSubs
				if err := h.cfg.SaveSubscriptions(); err == nil {
					configUpdated = true
				}
			}
		}

		// Auto trigger reload after deleting subscription (async with delay to avoid conflicts)
		reloadTriggered := false
		if h.nodeMgr != nil {
			reloadTriggered = true
			go func() {
				// Delay a bit to ensure all cleanup is done
				time.Sleep(500 * time.Millisecond)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_ = h.nodeMgr.TriggerReload(ctx)
			}()
		}

		writePoolJSON(w, map[string]any{
			"message":          "Subscription deleted",
			"config_updated":   configUpdated,
			"reload_triggered": reloadTriggered,
		})

	case http.MethodPost:
		if action == "refresh" {
			// Trigger refresh for this subscription
			// This would need integration with subscription manager
			writePoolJSON(w, map[string]any{"message": "Refresh triggered", "subscription_id": id})
			return
		}
		if action == "toggle" {
			sub, err := h.store.GetSubscription(id)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				writePoolJSON(w, map[string]any{"error": err.Error()})
				return
			}
			sub.Enabled = !sub.Enabled
			h.store.UpdateSubscription(sub)
			writePoolJSON(w, map[string]any{
				"message": "Subscription toggled",
				"enabled": sub.Enabled,
			})
			return
		}
		http.Error(w, "Unknown action", http.StatusBadRequest)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- Node Status API ---

// handleNodeStatus handles node enable/disable/blacklist
func (h *ProxyPoolHandler) handleNodeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/nodes/status/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		writePoolJSON(w, map[string]any{"error": "Node name and action required"})
		return
	}

	nodeName, _ := url.PathUnescape(parts[0])
	action := parts[1]

	var status store.NodeStatus
	switch action {
	case "enable":
		status = store.NodeStatusEnabled
	case "disable":
		status = store.NodeStatusDisabled
	case "blacklist":
		status = store.NodeStatusBlacklisted
	default:
		w.WriteHeader(http.StatusBadRequest)
		writePoolJSON(w, map[string]any{"error": "Invalid action. Use: enable, disable, blacklist"})
		return
	}

	if err := h.store.SetNodeStatus(nodeName, status); err != nil {
		w.WriteHeader(http.StatusNotFound)
		writePoolJSON(w, map[string]any{"error": err.Error()})
		return
	}

	// Refresh pool after status change
	h.pool.RefreshNodes()

	writePoolJSON(w, map[string]any{
		"message": fmt.Sprintf("Node %s status changed to %s", nodeName, status),
		"node":    nodeName,
		"status":  status,
	})
}

// --- Group API ---

// handleGroupsByLatency returns nodes grouped by latency (from monitor manager)
func (h *ProxyPoolHandler) handleGroupsByLatency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := make(map[string][]map[string]any)
	result["low"] = []map[string]any{}
	result["medium"] = []map[string]any{}
	result["high"] = []map[string]any{}
	result["unknown"] = []map[string]any{}

	if h.monitorMgr != nil {
		snapshots := h.monitorMgr.Snapshot()
		for _, snap := range snapshots {
			if !snap.Available || snap.Blacklisted {
				continue
			}
			node := map[string]any{
				"name":    snap.Name,
				"latency": snap.LastLatencyMs,
			}

			latencyMs := snap.LastLatencyMs
			if latencyMs <= 0 {
				result["unknown"] = append(result["unknown"], node)
			} else if latencyMs <= 100 {
				result["low"] = append(result["low"], node)
			} else if latencyMs <= 300 {
				result["medium"] = append(result["medium"], node)
			} else {
				result["high"] = append(result["high"], node)
			}
		}
	}

	writePoolJSON(w, map[string]any{
		"groups": result,
		"config": h.store.GetLatencyConfig(),
	})
}

// handleGroupsByRegion returns nodes grouped by region (from monitor manager)
func (h *ProxyPoolHandler) handleGroupsByRegion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := make(map[string][]map[string]any)

	if h.monitorMgr != nil {
		snapshots := h.monitorMgr.Snapshot()
		for _, snap := range snapshots {
			if !snap.Available || snap.Blacklisted {
				continue
			}
			region := snap.Region
			if region == "" {
				region = "unknown"
			}

			node := map[string]any{
				"name":          snap.Name,
				"latency":       snap.LastLatencyMs,
				"region_name":   snap.RegionName,
			}

			if result[region] == nil {
				result[region] = []map[string]any{}
			}
			result[region] = append(result[region], node)
		}
	}

	writePoolJSON(w, map[string]any{"groups": result})
}

// handleGroupsBySubscription returns nodes grouped by subscription
func (h *ProxyPoolHandler) handleGroupsBySubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subs := h.store.ListSubscriptions()
	result := make(map[string]any)

	for _, sub := range subs {
		nodes := h.store.ListNodesBySubscription(sub.ID)
		nodeList := make([]map[string]any, 0, len(nodes))
		for _, node := range nodes {
			nodeList = append(nodeList, map[string]any{
				"name":          node.Name,
				"region":        node.Region,
				"latency":       node.Latency,
				"latency_level": node.LatencyLevel,
			})
		}
		result[sub.Name] = map[string]any{
			"id":         sub.ID,
			"url":        sub.URL,
			"enabled":    sub.Enabled,
			"node_count": len(nodes),
			"nodes":      nodeList,
		}
	}

	writePoolJSON(w, map[string]any{"groups": result})
}

func writePoolJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}
