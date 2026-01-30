package proxypool

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"easy_proxies/internal/store"
)

// ProxyPool manages proxy rotation with multiple strategies
type ProxyPool struct {
	mu           sync.RWMutex
	store        *store.Store
	mode         store.PoolMode
	nodes        []*store.EnhancedNode
	currentIndex atomic.Int64
	
	// Weighted mode state
	weights      map[string]int // node name -> weight
	totalWeight  int

	// Settings
	fallbackEnabled bool
	apiKey          string
}

// Config for proxy pool
type Config struct {
	Mode            store.PoolMode
	FallbackEnabled bool
	APIKey          string
}

// NewProxyPool creates a new proxy pool
func NewProxyPool(s *store.Store, cfg Config) *ProxyPool {
	if cfg.Mode == "" {
		cfg.Mode = store.PoolModeSequential
	}

	return &ProxyPool{
		store:           s,
		mode:            cfg.Mode,
		weights:         make(map[string]int),
		fallbackEnabled: cfg.FallbackEnabled,
		apiKey:          cfg.APIKey,
	}
}

// SetMode changes the rotation mode
func (p *ProxyPool) SetMode(mode store.PoolMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mode = mode
}

// GetMode returns current rotation mode
func (p *ProxyPool) GetMode() store.PoolMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// RefreshNodes updates the internal node list from store
func (p *ProxyPool) RefreshNodes() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nodes = p.store.ListAvailableNodes()
	p.updateWeights()
}

// updateWeights recalculates weights based on latency
func (p *ProxyPool) updateWeights() {
	p.weights = make(map[string]int)
	p.totalWeight = 0

	for _, node := range p.nodes {
		weight := calculateWeight(node.Latency)
		p.weights[node.Name] = weight
		p.totalWeight += weight
	}
}

// calculateWeight returns weight based on latency (lower latency = higher weight)
func calculateWeight(latencyMs int64) int {
	if latencyMs < 0 {
		return 1 // Unknown latency gets minimal weight
	}
	if latencyMs <= 50 {
		return 100
	}
	if latencyMs <= 100 {
		return 80
	}
	if latencyMs <= 200 {
		return 50
	}
	if latencyMs <= 300 {
		return 30
	}
	if latencyMs <= 500 {
		return 15
	}
	return 5
}

// Filter options for GetProxy
type Filter struct {
	LatencyLevel store.LatencyLevel
	Region       string
	Subscription string
}

// GetProxy returns a proxy based on current mode and optional filters
func (p *ProxyPool) GetProxy(filter *Filter) (*store.EnhancedNode, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get filtered nodes
	nodes := p.getFilteredNodes(filter)
	if len(nodes) == 0 {
		// Try fallback if enabled
		if p.fallbackEnabled && filter != nil {
			// Try with relaxed filter
			nodes = p.getFallbackNodes(filter)
		}
		if len(nodes) == 0 {
			return nil, ErrNoAvailableProxy
		}
	}

	// Select based on mode
	switch p.mode {
	case store.PoolModeRandom:
		return p.selectRandom(nodes), nil
	case store.PoolModeLatencyFirst:
		return p.selectLatencyFirst(nodes), nil
	case store.PoolModeWeighted:
		return p.selectWeighted(nodes), nil
	default: // Sequential
		return p.selectSequential(nodes), nil
	}
}

// GetProxyList returns multiple proxies based on filters
func (p *ProxyPool) GetProxyList(filter *Filter, limit int) []*store.EnhancedNode {
	p.mu.RLock()
	defer p.mu.RUnlock()

	nodes := p.getFilteredNodes(filter)
	if limit > 0 && len(nodes) > limit {
		nodes = nodes[:limit]
	}
	return nodes
}

// getFilteredNodes returns nodes matching the filter
func (p *ProxyPool) getFilteredNodes(filter *Filter) []*store.EnhancedNode {
	if filter == nil {
		return p.nodes
	}

	var result []*store.EnhancedNode
	for _, node := range p.nodes {
		if filter.LatencyLevel != "" && node.LatencyLevel != filter.LatencyLevel {
			continue
		}
		if filter.Region != "" && node.Region != filter.Region {
			continue
		}
		if filter.Subscription != "" && node.SubscriptionID != filter.Subscription && node.SubscriptionName != filter.Subscription {
			continue
		}
		result = append(result, node)
	}
	return result
}

// getFallbackNodes returns nodes with relaxed filters for fallback
func (p *ProxyPool) getFallbackNodes(filter *Filter) []*store.EnhancedNode {
	// If filtering by low latency, try medium
	if filter.LatencyLevel == store.LatencyLevelLow {
		relaxed := *filter
		relaxed.LatencyLevel = store.LatencyLevelMedium
		nodes := p.getFilteredNodes(&relaxed)
		if len(nodes) > 0 {
			return nodes
		}
		// Try high latency
		relaxed.LatencyLevel = store.LatencyLevelHigh
		return p.getFilteredNodes(&relaxed)
	}

	// If filtering by medium, try high
	if filter.LatencyLevel == store.LatencyLevelMedium {
		relaxed := *filter
		relaxed.LatencyLevel = store.LatencyLevelHigh
		return p.getFilteredNodes(&relaxed)
	}

	// Try without latency filter
	relaxed := *filter
	relaxed.LatencyLevel = ""
	return p.getFilteredNodes(&relaxed)
}

// selectSequential returns the next proxy in sequence
func (p *ProxyPool) selectSequential(nodes []*store.EnhancedNode) *store.EnhancedNode {
	if len(nodes) == 0 {
		return nil
	}
	idx := p.currentIndex.Add(1) - 1
	return nodes[idx%int64(len(nodes))]
}

// selectRandom returns a random proxy
func (p *ProxyPool) selectRandom(nodes []*store.EnhancedNode) *store.EnhancedNode {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[rand.Intn(len(nodes))]
}

// selectLatencyFirst returns the proxy with lowest latency
func (p *ProxyPool) selectLatencyFirst(nodes []*store.EnhancedNode) *store.EnhancedNode {
	if len(nodes) == 0 {
		return nil
	}
	
	// Sort by latency
	sorted := make([]*store.EnhancedNode, len(nodes))
	copy(sorted, nodes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Latency < 0 {
			return false
		}
		if sorted[j].Latency < 0 {
			return true
		}
		return sorted[i].Latency < sorted[j].Latency
	})

	// Pick from top 3 lowest latency nodes randomly to avoid overloading one node
	topN := 3
	if len(sorted) < topN {
		topN = len(sorted)
	}
	return sorted[rand.Intn(topN)]
}

// selectWeighted returns a proxy based on weight (lower latency = higher chance)
func (p *ProxyPool) selectWeighted(nodes []*store.EnhancedNode) *store.EnhancedNode {
	if len(nodes) == 0 {
		return nil
	}

	// Calculate total weight for filtered nodes
	totalWeight := 0
	for _, node := range nodes {
		if weight, ok := p.weights[node.Name]; ok {
			totalWeight += weight
		} else {
			totalWeight += 1
		}
	}

	if totalWeight == 0 {
		return nodes[rand.Intn(len(nodes))]
	}

	// Weighted random selection
	r := rand.Intn(totalWeight)
	for _, node := range nodes {
		weight := 1
		if w, ok := p.weights[node.Name]; ok {
			weight = w
		}
		r -= weight
		if r < 0 {
			return node
		}
	}

	return nodes[0]
}

// ValidateAPIKey checks if the provided key is valid
func (p *ProxyPool) ValidateAPIKey(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.apiKey == "" {
		return true // No authentication required
	}
	return key == p.apiKey
}

// SetAPIKey updates the API key
func (p *ProxyPool) SetAPIKey(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apiKey = key
}

// Stats returns pool statistics
func (p *ProxyPool) Stats() PoolStats {
	// Refresh nodes first to ensure up-to-date data
	p.RefreshNodes()

	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalNodes:     len(p.nodes),
		Mode:           string(p.mode),
		ByLatency:      make(map[string]int),
		ByRegion:       make(map[string]int),
		BySubscription: make(map[string]int),
	}

	for _, node := range p.nodes {
		stats.ByLatency[string(node.LatencyLevel)]++
		if node.Region != "" {
			stats.ByRegion[node.Region]++
		} else {
			stats.ByRegion["unknown"]++
		}
		if node.SubscriptionName != "" {
			stats.BySubscription[node.SubscriptionName]++
		} else {
			stats.BySubscription["manual"]++
		}
	}

	return stats
}

// PoolStats contains pool statistics
type PoolStats struct {
	TotalNodes     int            `json:"total_nodes"`
	Mode           string         `json:"mode"`
	ByLatency      map[string]int `json:"by_latency"`
	ByRegion       map[string]int `json:"by_region"`
	BySubscription map[string]int `json:"by_subscription"`
}

// Errors
var (
	ErrNoAvailableProxy = errors.New("no available proxy")
	ErrUnauthorized     = errors.New("unauthorized: invalid API key")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
