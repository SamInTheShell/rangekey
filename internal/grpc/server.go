package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"github.com/samintheshell/rangekey/internal/storage"
	"github.com/samintheshell/rangekey/internal/raft"
	"github.com/samintheshell/rangekey/internal/metadata"
	"github.com/samintheshell/rangekey/internal/partition"
	"github.com/samintheshell/rangekey/internal/transaction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config holds the gRPC server configuration
type Config struct {
	ListenAddr     string
	RequestTimeout time.Duration
	Storage        *storage.Engine
	Raft           *raft.Node
	Metadata       *metadata.Store
	Partitions     *partition.Manager
	Transactions   *transaction.Manager
}

// Server represents the gRPC server
type Server struct {
	v1.UnimplementedRangeDBServer
	config *Config
	server *grpc.Server

	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// NewServer creates a new gRPC server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("gRPC server config is required")
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	}

	grpcServer := grpc.NewServer(opts...)

	server := &Server{
		config: config,
		server: grpcServer,
		stopCh: make(chan struct{}),
	}

	// Register the RangeDB service
	v1.RegisterRangeDBServer(grpcServer, server)

	return server, nil
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	if s.started {
		return fmt.Errorf("gRPC server is already started")
	}

	log.Printf("Starting gRPC server on %s", s.config.ListenAddr)

	// Listen on the specified address
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}

	// Start serving in a goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	s.started = true
	log.Printf("gRPC server started successfully on %s", s.config.ListenAddr)

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	if !s.started {
		return nil
	}

	close(s.stopCh)

	log.Println("Stopping gRPC server...")

	// Graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server stopped gracefully")
	case <-ctx.Done():
		log.Println("gRPC server shutdown timeout, forcing stop")
		s.server.Stop()
	}

	s.started = false
	return nil
}

// RangeDB service implementations

// Get handles key-value get requests
func (s *Server) Get(ctx context.Context, req *v1.GetRequest) (*v1.GetResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Add request timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	// Find partition for key
	partition, err := s.config.Partitions.FindPartitionForKey(ctx, req.Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find partition: %v", err)
	}

	// Check if we're the leader for this partition
	if !s.config.Partitions.IsPartitionLeader(partition.ID) {
		// Forward to leader
		return nil, status.Error(codes.FailedPrecondition, "not leader for this partition")
	}

	// Get from storage
	value, err := s.config.Storage.Get(ctx, req.Key)
	if err != nil {
		if err == storage.ErrKeyNotFound {
			return &v1.GetResponse{
				Found: false,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "storage error: %v", err)
	}

	return &v1.GetResponse{
		Kv: &v1.KeyValue{
			Key:   req.Key,
			Value: value,
		},
		Found: true,
	}, nil
}

// Put handles key-value put requests
func (s *Server) Put(ctx context.Context, req *v1.PutRequest) (*v1.PutResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Add request timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	// Find partition for key
	partition, err := s.config.Partitions.FindPartitionForKey(ctx, req.Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find partition: %v", err)
	}

	// Check if we're the leader for this partition
	if !s.config.Partitions.IsPartitionLeader(partition.ID) {
		// Forward to leader
		return nil, status.Error(codes.FailedPrecondition, "not leader for this partition")
	}

	// Use Raft consensus for write operations
	if s.config.Raft != nil {
		// Wait a bit for leadership to be established
		for i := 0; i < 5; i++ {
			if s.config.Raft.IsLeader() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if s.config.Raft.IsLeader() {
			// Create operation for Raft
			opData, err := raft.CreatePutOperation(string(req.Key), req.Value)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create operation: %v", err)
			}

			// Propose through Raft
			if err := s.config.Raft.Propose(ctx, opData); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to propose: %v", err)
			}
		} else {
			// Fallback to direct storage if not leader
			if err := s.config.Storage.Put(ctx, req.Key, req.Value); err != nil {
				return nil, status.Errorf(codes.Internal, "storage error: %v", err)
			}
		}
	} else {
		// Fallback to direct storage (for single-node deployment)
		if err := s.config.Storage.Put(ctx, req.Key, req.Value); err != nil {
			return nil, status.Errorf(codes.Internal, "storage error: %v", err)
		}
	}

	return &v1.PutResponse{
		Success: true,
		Version: 1, // TODO: implement versioning
	}, nil
}

// Delete handles key-value delete requests
func (s *Server) Delete(ctx context.Context, req *v1.DeleteRequest) (*v1.DeleteResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Add request timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	// Find partition for key
	partition, err := s.config.Partitions.FindPartitionForKey(ctx, req.Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find partition: %v", err)
	}

	// Check if we're the leader for this partition
	if !s.config.Partitions.IsPartitionLeader(partition.ID) {
		// Forward to leader
		return nil, status.Error(codes.FailedPrecondition, "not leader for this partition")
	}

	// Use Raft consensus for write operations
	if s.config.Raft != nil && s.config.Raft.IsLeader() {
		// Create operation for Raft
		opData, err := raft.CreateDeleteOperation(string(req.Key))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create operation: %v", err)
		}

		// Propose through Raft
		if err := s.config.Raft.Propose(ctx, opData); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to propose: %v", err)
		}
	} else {
		// Fallback to direct storage (for single-node deployment)
		if err := s.config.Storage.Delete(ctx, req.Key); err != nil {
			return nil, status.Errorf(codes.Internal, "storage error: %v", err)
		}
	}

	return &v1.DeleteResponse{
		Success: true,
		Version: 1, // TODO: implement versioning
	}, nil
}

// Range handles key-value range requests
func (s *Server) Range(req *v1.RangeRequest, stream v1.RangeDB_RangeServer) error {
	if !s.started {
		return status.Error(codes.Unavailable, "server is not started")
	}

	// Add request timeout
	ctx, cancel := context.WithTimeout(stream.Context(), s.config.RequestTimeout)
	defer cancel()

	// Determine limit
	limit := 100 // default limit
	if req.Limit != nil {
		limit = int(*req.Limit)
	}

	// For now, implement a simple range scan
	// TODO: Implement proper range partitioning
	results, err := s.config.Storage.Range(ctx, req.StartKey, req.EndKey, limit)
	if err != nil {
		return status.Errorf(codes.Internal, "storage error: %v", err)
	}

	// Send results
	count := 0
	for key, value := range results {
		if err := stream.Send(&v1.RangeResponse{
			Kv: &v1.KeyValue{
				Key:   []byte(key),
				Value: value,
			},
			HasMore: count < len(results)-1, // simple has_more logic
		}); err != nil {
			return status.Errorf(codes.Internal, "stream error: %v", err)
		}
		count++
	}

	return nil
}

// Batch handles batch requests
func (s *Server) Batch(ctx context.Context, req *v1.BatchRequest) (*v1.BatchResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Add request timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()

	// TODO: Implement proper batch processing
	results := make([]*v1.BatchResult, len(req.Operations))

	for i, op := range req.Operations {
		switch op.Operation.(type) {
		case *v1.BatchOperation_Put:
			putReq := op.GetPut()
			putResp, err := s.Put(ctx, putReq)
			if err != nil {
				return &v1.BatchResponse{Success: false}, err
			}
			results[i] = &v1.BatchResult{
				Result: &v1.BatchResult_Put{Put: putResp},
			}
		case *v1.BatchOperation_Delete:
			deleteReq := op.GetDelete()
			deleteResp, err := s.Delete(ctx, deleteReq)
			if err != nil {
				return &v1.BatchResponse{Success: false}, err
			}
			results[i] = &v1.BatchResult{
				Result: &v1.BatchResult_Delete{Delete: deleteResp},
			}
		}
	}

	return &v1.BatchResponse{
		Results: results,
		Success: true,
	}, nil
}

// Transaction operations
func (s *Server) BeginTransaction(ctx context.Context, req *v1.BeginTransactionRequest) (*v1.BeginTransactionResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	if s.config.Transactions == nil {
		return nil, status.Error(codes.Unimplemented, "transactions not configured")
	}

	return s.config.Transactions.BeginTransaction(ctx, req)
}

func (s *Server) CommitTransaction(ctx context.Context, req *v1.CommitTransactionRequest) (*v1.CommitTransactionResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	if s.config.Transactions == nil {
		return nil, status.Error(codes.Unimplemented, "transactions not configured")
	}

	return s.config.Transactions.CommitTransaction(ctx, req)
}

func (s *Server) RollbackTransaction(ctx context.Context, req *v1.RollbackTransactionRequest) (*v1.RollbackTransactionResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	if s.config.Transactions == nil {
		return nil, status.Error(codes.Unimplemented, "transactions not configured")
	}

	return s.config.Transactions.RollbackTransaction(ctx, req)
}

// Cluster operations
func (s *Server) GetClusterInfo(ctx context.Context, req *v1.GetClusterInfoRequest) (*v1.GetClusterInfoResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Get cluster info from metadata store
	clusterInfo, err := s.config.Metadata.GetClusterConfig(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get cluster info: %v", err)
	}

	// Get partition count
	partitions, err := s.config.Metadata.ListPartitions(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get partitions: %v", err)
	}

	// Convert to proto
	nodes := make([]*v1.NodeInfo, len(clusterInfo.Nodes))
	for i, node := range clusterInfo.Nodes {
		nodes[i] = &v1.NodeInfo{
			NodeId:        node.ID,
			PeerAddress:   node.PeerAddress,
			ClientAddress: node.ClientAddress,
			Status:        v1.NodeStatus_NODE_RUNNING, // TODO: implement proper status
		}
	}

	return &v1.GetClusterInfoResponse{
		ClusterId:         clusterInfo.ID,
		Nodes:             nodes,
		ReplicationFactor: int32(clusterInfo.ReplicationFactor),
		NumPartitions:     int64(len(partitions)),
	}, nil
}

func (s *Server) GetNodeInfo(ctx context.Context, req *v1.GetNodeInfoRequest) (*v1.GetNodeInfoResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Get node info from metadata store
	nodeInfo, err := s.config.Metadata.GetNodeInfo(ctx, s.config.Metadata.GetNodeID())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get node info: %v", err)
	}

	return &v1.GetNodeInfoResponse{
		NodeId:        nodeInfo.ID,
		PeerAddress:   nodeInfo.PeerAddress,
		ClientAddress: nodeInfo.ClientAddress,
		Status:        v1.NodeStatus_NODE_RUNNING, // TODO: implement proper status
	}, nil
}

func (s *Server) JoinCluster(ctx context.Context, req *v1.JoinClusterRequest) (*v1.JoinClusterResponse, error) {
	if !s.started {
		return nil, status.Error(codes.Unavailable, "server is not started")
	}

	// Get current cluster configuration
	clusterConfig, err := s.config.Metadata.GetClusterConfig(ctx)
	if err != nil {
		return &v1.JoinClusterResponse{
			Success: false,
			Message: fmt.Sprintf("failed to get cluster config: %v", err),
		}, nil
	}

	// Check if node is already in the cluster
	for _, node := range clusterConfig.Nodes {
		if node.ID == req.NodeId {
			return &v1.JoinClusterResponse{
				Success: false,
				Message: "node already exists in cluster",
			}, nil
		}
	}

	// Create new node info
	newNode := metadata.NodeInfo{
		ID:            req.NodeId,
		PeerAddress:   req.PeerAddress,
		ClientAddress: req.ClientAddress,
		Status:        metadata.NodeStatusJoining,
		JoinedAt:      time.Now(),
	}

	// Add node to cluster configuration
	clusterConfig.Nodes = append(clusterConfig.Nodes, newNode)
	clusterConfig.UpdatedAt = time.Now()

	// Store updated cluster configuration
	if err := s.config.Metadata.StoreClusterConfig(ctx, clusterConfig); err != nil {
		return &v1.JoinClusterResponse{
			Success: false,
			Message: fmt.Sprintf("failed to store cluster config: %v", err),
		}, nil
	}

	// Convert metadata nodes to protobuf nodes
	var nodes []*v1.NodeInfo
	for _, node := range clusterConfig.Nodes {
		nodes = append(nodes, &v1.NodeInfo{
			NodeId:        node.ID,
			PeerAddress:   node.PeerAddress,
			ClientAddress: node.ClientAddress,
			Status:        v1.NodeStatus_NODE_RUNNING,
		})
	}

	return &v1.JoinClusterResponse{
		Success:   true,
		Message:   "node joined cluster successfully",
		ClusterId: clusterConfig.ID,
		Nodes:     nodes,
	}, nil
}

// loggingUnaryInterceptor logs unary gRPC requests
func loggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	if err != nil {
		log.Printf("gRPC [%s] failed in %v: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("gRPC [%s] completed in %v", info.FullMethod, duration)
	}

	return resp, err
}

// loggingStreamInterceptor logs streaming gRPC requests
func loggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()

	err := handler(srv, ss)

	duration := time.Since(start)

	if err != nil {
		log.Printf("gRPC Stream [%s] failed in %v: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("gRPC Stream [%s] completed in %v", info.FullMethod, duration)
	}

	return err
}
