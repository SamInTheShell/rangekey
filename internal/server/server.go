package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"github.com/samintheshell/rangekey/client"
	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/storage"
	"github.com/samintheshell/rangekey/internal/raft"
	"github.com/samintheshell/rangekey/internal/grpc"
	"github.com/samintheshell/rangekey/internal/metadata"
	"github.com/samintheshell/rangekey/internal/partition"
	"github.com/samintheshell/rangekey/internal/transaction"
)

// Server represents the RangeDB server
type Server struct {
	config *config.ServerConfig

	// Core components
	storage      *storage.Engine
	raftNode     *raft.Node
	grpcServer   *grpc.Server
	metadata     *metadata.Store
	partitions   *partition.Manager
	transactions *transaction.Manager

	// Lifecycle management
	mu       sync.RWMutex
	started  bool
	stopping bool
	stopCh   chan struct{}

	// Background workers
	workers sync.WaitGroup
}

// NewServer creates a new RangeDB server
func NewServer(cfg *config.ServerConfig) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("server config is required")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize storage engine
	storageEngine, err := storage.NewEngine(&storage.Config{
		DataDir:         cfg.GetDataPath("storage"),
		WALDir:          cfg.GetDataPath("wal"),
		MaxBatchSize:    cfg.MaxBatchSize,
		FlushInterval:   cfg.FlushInterval,
		CompactionLevel: cfg.CompactionLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage engine: %w", err)
	}

	// Initialize metadata store
	metadataStore, err := metadata.NewStore(&metadata.Config{
		DataDir: cfg.GetDataPath("metadata"),
		NodeID:  cfg.GetNodeID(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metadata store: %w", err)
	}

	// Initialize partition manager
	partitionManager := partition.NewManager(&partition.Config{
		NodeID:            cfg.GetNodeID(),
		ReplicationFactor: cfg.ReplicationFactor,
		Storage:           storageEngine,
		Metadata:          metadataStore,
	})

	// Build peer list - for joining nodes, convert client addresses to peer addresses
	var peers []string
	if len(cfg.JoinAddresses) > 0 {
		// Convert client addresses (port 8081) to peer addresses (port 8080)
		for _, clientAddr := range cfg.JoinAddresses {
			host, _, err := net.SplitHostPort(clientAddr)
			if err != nil {
				return nil, fmt.Errorf("invalid join address %s: %w", clientAddr, err)
			}
			peerAddr := fmt.Sprintf("%s:8080", host)
			peers = append(peers, peerAddr)
		}
	} else {
		peers = cfg.Peers
	}

	// Initialize Raft node
	raftNode, err := raft.NewNode(&raft.Config{
		NodeID:           cfg.GetNodeID(),
		ListenAddr:       cfg.GetRaftAddress(),
		DataDir:          cfg.GetDataPath("raft"),
		HeartbeatTimeout: cfg.HeartbeatTimeout,
		ElectionTimeout:  cfg.ElectionTimeout,
		Storage:          storageEngine,
		Metadata:         metadataStore,
		Partitions:       partitionManager,
		Peers:            peers,
		Join:             len(cfg.JoinAddresses) > 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Raft node: %w", err)
	}

	// Initialize transaction manager
	transactionManager, err := transaction.NewManager(&transaction.Config{
		DefaultTimeout:  cfg.RequestTimeout,
		MaxTransactions: 1000,
		Storage:        storageEngine,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transaction manager: %w", err)
	}

	// Initialize gRPC server
	grpcServer, err := grpc.NewServer(&grpc.Config{
		ListenAddr:     cfg.ClientAddress,
		RequestTimeout: cfg.RequestTimeout,
		Storage:        storageEngine,
		Raft:           raftNode,
		Metadata:       metadataStore,
		Partitions:     partitionManager,
		Transactions:   transactionManager,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	server := &Server{
		config:       cfg,
		storage:      storageEngine,
		raftNode:     raftNode,
		grpcServer:   grpcServer,
		metadata:     metadataStore,
		partitions:   partitionManager,
		transactions: transactionManager,
		stopCh:       make(chan struct{}),
	}

	return server, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server is already started")
	}

	log.Printf("Starting RangeDB server (Node ID: %s)", s.config.GetNodeID())

	// Start storage engine
	if err := s.storage.Start(ctx); err != nil {
		return fmt.Errorf("failed to start storage engine: %w", err)
	}

	// Start metadata store
	if err := s.metadata.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metadata store: %w", err)
	}

	// Initialize cluster if needed
	if s.config.IsBootstrap() {
		log.Println("Initializing new cluster...")
		if err := s.initializeCluster(ctx); err != nil {
			return fmt.Errorf("failed to initialize cluster: %w", err)
		}
	} else if s.config.IsJoinMode() {
		log.Println("Joining existing cluster...")
		if err := s.joinCluster(ctx); err != nil {
			return fmt.Errorf("failed to join cluster: %w", err)
		}
	} else {
		log.Println("Starting as standalone node...")
		if err := s.startStandalone(ctx); err != nil {
			return fmt.Errorf("failed to start standalone: %w", err)
		}
	}

	// Start Raft node
	if err := s.raftNode.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Raft node: %w", err)
	}

	// Start partition manager
	if err := s.partitions.Start(ctx); err != nil {
		return fmt.Errorf("failed to start partition manager: %w", err)
	}

	// Start transaction manager
	if err := s.transactions.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transaction manager: %w", err)
	}

	// Start gRPC server
	if err := s.grpcServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Start background workers
	s.startBackgroundWorkers(ctx)

	s.started = true
	log.Printf("RangeDB server started successfully")
	log.Printf("  - Client address: %s", s.config.ClientAddress)
	log.Printf("  - Peer address: %s", s.config.PeerAddress)
	log.Printf("  - Raft address: %s", s.config.GetRaftAddress())
	log.Printf("  - Data directory: %s", s.config.DataDir)

	return nil
}

// Stop stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.stopping {
		return nil
	}

	s.stopping = true
	close(s.stopCh)

	log.Println("Stopping RangeDB server...")

	// Stop background workers
	s.workers.Wait()

	// Stop gRPC server
	if err := s.grpcServer.Stop(ctx); err != nil {
		log.Printf("Error stopping gRPC server: %v", err)
	}

	// Stop partition manager
	if err := s.partitions.Stop(ctx); err != nil {
		log.Printf("Error stopping partition manager: %v", err)
	}

	// Stop transaction manager
	if err := s.transactions.Stop(ctx); err != nil {
		log.Printf("Error stopping transaction manager: %v", err)
	}

	// Stop Raft node
	if err := s.raftNode.Stop(ctx); err != nil {
		log.Printf("Error stopping Raft node: %v", err)
	}

	// Stop metadata store
	if err := s.metadata.Stop(ctx); err != nil {
		log.Printf("Error stopping metadata store: %v", err)
	}

	// Stop storage engine
	if err := s.storage.Stop(ctx); err != nil {
		log.Printf("Error stopping storage engine: %v", err)
	}

	s.started = false
	s.stopping = false

	log.Println("RangeDB server stopped")
	return nil
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started && !s.stopping
}

// initializeCluster initializes a new cluster
func (s *Server) initializeCluster(ctx context.Context) error {
	log.Println("Initializing cluster metadata...")

	// Create cluster configuration
	clusterConfig := &metadata.ClusterConfig{
		ID:                "rangedb-cluster",
		ReplicationFactor: s.config.ReplicationFactor,
		Nodes: []metadata.NodeInfo{
			{
				ID:            s.config.GetNodeID(),
				PeerAddress:   s.config.PeerAddress,
				ClientAddress: s.config.ClientAddress,
				RaftAddress:   s.config.GetRaftAddress(),
				Status:        metadata.NodeStatusActive,
				JoinedAt:      time.Now(),
			},
		},
		CreatedAt: time.Now(),
	}

	// Store cluster configuration
	if err := s.metadata.StoreClusterConfig(ctx, clusterConfig); err != nil {
		return fmt.Errorf("failed to store cluster config: %w", err)
	}

	// Initialize initial partitions
	if err := s.partitions.InitializePartitions(ctx); err != nil {
		return fmt.Errorf("failed to initialize partitions: %w", err)
	}

	log.Println("Cluster initialized successfully")
	return nil
}

// joinCluster joins an existing cluster
func (s *Server) joinCluster(ctx context.Context) error {
	log.Printf("Joining cluster via: %v", s.config.GetJoinList())

	joinAddresses := s.config.GetJoinList()
	if len(joinAddresses) == 0 {
		return fmt.Errorf("no join addresses provided")
	}

	// Try to contact each join address until one succeeds
	for _, address := range joinAddresses {
		err := s.attemptJoinCluster(ctx, address)
		if err == nil {
			log.Printf("Successfully joined cluster via %s", address)
			return nil
		}
		log.Printf("Failed to join cluster via %s: %v", address, err)
	}

	return fmt.Errorf("failed to join cluster via any of the provided addresses")
}

// attemptJoinCluster attempts to join a cluster via a specific address
func (s *Server) attemptJoinCluster(ctx context.Context, address string) error {
	// Import client package
	clientConfig := &client.Config{
		Address:        address,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
	}

	// Create client
	c, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Connect to the cluster node
	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to cluster node: %w", err)
	}
	defer c.Close()

	// Request to join the cluster
	joinResponse, err := c.JoinCluster(ctx, &v1.JoinClusterRequest{
		NodeId:        s.config.GetNodeID(),
		PeerAddress:   s.config.PeerAddress,
		ClientAddress: s.config.ClientAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to request cluster join: %w", err)
	}

	if !joinResponse.Success {
		return fmt.Errorf("cluster join rejected: %s", joinResponse.Message)
	}

	// Store the cluster configuration we received
	clusterConfig := &metadata.ClusterConfig{
		ID:        joinResponse.ClusterId,
		Nodes:     make([]metadata.NodeInfo, len(joinResponse.Nodes)),
		UpdatedAt: time.Now(),
	}

	for i, node := range joinResponse.Nodes {
		clusterConfig.Nodes[i] = metadata.NodeInfo{
			ID:            node.NodeId,
			PeerAddress:   node.PeerAddress,
			ClientAddress: node.ClientAddress,
			Status:        metadata.NodeStatusActive,
			JoinedAt:      time.Now(),
		}
	}

	// Store the cluster configuration
	if err := s.metadata.StoreClusterConfig(ctx, clusterConfig); err != nil {
		return fmt.Errorf("failed to store cluster config: %w", err)
	}

	log.Printf("Successfully joined cluster %s with %d nodes", joinResponse.ClusterId, len(joinResponse.Nodes))
	return nil
}

// startStandalone starts as a standalone node
func (s *Server) startStandalone(ctx context.Context) error {
	log.Println("Starting as standalone node...")

	// Check if cluster configuration exists
	clusterConfig, err := s.metadata.GetClusterConfig(ctx)
	if err != nil {
		// If no cluster config exists, create a single-node cluster
		return s.initializeCluster(ctx)
	}

	// Update node status
	nodeInfo := metadata.NodeInfo{
		ID:            s.config.GetNodeID(),
		PeerAddress:   s.config.PeerAddress,
		ClientAddress: s.config.ClientAddress,
		RaftAddress:   s.config.GetRaftAddress(),
		Status:        metadata.NodeStatusActive,
		LastSeen:      time.Now(),
	}

	if err := s.metadata.UpdateNodeInfo(ctx, nodeInfo); err != nil {
		return fmt.Errorf("failed to update node info: %w", err)
	}

	log.Printf("Rejoined cluster: %s", clusterConfig.ID)
	return nil
}

// startBackgroundWorkers starts background maintenance workers
func (s *Server) startBackgroundWorkers(ctx context.Context) {
	// Health check worker
	s.workers.Add(1)
	go s.healthCheckWorker(ctx)

	// Metrics collection worker
	s.workers.Add(1)
	go s.metricsWorker(ctx)

	// Partition rebalancing worker
	s.workers.Add(1)
	go s.rebalanceWorker(ctx)
}

// healthCheckWorker performs periodic health checks
func (s *Server) healthCheckWorker(ctx context.Context) {
	defer s.workers.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.performHealthCheck(ctx)
		}
	}
}

// metricsWorker collects and reports metrics
func (s *Server) metricsWorker(ctx context.Context) {
	defer s.workers.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.collectMetrics(ctx)
		}
	}
}

// rebalanceWorker performs automatic partition rebalancing
func (s *Server) rebalanceWorker(ctx context.Context) {
	defer s.workers.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkRebalancing(ctx)
		}
	}
}

// performHealthCheck performs a health check
func (s *Server) performHealthCheck(ctx context.Context) {
	// Update node last seen time
	nodeInfo := metadata.NodeInfo{
		ID:       s.config.GetNodeID(),
		Status:   metadata.NodeStatusActive,
		LastSeen: time.Now(),
	}

	if err := s.metadata.UpdateNodeInfo(ctx, nodeInfo); err != nil {
		log.Printf("Failed to update node health: %v", err)
	}

	// Check storage health
	if err := s.storage.HealthCheck(ctx); err != nil {
		log.Printf("Storage health check failed: %v", err)
	}

	// Check Raft health
	if err := s.raftNode.HealthCheck(ctx); err != nil {
		log.Printf("Raft health check failed: %v", err)
	}
}

// collectMetrics collects and reports metrics
func (s *Server) collectMetrics(ctx context.Context) {
	// TODO: Implement metrics collection
	// This would include:
	// - Storage metrics (size, operations/sec)
	// - Raft metrics (leader status, log size)
	// - Network metrics (request latency, throughput)
	// - Partition metrics (distribution, migration status)
}

// checkRebalancing checks if partition rebalancing is needed
func (s *Server) checkRebalancing(ctx context.Context) {
	log.Println("Checking if partition rebalancing is needed...")

	// Calculate current partition loads
	loads, err := s.partitions.CalculatePartitionLoad(ctx)
	if err != nil {
		log.Printf("Failed to calculate partition loads: %v", err)
		return
	}

	// Check if any partition is overloaded
	needsRebalancing := false
	for partitionID, load := range loads {
		if load > 2.0 { // Threshold for rebalancing
			log.Printf("Partition %s is overloaded (load: %.2f)", partitionID, load)
			needsRebalancing = true
		}
	}

	// Trigger rebalancing if needed
	if needsRebalancing {
		log.Println("Triggering partition rebalancing...")
		if err := s.partitions.RebalancePartitions(ctx); err != nil {
			log.Printf("Failed to rebalance partitions: %v", err)
		}
	}
}

// IsLeader returns true if this node is the Raft leader
func (s *Server) IsLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.raftNode == nil {
		return false
	}

	return s.raftNode.IsLeader()
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *config.ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config
}

// Backup creates a backup of the server data
func (s *Server) Backup(ctx context.Context, backupPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return fmt.Errorf("server is not started")
	}

	log.Printf("Starting backup to %s", backupPath)

	// Create backup using storage engine
	if err := s.storage.Backup(ctx, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	log.Printf("Backup completed successfully")
	return nil
}

// Restore restores the server from a backup
func (s *Server) Restore(ctx context.Context, backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("cannot restore while server is running - stop server first")
	}

	log.Printf("Starting restore from %s", backupPath)

	// Restore using storage engine
	if err := s.storage.Restore(ctx, backupPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	log.Printf("Restore completed successfully")
	return nil
}

// GetBackupMetadata returns metadata about a backup
func (s *Server) GetBackupMetadata(backupPath string) (*storage.BackupMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.storage.GetBackupMetadata(backupPath)
}
