package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Config holds the metadata store configuration
type Config struct {
	DataDir string
	NodeID  string
}

// Store represents the metadata store
type Store struct {
	config *Config
	db     *badger.DB
	mu     sync.RWMutex

	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// NodeStatus represents the status of a node
type NodeStatus int

const (
	NodeStatusActive NodeStatus = iota
	NodeStatusInactive
	NodeStatusLeaving
	NodeStatusJoining
)

// NodeInfo represents information about a cluster node
type NodeInfo struct {
	ID            string     `json:"id"`
	PeerAddress   string     `json:"peer_address"`
	ClientAddress string     `json:"client_address"`
	RaftAddress   string     `json:"raft_address"`
	Status        NodeStatus `json:"status"`
	JoinedAt      time.Time  `json:"joined_at"`
	LastSeen      time.Time  `json:"last_seen"`
}

// ClusterConfig represents the cluster configuration
type ClusterConfig struct {
	ID                string     `json:"id"`
	ReplicationFactor int        `json:"replication_factor"`
	Nodes             []NodeInfo `json:"nodes"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// PartitionInfo represents information about a partition
type PartitionInfo struct {
	ID           string    `json:"id"`
	StartKey     []byte    `json:"start_key"`
	EndKey       []byte    `json:"end_key"`
	RaftGroupID  string    `json:"raft_group_id"`
	Replicas     []string  `json:"replicas"`
	Leader       string    `json:"leader,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RaftGroupInfo represents information about a Raft group
type RaftGroupInfo struct {
	ID          string    `json:"id"`
	Members     []string  `json:"members"`
	Leader      string    `json:"leader,omitempty"`
	Term        uint64    `json:"term"`
	CommitIndex uint64    `json:"commit_index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Metadata key prefixes
const (
	MetadataPrefix         = "_/"
	ClusterConfigPrefix    = "_/cluster/config/"
	NodePrefix             = "_/cluster/nodes/"
	PartitionPrefix        = "_/partitions/"
	RaftGroupPrefix        = "_/raft/groups/"
	BackupPrefix           = "_/backups/"
)

// NewStore creates a new metadata store
func NewStore(config *Config) (*Store, error) {
	if config == nil {
		return nil, fmt.Errorf("metadata store config is required")
	}

	// Create data directory
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Initialize BadgerDB for metadata
	opts := badger.DefaultOptions(config.DataDir)
	opts.Logger = &metadataLogger{}
	opts.SyncWrites = true // Metadata needs to be durable
	opts.NumVersionsToKeep = 5 // Keep some versions for debugging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata database: %w", err)
	}

	store := &Store{
		config: config,
		db:     db,
		stopCh: make(chan struct{}),
	}

	return store, nil
}

// Start starts the metadata store
func (s *Store) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("metadata store is already started")
	}

	log.Println("Starting metadata store...")

	// Initialize metadata structure if needed
	if err := s.initializeMetadata(ctx); err != nil {
		return fmt.Errorf("failed to initialize metadata: %w", err)
	}

	s.started = true
	log.Println("Metadata store started successfully")

	return nil
}

// Stop stops the metadata store
func (s *Store) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	close(s.stopCh)

	log.Println("Stopping metadata store...")

	// Close database
	if err := s.db.Close(); err != nil {
		log.Printf("Error closing metadata database: %v", err)
	}

	s.started = false
	log.Println("Metadata store stopped")

	return nil
}

// GetNodeID returns the node ID from the store configuration
func (s *Store) GetNodeID() string {
	return s.config.NodeID
}

// StoreClusterConfig stores the cluster configuration
func (s *Store) StoreClusterConfig(ctx context.Context, config *ClusterConfig) error {
	if !s.started {
		return fmt.Errorf("metadata store is not started")
	}

	config.UpdatedAt = time.Now()

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal cluster config: %w", err)
	}

	key := ClusterConfigPrefix + "main"
	return s.put(key, data)
}

// GetClusterConfig retrieves the cluster configuration
func (s *Store) GetClusterConfig(ctx context.Context) (*ClusterConfig, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	key := ClusterConfigPrefix + "main"
	data, err := s.get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	var config ClusterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster config: %w", err)
	}

	return &config, nil
}

// UpdateNodeInfo updates node information
func (s *Store) UpdateNodeInfo(ctx context.Context, nodeInfo NodeInfo) error {
	if !s.started {
		return fmt.Errorf("metadata store is not started")
	}

	nodeInfo.LastSeen = time.Now()

	data, err := json.Marshal(nodeInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal node info: %w", err)
	}

	key := NodePrefix + nodeInfo.ID
	return s.put(key, data)
}

// GetNodeInfo retrieves node information
func (s *Store) GetNodeInfo(ctx context.Context, nodeID string) (*NodeInfo, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	key := NodePrefix + nodeID
	data, err := s.get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	var nodeInfo NodeInfo
	if err := json.Unmarshal(data, &nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node info: %w", err)
	}

	return &nodeInfo, nil
}

// ListNodes lists all nodes in the cluster
func (s *Store) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	var nodes []NodeInfo

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(NodePrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			var nodeInfo NodeInfo
			if err := json.Unmarshal(value, &nodeInfo); err != nil {
				return err
			}

			nodes = append(nodes, nodeInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return nodes, nil
}

// StorePartitionInfo stores partition information
func (s *Store) StorePartitionInfo(ctx context.Context, partitionInfo *PartitionInfo) error {
	if !s.started {
		return fmt.Errorf("metadata store is not started")
	}

	partitionInfo.UpdatedAt = time.Now()

	data, err := json.Marshal(partitionInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal partition info: %w", err)
	}

	key := PartitionPrefix + partitionInfo.ID
	return s.put(key, data)
}

// GetPartitionInfo retrieves partition information
func (s *Store) GetPartitionInfo(ctx context.Context, partitionID string) (*PartitionInfo, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	key := PartitionPrefix + partitionID
	data, err := s.get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get partition info: %w", err)
	}

	var partitionInfo PartitionInfo
	if err := json.Unmarshal(data, &partitionInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal partition info: %w", err)
	}

	return &partitionInfo, nil
}

// ListPartitions lists all partitions
func (s *Store) ListPartitions(ctx context.Context) ([]PartitionInfo, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	var partitions []PartitionInfo

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(PartitionPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			var partitionInfo PartitionInfo
			if err := json.Unmarshal(value, &partitionInfo); err != nil {
				return err
			}

			partitions = append(partitions, partitionInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list partitions: %w", err)
	}

	return partitions, nil
}

// DeletePartitionInfo deletes partition information
func (s *Store) DeletePartitionInfo(ctx context.Context, partitionID string) error {
	if !s.started {
		return fmt.Errorf("metadata store is not started")
	}

	key := fmt.Sprintf("%s%s", PartitionPrefix, partitionID)
	
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// StoreRaftGroupInfo stores Raft group information
func (s *Store) StoreRaftGroupInfo(ctx context.Context, raftGroupInfo *RaftGroupInfo) error {
	if !s.started {
		return fmt.Errorf("metadata store is not started")
	}

	raftGroupInfo.UpdatedAt = time.Now()

	data, err := json.Marshal(raftGroupInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal raft group info: %w", err)
	}

	key := RaftGroupPrefix + raftGroupInfo.ID
	return s.put(key, data)
}

// GetRaftGroupInfo retrieves Raft group information
func (s *Store) GetRaftGroupInfo(ctx context.Context, raftGroupID string) (*RaftGroupInfo, error) {
	if !s.started {
		return nil, fmt.Errorf("metadata store is not started")
	}

	key := RaftGroupPrefix + raftGroupID
	data, err := s.get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get raft group info: %w", err)
	}

	var raftGroupInfo RaftGroupInfo
	if err := json.Unmarshal(data, &raftGroupInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raft group info: %w", err)
	}

	return &raftGroupInfo, nil
}

// FindPartitionForKey finds the partition that contains the given key
func (s *Store) FindPartitionForKey(ctx context.Context, key []byte) (*PartitionInfo, error) {
	partitions, err := s.ListPartitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list partitions: %w", err)
	}

	for _, partition := range partitions {
		if s.keyInRange(key, partition.StartKey, partition.EndKey) {
			return &partition, nil
		}
	}

	return nil, fmt.Errorf("no partition found for key")
}

// keyInRange checks if a key is within the given range
func (s *Store) keyInRange(key, startKey, endKey []byte) bool {
	if len(startKey) == 0 && len(endKey) == 0 {
		return true // Full range
	}

	if len(startKey) > 0 && string(key) < string(startKey) {
		return false
	}

	if len(endKey) > 0 && string(key) >= string(endKey) {
		return false
	}

	return true
}

// initializeMetadata initializes the metadata structure
func (s *Store) initializeMetadata(ctx context.Context) error {
	// Check if metadata is already initialized
	_, err := s.GetClusterConfig(ctx)
	if err == nil {
		log.Println("Metadata already initialized")
		return nil
	}

	log.Println("Initializing metadata structure...")

	// The metadata directories are implicitly created when we store actual data
	// No need to create placeholder entries

	log.Println("Metadata structure initialized")
	return nil
}

// put stores a key-value pair in the metadata store
func (s *Store) put(key string, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// get retrieves a value from the metadata store
func (s *Store) get(key string) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, err
	}

	return value, nil
}

// delete removes a key from the metadata store
func (s *Store) delete(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// IsMetadataKey checks if a key is a metadata key
func IsMetadataKey(key string) bool {
	return strings.HasPrefix(key, MetadataPrefix)
}

// metadataLogger implements badger.Logger for metadata operations
type metadataLogger struct{}

func (l *metadataLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[METADATA ERROR] "+format, args...)
}

func (l *metadataLogger) Warningf(format string, args ...interface{}) {
	log.Printf("[METADATA WARN] "+format, args...)
}

func (l *metadataLogger) Infof(format string, args ...interface{}) {
	log.Printf("[METADATA INFO] "+format, args...)
}

func (l *metadataLogger) Debugf(format string, args ...interface{}) {
	// Skip debug messages
}
