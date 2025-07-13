package raft

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/samintheshell/rangekey/internal/storage"
	"github.com/samintheshell/rangekey/internal/metadata"
	"github.com/samintheshell/rangekey/internal/partition"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// Config holds the Raft node configuration
type Config struct {
	NodeID           string
	ListenAddr       string
	DataDir          string
	HeartbeatTimeout time.Duration
	ElectionTimeout  time.Duration
	Storage          *storage.Engine
	Metadata         *metadata.Store
	Partitions       *partition.Manager
	Peers            []string // List of initial peers
	Join             bool     // Whether joining an existing cluster
}

// Node represents a Raft node
type Node struct {
	config *Config

	// Raft components
	node        raft.Node
	raftStorage *raft.MemoryStorage

	// State machine
	stateMachine *StateMachine

	// Transport layer
	transport *Transport

	// Commit and error channels
	commitC     chan *commit
	errorC      chan error

	// Lifecycle
	started bool
	stopCh  chan struct{}
	mu      sync.RWMutex
}

// NodeState represents the state of a Raft node
type NodeState int

const (
	StateFollower NodeState = iota
	StateCandidate
	StateLeader
)

// commit represents a committed entry
type commit struct {
	data       []byte
	applyDoneC chan struct{}
}

// peerToRaftAddress converts a peer address to a Raft transport address
// Peer addresses use port 8080, Raft addresses use port 8082
func peerToRaftAddress(peerAddr string) string {
	host, _, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return peerAddr // Return as-is if parsing fails
	}
	return fmt.Sprintf("%s:8082", host)
}

// NewNode creates a new Raft node
func NewNode(config *Config) (*Node, error) {
	if config == nil {
		return nil, fmt.Errorf("raft config is required")
	}

	// Create the commit and error channels
	commitC := make(chan *commit)
	errorC := make(chan error)

	// Create Raft storage
	raftStorage := raft.NewMemoryStorage()

	// Create state machine
	stateMachine := NewStateMachine(config.Storage, commitC, errorC)

	// Parse node ID to uint64
	nodeID, err := strconv.ParseUint(config.NodeID, 10, 64)
	if err != nil {
		// If NodeID is not numeric, create a hash
		nodeID = uint64(len(config.NodeID))
		for _, b := range []byte(config.NodeID) {
			nodeID = nodeID*31 + uint64(b)
		}
	}

	// Create transport layer
	transport := NewTransport(&TransportConfig{
		ListenAddr:  config.ListenAddr,
		NodeID:      nodeID,
		DialTimeout: 5 * time.Second,
	})

	// Create the node
	node := &Node{
		config:       config,
		raftStorage:  raftStorage,
		stateMachine: stateMachine,
		transport:    transport,
		commitC:      commitC,
		errorC:       errorC,
		stopCh:       make(chan struct{}),
	}

	// Set node as the transport's message handler
	transport.SetMessageHandler(node)

	return node, nil
}

// Start starts the Raft node
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.started {
		return fmt.Errorf("raft node is already started")
	}

	log.Printf("Starting Raft node: %s", n.config.NodeID)

	// Start the transport layer first
	if err := n.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport layer: %w", err)
	}

	// Parse node ID to uint64
	nodeID, err := strconv.ParseUint(n.config.NodeID, 10, 64)
	if err != nil {
		// If NodeID is not numeric, create a hash
		nodeID = uint64(len(n.config.NodeID))
		for _, b := range []byte(n.config.NodeID) {
			nodeID = nodeID*31 + uint64(b)
		}
	}

	// Create Raft configuration
	raftConfig := &raft.Config{
		ID:              nodeID,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         n.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		Logger:          &raftLogger{},
	}

	// Create initial peer list and add them to transport
	var peers []raft.Peer
	if len(n.config.Peers) > 0 {
		for i, peer := range n.config.Peers {
			peerID := uint64(i + 1)
			if peer == n.config.ListenAddr {
				peerID = nodeID
			}
			peers = append(peers, raft.Peer{
				ID:      peerID,
				Context: []byte(peer),
			})

			// Add peer to transport layer - don't add ourselves
			if peer != n.config.ListenAddr {
				raftAddr := peerToRaftAddress(peer)
				n.transport.AddPeer(peerID, raftAddr)
			}
		}
	} else {
		// Single node cluster
		peers = []raft.Peer{{
			ID:      nodeID,
			Context: []byte(n.config.NodeID),
		}}
	}

	// Start Raft node
	if !n.config.Join {
		// Starting a new cluster
		n.node = raft.StartNode(raftConfig, peers)
	} else {
		// Joining existing cluster - use RestartNode for existing clusters
		n.node = raft.RestartNode(raftConfig)
	}

	// Start the state machine
	if err := n.stateMachine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start state machine: %w", err)
	}

	// Start the main loop
	go n.run()

	n.started = true
	log.Println("Raft node started successfully")

	return nil
}

// Stop stops the Raft node
func (n *Node) Stop(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.started {
		return nil
	}

	log.Println("Stopping Raft node...")

	// Stop the transport layer
	if n.transport != nil {
		n.transport.Stop(ctx)
	}

	// Stop the state machine
	if n.stateMachine != nil {
		n.stateMachine.Stop(ctx)
	}

	// Stop the main loop
	close(n.stopCh)

	// Stop the Raft node
	if n.node != nil {
		n.node.Stop()
	}

	n.started = false
	log.Println("Raft node stopped")

	return nil
}

// IsLeader returns true if this node is the leader
func (n *Node) IsLeader() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.node == nil {
		return false
	}

	return n.node.Status().Lead == n.node.Status().ID
}

// GetLeader returns the current leader node ID
func (n *Node) GetLeader() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.node == nil {
		return ""
	}

	leadID := n.node.Status().Lead
	if leadID == 0 {
		return ""
	}

	return fmt.Sprintf("%d", leadID)
}

// GetTerm returns the current term
func (n *Node) GetTerm() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.node == nil {
		return 0
	}

	return n.node.Status().Term
}

// HealthCheck performs a Raft health check
func (n *Node) HealthCheck(ctx context.Context) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.started {
		return fmt.Errorf("raft node is not started")
	}

	if n.node == nil {
		return fmt.Errorf("raft node is not initialized")
	}

	// Check if we have a leader
	status := n.node.Status()
	if status.Lead == 0 {
		return fmt.Errorf("no leader elected")
	}

	return nil
}

// Propose proposes a new entry to the Raft log
func (n *Node) Propose(ctx context.Context, data []byte) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.started {
		return fmt.Errorf("raft node is not started")
	}

	if n.node == nil {
		return fmt.Errorf("raft node is not initialized")
	}

	// Only leaders can propose
	if !n.IsLeader() {
		return fmt.Errorf("only leader can propose entries")
	}

	// Propose the entry
	return n.node.Propose(ctx, data)
}

// ReadIndex provides linearizable read support
func (n *Node) ReadIndex(ctx context.Context) (uint64, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.started {
		return 0, fmt.Errorf("raft node is not started")
	}

	if n.node == nil {
		return 0, fmt.Errorf("raft node is not initialized")
	}

	// Use ReadIndex for linearizable reads
	return 0, n.node.ReadIndex(ctx, nil)
}

// Process implements the raft.Node interface for receiving messages
func (n *Node) Process(ctx context.Context, m raftpb.Message) error {
	return n.node.Step(ctx, m)
}

// IsIDRemoved implements the raft.Node interface
func (n *Node) IsIDRemoved(id uint64) bool {
	return false
}

// ReportUnreachable implements the raft.Node interface
func (n *Node) ReportUnreachable(id uint64) {
	n.node.ReportUnreachable(id)
}

// ReportSnapshot implements the raft.Node interface
func (n *Node) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	n.node.ReportSnapshot(id, status)
}

// raftLogger implements the raft.Logger interface
type raftLogger struct{}

func (l *raftLogger) Debug(v ...interface{}) {
	log.Print(v...)
}

func (l *raftLogger) Debugf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *raftLogger) Error(v ...interface{}) {
	log.Print(v...)
}

func (l *raftLogger) Errorf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *raftLogger) Info(v ...interface{}) {
	log.Print(v...)
}

func (l *raftLogger) Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *raftLogger) Warning(v ...interface{}) {
	log.Print(v...)
}

func (l *raftLogger) Warningf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *raftLogger) Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func (l *raftLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

func (l *raftLogger) Panic(v ...interface{}) {
	log.Panic(v...)
}

func (l *raftLogger) Panicf(format string, v ...interface{}) {
	log.Panicf(format, v...)
}

// sendMessage sends a message to another node
func (n *Node) sendMessage(ctx context.Context, msg raftpb.Message) error {
	if n.transport == nil {
		return fmt.Errorf("transport layer not available")
	}

	return n.transport.SendMessage(ctx, msg.To, &msg)
}

// HandleMessage implements MessageHandler interface
func (n *Node) HandleMessage(ctx context.Context, msg *raftpb.Message) error {
	return n.node.Step(ctx, *msg)
}

// AddNode adds a new node to the cluster
func (n *Node) AddNode(ctx context.Context, nodeID uint64, address string) error {
	if n.transport != nil {
		n.transport.AddPeer(nodeID, address)
	}

	// Add node to Raft cluster
	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  nodeID,
		Context: []byte(address),
	}

	return n.node.ProposeConfChange(ctx, cc)
}

// RemoveNode removes a node from the cluster
func (n *Node) RemoveNode(ctx context.Context, nodeID uint64) error {
	if n.transport != nil {
		n.transport.RemovePeer(nodeID)
	}

	// Remove node from Raft cluster
	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeRemoveNode,
		NodeID: nodeID,
	}

	return n.node.ProposeConfChange(ctx, cc)
}

// CreateSnapshot creates a snapshot of the current state
func (n *Node) CreateSnapshot(ctx context.Context) error {
	// TODO: Implement snapshot creation
	log.Printf("Creating snapshot for node %s", n.config.NodeID)
	return nil
}

// RestoreSnapshot restores from a snapshot
func (n *Node) RestoreSnapshot(ctx context.Context, snapshot []byte) error {
	// TODO: Implement snapshot restoration
	log.Printf("Restoring snapshot for node %s", n.config.NodeID)
	return nil
}

// run is the main Raft loop
func (n *Node) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Track if we need to campaign for single-node cluster
	needCampaign := len(n.config.Peers) <= 1 && !n.config.Join
	campaignTimer := time.NewTimer(200 * time.Millisecond)
	defer campaignTimer.Stop()

	for {
		select {
		case <-ticker.C:
			n.node.Tick()

		case <-campaignTimer.C:
			if needCampaign {
				log.Printf("Starting campaign for single-node cluster")
				n.node.Campaign(context.Background())
				needCampaign = false
			}

		case rd := <-n.node.Ready():
			// Save to storage
			if err := n.raftStorage.Append(rd.Entries); err != nil {
				log.Printf("Failed to append entries: %v", err)
				continue
			}

			// Send messages to other nodes
			for _, msg := range rd.Messages {
				go func(msg raftpb.Message) {
					if err := n.sendMessage(context.Background(), msg); err != nil {
						log.Printf("Failed to send message to node %d: %v", msg.To, err)
					}
				}(msg)
			}

			// Apply committed entries
			for _, entry := range rd.CommittedEntries {
				if entry.Data != nil {
					cc := &commit{
						data:       entry.Data,
						applyDoneC: make(chan struct{}),
					}
					select {
					case n.commitC <- cc:
					case <-n.stopCh:
						return
					}
					<-cc.applyDoneC
				}
			}

			// Advance
			n.node.Advance()

		case <-n.stopCh:
			return
		}
	}
}
