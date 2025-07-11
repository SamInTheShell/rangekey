package partition

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samintheshell/rangekey/internal/storage"
	"github.com/samintheshell/rangekey/internal/metadata"
)

// Config holds the partition manager configuration
type Config struct {
	NodeID            string
	ReplicationFactor int
	Storage           *storage.Engine
	Metadata          *metadata.Store
}

// Manager manages partitions and their distribution
type Manager struct {
	config *Config

	// Partition state
	partitions map[string]*PartitionInfo

	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// PartitionInfo represents local partition information
type PartitionInfo struct {
	ID       string
	StartKey []byte
	EndKey   []byte
	Replicas []string
	Leader   string
	Status   string
}

// NewManager creates a new partition manager
func NewManager(config *Config) *Manager {
	return &Manager{
		config:     config,
		partitions: make(map[string]*PartitionInfo),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the partition manager
func (m *Manager) Start(ctx context.Context) error {
	if m.started {
		return fmt.Errorf("partition manager is already started")
	}

	log.Println("Starting partition manager...")

	// Load existing partitions
	if err := m.loadPartitions(ctx); err != nil {
		return fmt.Errorf("failed to load partitions: %w", err)
	}

	m.started = true
	log.Printf("Partition manager started with %d partitions", len(m.partitions))

	return nil
}

// Stop stops the partition manager
func (m *Manager) Stop(ctx context.Context) error {
	if !m.started {
		return nil
	}

	close(m.stopCh)

	log.Println("Stopping partition manager...")

	// TODO: Implement proper partition shutdown
	// This includes:
	// - Stopping partition workers
	// - Flushing partition state
	// - Coordinating with other nodes

	m.started = false
	log.Println("Partition manager stopped")

	return nil
}

// InitializePartitions initializes the initial partition layout
func (m *Manager) InitializePartitions(ctx context.Context) error {
	log.Println("Initializing initial partitions...")

	// Create initial partition covering the entire key space
	partition := &metadata.PartitionInfo{
		ID:          "partition-0",
		StartKey:    []byte(""),  // Empty start key means beginning of key space
		EndKey:      []byte(""),  // Empty end key means end of key space
		RaftGroupID: "raft-group-0",
		Replicas:    []string{m.config.NodeID},
		Leader:      m.config.NodeID,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store partition metadata
	if err := m.config.Metadata.StorePartitionInfo(ctx, partition); err != nil {
		return fmt.Errorf("failed to store initial partition: %w", err)
	}

	// Store Raft group metadata
	raftGroup := &metadata.RaftGroupInfo{
		ID:        "raft-group-0",
		Members:   []string{m.config.NodeID},
		Leader:    m.config.NodeID,
		Term:      1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.config.Metadata.StoreRaftGroupInfo(ctx, raftGroup); err != nil {
		return fmt.Errorf("failed to store initial raft group: %w", err)
	}

	log.Println("Initial partitions created successfully")
	return nil
}

// loadPartitions loads partition information from metadata
func (m *Manager) loadPartitions(ctx context.Context) error {
	partitions, err := m.config.Metadata.ListPartitions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list partitions: %w", err)
	}

	for _, partition := range partitions {
		// Convert metadata partition to local partition info
		localPartition := &PartitionInfo{
			ID:       partition.ID,
			StartKey: partition.StartKey,
			EndKey:   partition.EndKey,
			Replicas: partition.Replicas,
			Leader:   partition.Leader,
			Status:   partition.Status,
		}

		m.partitions[partition.ID] = localPartition
	}

	return nil
}

// FindPartitionForKey finds the partition that should contain the given key
func (m *Manager) FindPartitionForKey(ctx context.Context, key []byte) (*PartitionInfo, error) {
	if !m.started {
		return nil, fmt.Errorf("partition manager is not started")
	}

	// Use metadata store to find partition
	partitionInfo, err := m.config.Metadata.FindPartitionForKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find partition for key: %w", err)
	}

	// Convert to local partition info
	return &PartitionInfo{
		ID:       partitionInfo.ID,
		StartKey: partitionInfo.StartKey,
		EndKey:   partitionInfo.EndKey,
		Replicas: partitionInfo.Replicas,
		Leader:   partitionInfo.Leader,
		Status:   partitionInfo.Status,
	}, nil
}

// GetPartitions returns all partitions managed by this node
func (m *Manager) GetPartitions() map[string]*PartitionInfo {
	result := make(map[string]*PartitionInfo)
	for k, v := range m.partitions {
		result[k] = v
	}
	return result
}

// IsPartitionLeader checks if this node is the leader for the given partition
func (m *Manager) IsPartitionLeader(partitionID string) bool {
	partition, exists := m.partitions[partitionID]
	if !exists {
		return false
	}
	return partition.Leader == m.config.NodeID
}

// GetPartitionReplicas returns the replicas for a partition
func (m *Manager) GetPartitionReplicas(partitionID string) []string {
	partition, exists := m.partitions[partitionID]
	if !exists {
		return nil
	}
	return partition.Replicas
}

// SplitPartition splits a partition into two partitions
func (m *Manager) SplitPartition(ctx context.Context, partitionID string, splitKey []byte) error {
	if !m.started {
		return fmt.Errorf("partition manager is not started")
	}

	// TODO: Implement partition splitting
	// This would include:
	// 1. Validating the split operation
	// 2. Creating new partition metadata
	// 3. Coordinating with Raft groups
	// 4. Migrating data
	// 5. Updating routing tables

	return fmt.Errorf("partition splitting not implemented yet")
}

// MergePartitions merges two adjacent partitions
func (m *Manager) MergePartitions(ctx context.Context, leftPartitionID, rightPartitionID string) error {
	if !m.started {
		return fmt.Errorf("partition manager is not started")
	}

	// TODO: Implement partition merging
	// This would include:
	// 1. Validating the merge operation
	// 2. Migrating data from one partition to another
	// 3. Updating metadata
	// 4. Cleaning up old partition

	return fmt.Errorf("partition merging not implemented yet")
}

// RebalancePartitions rebalances partitions across nodes
func (m *Manager) RebalancePartitions(ctx context.Context) error {
	if !m.started {
		return fmt.Errorf("partition manager is not started")
	}

	// TODO: Implement partition rebalancing
	// This would include:
	// 1. Analyzing current distribution
	// 2. Calculating optimal distribution
	// 3. Planning migration steps
	// 4. Executing migrations
	// 5. Updating metadata

	return fmt.Errorf("partition rebalancing not implemented yet")
}
