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

	log.Printf("Splitting partition %s at key %s", partitionID, string(splitKey))

	// Get the current partition
	partition, exists := m.partitions[partitionID]
	if !exists {
		return fmt.Errorf("partition %s not found", partitionID)
	}

	// Validate that we can split this partition
	if !m.IsPartitionLeader(partitionID) {
		return fmt.Errorf("cannot split partition %s: not the leader", partitionID)
	}

	// Create new partition IDs
	leftPartitionID := partitionID + "-left"
	rightPartitionID := partitionID + "-right"

	// Create left partition (original start key to split key)
	leftPartition := &metadata.PartitionInfo{
		ID:          leftPartitionID,
		StartKey:    partition.StartKey,
		EndKey:      splitKey,
		RaftGroupID: leftPartitionID,
		Replicas:    partition.Replicas,
		Leader:      partition.Leader,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create right partition (split key to original end key)
	rightPartition := &metadata.PartitionInfo{
		ID:          rightPartitionID,
		StartKey:    splitKey,
		EndKey:      partition.EndKey,
		RaftGroupID: rightPartitionID,
		Replicas:    partition.Replicas,
		Leader:      partition.Leader,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store new partitions in metadata
	if err := m.config.Metadata.StorePartitionInfo(ctx, leftPartition); err != nil {
		return fmt.Errorf("failed to store left partition: %w", err)
	}

	if err := m.config.Metadata.StorePartitionInfo(ctx, rightPartition); err != nil {
		return fmt.Errorf("failed to store right partition: %w", err)
	}

	// Remove old partition
	if err := m.config.Metadata.DeletePartitionInfo(ctx, partitionID); err != nil {
		return fmt.Errorf("failed to delete old partition: %w", err)
	}

	// Update local partition cache
	delete(m.partitions, partitionID)
	m.partitions[leftPartitionID] = &PartitionInfo{
		ID:       leftPartitionID,
		StartKey: partition.StartKey,
		EndKey:   splitKey,
		Replicas: partition.Replicas,
		Leader:   partition.Leader,
		Status:   "active",
	}
	m.partitions[rightPartitionID] = &PartitionInfo{
		ID:       rightPartitionID,
		StartKey: splitKey,
		EndKey:   partition.EndKey,
		Replicas: partition.Replicas,
		Leader:   partition.Leader,
		Status:   "active",
	}

	log.Printf("Successfully split partition %s into %s and %s", partitionID, leftPartitionID, rightPartitionID)
	return nil
}

// MergePartitions merges two adjacent partitions
func (m *Manager) MergePartitions(ctx context.Context, leftPartitionID, rightPartitionID string) error {
	if !m.started {
		return fmt.Errorf("partition manager is not started")
	}

	log.Printf("Merging partitions %s and %s", leftPartitionID, rightPartitionID)

	// Get the partitions to merge
	leftPartition, leftExists := m.partitions[leftPartitionID]
	rightPartition, rightExists := m.partitions[rightPartitionID]

	if !leftExists || !rightExists {
		return fmt.Errorf("one or both partitions not found: %s, %s", leftPartitionID, rightPartitionID)
	}

	// Validate that we can merge these partitions
	if !m.IsPartitionLeader(leftPartitionID) || !m.IsPartitionLeader(rightPartitionID) {
		return fmt.Errorf("cannot merge partitions: not the leader of both partitions")
	}

	// Create merged partition ID
	mergedPartitionID := leftPartitionID + "-merged"

	// Create merged partition
	mergedPartition := &metadata.PartitionInfo{
		ID:          mergedPartitionID,
		StartKey:    leftPartition.StartKey,
		EndKey:      rightPartition.EndKey,
		RaftGroupID: mergedPartitionID,
		Replicas:    leftPartition.Replicas,
		Leader:      leftPartition.Leader,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store merged partition in metadata
	if err := m.config.Metadata.StorePartitionInfo(ctx, mergedPartition); err != nil {
		return fmt.Errorf("failed to store merged partition: %w", err)
	}

	// Remove old partitions
	if err := m.config.Metadata.DeletePartitionInfo(ctx, leftPartitionID); err != nil {
		return fmt.Errorf("failed to delete left partition: %w", err)
	}

	if err := m.config.Metadata.DeletePartitionInfo(ctx, rightPartitionID); err != nil {
		return fmt.Errorf("failed to delete right partition: %w", err)
	}

	// Update local partition cache
	delete(m.partitions, leftPartitionID)
	delete(m.partitions, rightPartitionID)
	m.partitions[mergedPartitionID] = &PartitionInfo{
		ID:       mergedPartitionID,
		StartKey: leftPartition.StartKey,
		EndKey:   rightPartition.EndKey,
		Replicas: leftPartition.Replicas,
		Leader:   leftPartition.Leader,
		Status:   "active",
	}

	log.Printf("Successfully merged partitions %s and %s into %s", leftPartitionID, rightPartitionID, mergedPartitionID)
	return nil
}

// RebalancePartitions rebalances partitions across nodes
func (m *Manager) RebalancePartitions(ctx context.Context) error {
	if !m.started {
		return fmt.Errorf("partition manager is not started")
	}

	log.Println("Starting partition rebalancing...")

	// Get cluster information
	clusterConfig, err := m.config.Metadata.GetClusterConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster config: %w", err)
	}

	// Get all partitions
	partitions, err := m.config.Metadata.ListPartitions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list partitions: %w", err)
	}

	// Calculate partition distribution
	nodePartitionCount := make(map[string]int)
	for _, node := range clusterConfig.Nodes {
		nodePartitionCount[node.ID] = 0
	}

	for _, partition := range partitions {
		for _, replica := range partition.Replicas {
			nodePartitionCount[replica]++
		}
	}

	// Calculate ideal distribution
	totalPartitions := len(partitions) * m.config.ReplicationFactor
	nodesCount := len(clusterConfig.Nodes)
	idealPartitionsPerNode := totalPartitions / nodesCount

	log.Printf("Rebalancing %d partitions across %d nodes (ideal: %d per node)", 
		len(partitions), nodesCount, idealPartitionsPerNode)

	// Find imbalanced nodes
	overloadedNodes := make([]string, 0)
	underloadedNodes := make([]string, 0)

	for nodeID, count := range nodePartitionCount {
		if count > idealPartitionsPerNode+1 {
			overloadedNodes = append(overloadedNodes, nodeID)
		} else if count < idealPartitionsPerNode-1 {
			underloadedNodes = append(underloadedNodes, nodeID)
		}
	}

	// Perform rebalancing moves
	moveCount := 0
	for len(overloadedNodes) > 0 && len(underloadedNodes) > 0 {
		overloadedNode := overloadedNodes[0]
		underloadedNode := underloadedNodes[0]

		// Find a partition to move
		var partitionToMove *metadata.PartitionInfo
		for _, partition := range partitions {
			for _, replica := range partition.Replicas {
				if replica == overloadedNode {
					partitionToMove = &partition
					break
				}
			}
			if partitionToMove != nil {
				break
			}
		}

		if partitionToMove != nil {
			// Move partition replica
			if err := m.movePartitionReplica(ctx, partitionToMove, overloadedNode, underloadedNode); err != nil {
				log.Printf("Failed to move partition %s from %s to %s: %v", 
					partitionToMove.ID, overloadedNode, underloadedNode, err)
			} else {
				moveCount++
				log.Printf("Moved partition %s replica from %s to %s", 
					partitionToMove.ID, overloadedNode, underloadedNode)
			}
		}

		// Update counts
		nodePartitionCount[overloadedNode]--
		nodePartitionCount[underloadedNode]++

		// Remove from lists if balanced
		if nodePartitionCount[overloadedNode] <= idealPartitionsPerNode+1 {
			overloadedNodes = overloadedNodes[1:]
		}
		if nodePartitionCount[underloadedNode] >= idealPartitionsPerNode-1 {
			underloadedNodes = underloadedNodes[1:]
		}
	}

	log.Printf("Rebalancing completed: %d partition moves", moveCount)
	return nil
}

// movePartitionReplica moves a partition replica from one node to another
func (m *Manager) movePartitionReplica(ctx context.Context, partition *metadata.PartitionInfo, fromNode, toNode string) error {
	// Update partition replicas
	newReplicas := make([]string, 0, len(partition.Replicas))
	for _, replica := range partition.Replicas {
		if replica == fromNode {
			newReplicas = append(newReplicas, toNode)
		} else {
			newReplicas = append(newReplicas, replica)
		}
	}

	// Update partition metadata
	updatedPartition := &metadata.PartitionInfo{
		ID:          partition.ID,
		StartKey:    partition.StartKey,
		EndKey:      partition.EndKey,
		RaftGroupID: partition.RaftGroupID,
		Replicas:    newReplicas,
		Leader:      partition.Leader,
		Status:      "migrating",
		CreatedAt:   partition.CreatedAt,
		UpdatedAt:   time.Now(),
	}

	if err := m.config.Metadata.StorePartitionInfo(ctx, updatedPartition); err != nil {
		return fmt.Errorf("failed to update partition metadata: %w", err)
	}

	// TODO: Implement actual data migration
	// This would involve:
	// 1. Copying data from source to destination
	// 2. Updating Raft group membership
	// 3. Waiting for replication to complete
	// 4. Updating status to "active"

	// For now, just mark as active
	updatedPartition.Status = "active"
	return m.config.Metadata.StorePartitionInfo(ctx, updatedPartition)
}

// CalculatePartitionLoad calculates the load for each partition
func (m *Manager) CalculatePartitionLoad(ctx context.Context) (map[string]float64, error) {
	loads := make(map[string]float64)
	
	// Get partition metrics from storage
	for partitionID := range m.partitions {
		// TODO: Implement proper load calculation
		// This would include:
		// - Request rate per partition
		// - Data size per partition
		// - CPU/memory usage per partition
		loads[partitionID] = 1.0 // Default load
	}
	
	return loads, nil
}
