package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Subscription represents a proxy subscription source
type Subscription struct {
	ID              string    `json:"id" yaml:"id"`
	Name            string    `json:"name" yaml:"name"`
	URL             string    `json:"url" yaml:"url"`
	Enabled         bool      `json:"enabled" yaml:"enabled"`
	RefreshInterval string    `json:"refresh_interval" yaml:"refresh_interval"` // e.g., "1h", "30m"
	LastRefreshAt   time.Time `json:"last_refresh_at" yaml:"-"`
	NodeCount       int       `json:"node_count" yaml:"-"`
	LastError       string    `json:"last_error,omitempty" yaml:"-"`
	CreatedAt       time.Time `json:"created_at" yaml:"-"`
	UpdatedAt       time.Time `json:"updated_at" yaml:"-"`
}

// NodeStatus represents the status of a node
type NodeStatus string

const (
	NodeStatusEnabled     NodeStatus = "enabled"
	NodeStatusDisabled    NodeStatus = "disabled"
	NodeStatusBlacklisted NodeStatus = "blacklisted"
)

// LatencyLevel represents the latency category
type LatencyLevel string

const (
	LatencyLevelLow     LatencyLevel = "low"     // ≤100ms
	LatencyLevelMedium  LatencyLevel = "medium"  // 100-300ms
	LatencyLevelHigh    LatencyLevel = "high"    // >300ms
	LatencyLevelUnknown LatencyLevel = "unknown" // not tested
)

// EnhancedNode extends basic node config with additional metadata
type EnhancedNode struct {
	Name             string       `json:"name"`
	URI              string       `json:"uri"`
	Port             uint16       `json:"port,omitempty"`
	SubscriptionID   string       `json:"subscription_id,omitempty"`
	SubscriptionName string       `json:"subscription_name,omitempty"`
	Region           string       `json:"region,omitempty"`      // e.g., "US", "JP", "HK"
	RegionName       string       `json:"region_name,omitempty"` // e.g., "United States", "Japan"
	Latency          int64        `json:"latency"`               // in milliseconds, -1 if unknown
	LatencyLevel     LatencyLevel `json:"latency_level"`
	Status           NodeStatus   `json:"status"`
	Available        bool         `json:"available"`
	LastCheckAt      time.Time    `json:"last_check_at,omitempty"`
	FailureCount     int          `json:"failure_count"`
	SuccessCount     int64        `json:"success_count"`
}

// LatencyConfig defines thresholds for latency grouping
type LatencyConfig struct {
	LowThreshold    int64 `json:"low_threshold" yaml:"low_threshold"`       // ≤100ms default
	MediumThreshold int64 `json:"medium_threshold" yaml:"medium_threshold"` // ≤300ms default
}

// PoolMode defines proxy rotation strategy
type PoolMode string

const (
	PoolModeSequential   PoolMode = "sequential"
	PoolModeRandom       PoolMode = "random"
	PoolModeLatencyFirst PoolMode = "latency_first"
	PoolModeWeighted     PoolMode = "weighted"
)

// Store handles persistent storage of subscriptions and node states
type Store struct {
	mu            sync.RWMutex
	dataDir       string
	subscriptions map[string]*Subscription
	nodeStates    map[string]*EnhancedNode // key: node name or URI hash
	latencyConfig LatencyConfig
}

// NewStore creates a new store instance
func NewStore(dataDir string) (*Store, error) {
	s := &Store{
		dataDir:       dataDir,
		subscriptions: make(map[string]*Subscription),
		nodeStates:    make(map[string]*EnhancedNode),
		latencyConfig: LatencyConfig{
			LowThreshold:    100,
			MediumThreshold: 300,
		},
	}

	// Create data directory if not exists
	if dataDir != "" {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, err
		}
		// Load existing data
		_ = s.load()
	}

	return s, nil
}

// SetLatencyConfig updates the latency thresholds
func (s *Store) SetLatencyConfig(cfg LatencyConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencyConfig = cfg
}

// GetLatencyConfig returns current latency thresholds
func (s *Store) GetLatencyConfig() LatencyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latencyConfig
}

// CalculateLatencyLevel determines the latency level based on thresholds
func (s *Store) CalculateLatencyLevel(latencyMs int64) LatencyLevel {
	s.mu.RLock()
	cfg := s.latencyConfig
	s.mu.RUnlock()

	if latencyMs < 0 {
		return LatencyLevelUnknown
	}
	if latencyMs <= cfg.LowThreshold {
		return LatencyLevelLow
	}
	if latencyMs <= cfg.MediumThreshold {
		return LatencyLevelMedium
	}
	return LatencyLevelHigh
}

// --- Subscription Methods ---

// AddSubscription adds a new subscription
func (s *Store) AddSubscription(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub.ID == "" {
		sub.ID = generateID()
	}
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()
	if sub.RefreshInterval == "" {
		sub.RefreshInterval = "1h"
	}
	s.subscriptions[sub.ID] = sub
	return s.save()
}

// UpdateSubscription updates an existing subscription
func (s *Store) UpdateSubscription(sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[sub.ID]; !exists {
		return ErrSubscriptionNotFound
	}
	sub.UpdatedAt = time.Now()
	s.subscriptions[sub.ID] = sub
	return s.save()
}

// DeleteSubscription removes a subscription
func (s *Store) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[id]; !exists {
		return ErrSubscriptionNotFound
	}
	delete(s.subscriptions, id)
	return s.save()
}

// GetSubscription retrieves a subscription by ID
func (s *Store) GetSubscription(id string) (*Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, exists := s.subscriptions[id]
	if !exists {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

// ListSubscriptions returns all subscriptions
func (s *Store) ListSubscriptions() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Subscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		result = append(result, sub)
	}
	return result
}

// --- Node State Methods ---

// UpdateNodeState updates or creates a node state
func (s *Store) UpdateNodeState(node *EnhancedNode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := node.Name
	if key == "" {
		key = node.URI
	}
	s.nodeStates[key] = node
}

// GetNodeState retrieves a node state
func (s *Store) GetNodeState(name string) (*EnhancedNode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodeStates[name]
	return node, exists
}

// SetNodeStatus updates the status of a node
func (s *Store) SetNodeStatus(name string, status NodeStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodeStates[name]
	if !exists {
		return ErrNodeNotFound
	}
	node.Status = status
	return s.save()
}

// ListNodesByLatency returns nodes filtered by latency level
func (s *Store) ListNodesByLatency(level LatencyLevel) []*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*EnhancedNode
	for _, node := range s.nodeStates {
		if node.LatencyLevel == level && node.Status == NodeStatusEnabled && node.Available {
			result = append(result, node)
		}
	}
	return result
}

// ListNodesByRegion returns nodes filtered by region
func (s *Store) ListNodesByRegion(region string) []*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*EnhancedNode
	for _, node := range s.nodeStates {
		if node.Region == region && node.Status == NodeStatusEnabled && node.Available {
			result = append(result, node)
		}
	}
	return result
}

// ListNodesBySubscription returns nodes filtered by subscription
func (s *Store) ListNodesBySubscription(subID string) []*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*EnhancedNode
	for _, node := range s.nodeStates {
		if node.SubscriptionID == subID && node.Status == NodeStatusEnabled {
			result = append(result, node)
		}
	}
	return result
}

// ListAvailableNodes returns all enabled and available nodes
func (s *Store) ListAvailableNodes() []*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*EnhancedNode
	for _, node := range s.nodeStates {
		if node.Status == NodeStatusEnabled && node.Available {
			result = append(result, node)
		}
	}
	return result
}

// GetGroupedByLatency returns nodes grouped by latency level
func (s *Store) GetGroupedByLatency() map[LatencyLevel][]*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[LatencyLevel][]*EnhancedNode)
	for _, node := range s.nodeStates {
		if node.Status == NodeStatusEnabled && node.Available {
			result[node.LatencyLevel] = append(result[node.LatencyLevel], node)
		}
	}
	return result
}

// GetGroupedByRegion returns nodes grouped by region
func (s *Store) GetGroupedByRegion() map[string][]*EnhancedNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]*EnhancedNode)
	for _, node := range s.nodeStates {
		if node.Status == NodeStatusEnabled && node.Available {
			region := node.Region
			if region == "" {
				region = "unknown"
			}
			result[region] = append(result[region], node)
		}
	}
	return result
}

// --- Persistence ---

type storeData struct {
	Subscriptions map[string]*Subscription `json:"subscriptions"`
	NodeStates    map[string]*EnhancedNode `json:"node_states"`
	LatencyConfig LatencyConfig            `json:"latency_config"`
}

func (s *Store) save() error {
	if s.dataDir == "" {
		return nil
	}

	data := storeData{
		Subscriptions: s.subscriptions,
		NodeStates:    s.nodeStates,
		LatencyConfig: s.latencyConfig,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(s.dataDir, "store.json"), bytes, 0644)
}

func (s *Store) load() error {
	path := filepath.Join(s.dataDir, "store.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var data storeData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	if data.Subscriptions != nil {
		s.subscriptions = data.Subscriptions
	}
	if data.NodeStates != nil {
		s.nodeStates = data.NodeStates
	}
	if data.LatencyConfig.LowThreshold > 0 {
		s.latencyConfig = data.LatencyConfig
	}

	return nil
}

// Save persists all data to disk
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// --- Errors ---

var (
	ErrSubscriptionNotFound = &StoreError{Message: "subscription not found"}
	ErrNodeNotFound         = &StoreError{Message: "node not found"}
)

type StoreError struct {
	Message string
}

func (e *StoreError) Error() string {
	return e.Message
}

// --- Helpers ---

func generateID() string {
	return time.Now().Format("20060102150405") + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
