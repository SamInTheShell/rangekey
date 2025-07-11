package raft

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	raftapi "github.com/samintheshell/rangekey/api/raft/v1"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Transport handles inter-node Raft communication
type Transport struct {
	config *TransportConfig

	// gRPC server for receiving messages
	server *grpc.Server

	// Client connections to other nodes
	clients map[uint64]*grpc.ClientConn
	clientsMu sync.RWMutex

	// Peer addresses
	peers map[uint64]string
	peersMu sync.RWMutex

	// Message handling
	messageHandler MessageHandler

	// Lifecycle
	started bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// TransportConfig holds the transport configuration
type TransportConfig struct {
	ListenAddr string
	NodeID     uint64
	DialTimeout time.Duration
}

// MessageHandler defines the interface for handling incoming Raft messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *raftpb.Message) error
}

// RaftTransportService is the gRPC service for Raft communication
type RaftTransportService struct {
	raftapi.UnimplementedRaftTransportServer
	transport *Transport
}

// NewTransport creates a new Raft transport
func NewTransport(config *TransportConfig) *Transport {
	return &Transport{
		config:  config,
		clients: make(map[uint64]*grpc.ClientConn),
		peers:   make(map[uint64]string),
		stopCh:  make(chan struct{}),
	}
}

// SetMessageHandler sets the message handler
func (t *Transport) SetMessageHandler(handler MessageHandler) {
	t.messageHandler = handler
}

// Start starts the transport layer
func (t *Transport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport is already started")
	}

	// Create gRPC server
	t.server = grpc.NewServer()

	// Register the transport service
	service := &RaftTransportService{transport: t}
	raftapi.RegisterRaftTransportServer(t.server, service)

	// Listen on the configured address
	listener, err := net.Listen("tcp", t.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", t.config.ListenAddr, err)
	}

	log.Printf("Starting Raft transport on %s", t.config.ListenAddr)

	// Start the gRPC server
	go func() {
		if err := t.server.Serve(listener); err != nil {
			log.Printf("Raft transport server error: %v", err)
		}
	}()

	t.started = true
	log.Printf("Raft transport started on %s", t.config.ListenAddr)

	return nil
}

// Stop stops the transport layer
func (t *Transport) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return nil
	}

	log.Println("Stopping Raft transport...")

	// Stop the gRPC server
	if t.server != nil {
		t.server.GracefulStop()
	}

	// Close all client connections
	t.clientsMu.Lock()
	for nodeID, conn := range t.clients {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection to node %d: %v", nodeID, err)
		}
	}
	t.clients = make(map[uint64]*grpc.ClientConn)
	t.clientsMu.Unlock()

	// Signal stop
	close(t.stopCh)

	t.started = false
	log.Println("Raft transport stopped")

	return nil
}

// SendMessage sends a Raft message to another node
func (t *Transport) SendMessage(ctx context.Context, nodeID uint64, msg *raftpb.Message) error {
	if nodeID == t.config.NodeID {
		// Local message, handle directly
		if t.messageHandler != nil {
			return t.messageHandler.HandleMessage(ctx, msg)
		}
		return nil
	}

	// Get or create client connection
	conn, err := t.getOrCreateClient(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get client for node %d: %w", nodeID, err)
	}

	// Create the gRPC client
	client := raftapi.NewRaftTransportClient(conn)

	// Serialize the Raft message
	msgData, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal Raft message: %w", err)
	}

	// Create the transport message
	transportMsg := &raftapi.RaftMessage{
		MessageData: msgData,
		From:        t.config.NodeID,
		To:          nodeID,
		Timestamp:   time.Now().UnixNano(),
	}

	// Send the message
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := client.SendMessage(ctx, transportMsg)
	if err != nil {
		return fmt.Errorf("failed to send message to node %d: %w", nodeID, err)
	}

	if !response.Success {
		return fmt.Errorf("message to node %d failed: %s", nodeID, response.Error)
	}

	log.Printf("Successfully sent Raft message to node %d: %s", nodeID, msg.Type)
	return nil
}

// getOrCreateClient gets or creates a gRPC client connection to a node
func (t *Transport) getOrCreateClient(nodeID uint64) (*grpc.ClientConn, error) {
	t.clientsMu.RLock()
	if conn, exists := t.clients[nodeID]; exists {
		t.clientsMu.RUnlock()
		return conn, nil
	}
	t.clientsMu.RUnlock()

	t.clientsMu.Lock()
	defer t.clientsMu.Unlock()

	// Double-check after acquiring write lock
	if conn, exists := t.clients[nodeID]; exists {
		return conn, nil
	}

	// Get node address from peers map
	t.peersMu.RLock()
	address, exists := t.peers[nodeID]
	t.peersMu.RUnlock()

	if !exists {
		// Fallback to localhost with port based on node ID
		address = fmt.Sprintf("localhost:%d", 19082+nodeID-1)
		log.Printf("No peer address found for node %d, using fallback: %s", nodeID, address)
	}

	// Create new connection
	ctx, cancel := context.WithTimeout(context.Background(), t.config.DialTimeout)
	defer cancel()

	log.Printf("Attempting to connect to node %d at %s", nodeID, address)
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect to node %d at %s: %v", nodeID, address, err)
		return nil, fmt.Errorf("failed to connect to node %d at %s: %w", nodeID, address, err)
	}

	t.clients[nodeID] = conn
	log.Printf("Successfully created connection to node %d at %s", nodeID, address)

	return conn, nil
}

// SendMessage handles incoming Raft messages from other nodes
func (s *RaftTransportService) SendMessage(ctx context.Context, req *raftapi.RaftMessage) (*raftapi.RaftMessageResponse, error) {
	// Deserialize the Raft message
	var msg raftpb.Message
	if err := msg.Unmarshal(req.MessageData); err != nil {
		return &raftapi.RaftMessageResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to unmarshal message: %v", err),
		}, nil
	}

	// Handle the message through the transport's message handler
	if s.transport.messageHandler != nil {
		if err := s.transport.messageHandler.HandleMessage(ctx, &msg); err != nil {
			return &raftapi.RaftMessageResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to handle message: %v", err),
			}, nil
		}
	}

	log.Printf("Received Raft message from node %d: %s", req.From, msg.Type)

	return &raftapi.RaftMessageResponse{
		Success: true,
		Error:   "",
	}, nil
}

// SendSnapshot handles incoming snapshot chunks from other nodes
func (s *RaftTransportService) SendSnapshot(stream raftapi.RaftTransport_SendSnapshotServer) error {
	// TODO: Implement snapshot streaming
	return status.Error(codes.Unimplemented, "snapshot streaming not implemented yet")
}

// Heartbeat handles heartbeat requests for connectivity checks
func (s *RaftTransportService) Heartbeat(ctx context.Context, req *raftapi.HeartbeatRequest) (*raftapi.HeartbeatResponse, error) {
	return &raftapi.HeartbeatResponse{
		Alive:     true,
		NodeId:    s.transport.config.NodeID,
		Timestamp: time.Now().UnixNano(),
		Status:    "healthy",
	}, nil
}

// AddPeer adds a peer node to the transport layer
func (t *Transport) AddPeer(nodeID uint64, address string) error {
	t.peersMu.Lock()
	defer t.peersMu.Unlock()

	t.peers[nodeID] = address
	log.Printf("Added peer node %d at %s", nodeID, address)

	return nil
}

// RemovePeer removes a peer node from the transport layer
func (t *Transport) RemovePeer(nodeID uint64) error {
	t.clientsMu.Lock()
	defer t.clientsMu.Unlock()

	if conn, exists := t.clients[nodeID]; exists {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection to node %d: %v", nodeID, err)
		}
		delete(t.clients, nodeID)
		log.Printf("Removed peer node %d", nodeID)
	}

	return nil
}
