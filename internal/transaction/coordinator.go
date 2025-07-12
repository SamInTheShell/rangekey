package transaction

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/samintheshell/rangekey/internal/metadata"
)

// DistributedTransactionCoordinator manages distributed transactions across multiple partitions
type DistributedTransactionCoordinator struct {
	nodeID   string
	metadata *metadata.Store
	
	// Active distributed transactions
	mu                  sync.RWMutex
	distributedTxns     map[string]*DistributedTransaction
	participantManagers map[string]ParticipantManager
	
	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// DistributedTransaction represents a distributed transaction
type DistributedTransaction struct {
	ID                string
	CoordinatorNodeID string
	Participants      []string // List of participant node IDs
	State             DistributedTransactionState
	StartTime         time.Time
	Timeout           time.Duration
	LastAccess        time.Time
	
	// Two-phase commit state
	mu                sync.RWMutex
	prepareResults    map[string]bool // participant -> prepare result
	commitResults     map[string]bool // participant -> commit result
	abortResults      map[string]bool // participant -> abort result
}

// DistributedTransactionState represents the state of a distributed transaction
type DistributedTransactionState int

const (
	DTxnActive DistributedTransactionState = iota
	DTxnPreparing
	DTxnPrepared
	DTxnCommitting
	DTxnCommitted
	DTxnAborting
	DTxnAborted
)

// ParticipantManager interface for managing transaction participants
type ParticipantManager interface {
	Prepare(ctx context.Context, txnID string) error
	Commit(ctx context.Context, txnID string) error
	Abort(ctx context.Context, txnID string) error
}

// NewDistributedTransactionCoordinator creates a new distributed transaction coordinator
func NewDistributedTransactionCoordinator(nodeID string, metadata *metadata.Store) *DistributedTransactionCoordinator {
	return &DistributedTransactionCoordinator{
		nodeID:              nodeID,
		metadata:            metadata,
		distributedTxns:     make(map[string]*DistributedTransaction),
		participantManagers: make(map[string]ParticipantManager),
		stopCh:              make(chan struct{}),
	}
}

// Start starts the distributed transaction coordinator
func (c *DistributedTransactionCoordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.started {
		return fmt.Errorf("distributed transaction coordinator is already started")
	}
	
	log.Println("Starting distributed transaction coordinator...")
	
	// Start background cleanup goroutine
	go c.cleanupExpiredDistributedTransactions()
	
	c.started = true
	log.Println("Distributed transaction coordinator started successfully")
	
	return nil
}

// Stop stops the distributed transaction coordinator
func (c *DistributedTransactionCoordinator) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.started {
		return nil
	}
	
	log.Println("Stopping distributed transaction coordinator...")
	
	// Signal stop to background goroutines
	close(c.stopCh)
	
	// Abort all active distributed transactions
	for id, dtxn := range c.distributedTxns {
		if dtxn.State == DTxnActive || dtxn.State == DTxnPreparing {
			c.abortDistributedTransaction(ctx, dtxn)
			log.Printf("Aborted distributed transaction %s during shutdown", id)
		}
	}
	
	c.started = false
	log.Println("Distributed transaction coordinator stopped")
	
	return nil
}

// BeginDistributedTransaction starts a new distributed transaction
func (c *DistributedTransactionCoordinator) BeginDistributedTransaction(ctx context.Context, participants []string, timeout time.Duration) (*DistributedTransaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.started {
		return nil, fmt.Errorf("distributed transaction coordinator is not started")
	}
	
	// Get coordinator node ID
	coordinatorNodeID := c.nodeID
	if coordinatorNodeID == "" {
		coordinatorNodeID = "unknown"
	}
	
	// Generate transaction ID
	txnID := fmt.Sprintf("dtxn-%d-%s", time.Now().UnixNano(), coordinatorNodeID)
	
	// Create distributed transaction
	dtxn := &DistributedTransaction{
		ID:                txnID,
		CoordinatorNodeID: coordinatorNodeID,
		Participants:      participants,
		State:             DTxnActive,
		StartTime:         time.Now(),
		Timeout:           timeout,
		LastAccess:        time.Now(),
		prepareResults:    make(map[string]bool),
		commitResults:     make(map[string]bool),
		abortResults:      make(map[string]bool),
	}
	
	c.distributedTxns[txnID] = dtxn
	
	log.Printf("Started distributed transaction %s with participants: %v", txnID, participants)
	return dtxn, nil
}

// PrepareDistributedTransaction executes the prepare phase of 2PC
func (c *DistributedTransactionCoordinator) PrepareDistributedTransaction(ctx context.Context, txnID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dtxn, exists := c.distributedTxns[txnID]
	if !exists {
		return fmt.Errorf("distributed transaction not found: %s", txnID)
	}
	
	dtxn.mu.Lock()
	defer dtxn.mu.Unlock()
	
	if dtxn.State != DTxnActive {
		return fmt.Errorf("distributed transaction is not active: %s", txnID)
	}
	
	// Change state to preparing
	dtxn.State = DTxnPreparing
	dtxn.LastAccess = time.Now()
	
	log.Printf("Starting prepare phase for distributed transaction %s", txnID)
	
	// Send prepare messages to all participants
	prepareSuccess := true
	for _, participant := range dtxn.Participants {
		manager, exists := c.participantManagers[participant]
		if !exists {
			log.Printf("No participant manager found for %s", participant)
			dtxn.prepareResults[participant] = false
			prepareSuccess = false
			continue
		}
		
		err := manager.Prepare(ctx, txnID)
		if err != nil {
			log.Printf("Prepare failed for participant %s: %v", participant, err)
			dtxn.prepareResults[participant] = false
			prepareSuccess = false
		} else {
			dtxn.prepareResults[participant] = true
		}
	}
	
	// Update state based on prepare results
	if prepareSuccess {
		dtxn.State = DTxnPrepared
		log.Printf("Prepare phase succeeded for distributed transaction %s", txnID)
	} else {
		dtxn.State = DTxnAborting
		log.Printf("Prepare phase failed for distributed transaction %s", txnID)
	}
	
	return nil
}

// CommitDistributedTransaction executes the commit phase of 2PC
func (c *DistributedTransactionCoordinator) CommitDistributedTransaction(ctx context.Context, txnID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dtxn, exists := c.distributedTxns[txnID]
	if !exists {
		return fmt.Errorf("distributed transaction not found: %s", txnID)
	}
	
	dtxn.mu.Lock()
	defer dtxn.mu.Unlock()
	
	if dtxn.State != DTxnPrepared {
		return fmt.Errorf("distributed transaction is not prepared: %s", txnID)
	}
	
	// Change state to committing
	dtxn.State = DTxnCommitting
	dtxn.LastAccess = time.Now()
	
	log.Printf("Starting commit phase for distributed transaction %s", txnID)
	
	// Send commit messages to all participants
	commitSuccess := true
	for _, participant := range dtxn.Participants {
		manager, exists := c.participantManagers[participant]
		if !exists {
			log.Printf("No participant manager found for %s", participant)
			dtxn.commitResults[participant] = false
			commitSuccess = false
			continue
		}
		
		err := manager.Commit(ctx, txnID)
		if err != nil {
			log.Printf("Commit failed for participant %s: %v", participant, err)
			dtxn.commitResults[participant] = false
			commitSuccess = false
		} else {
			dtxn.commitResults[participant] = true
		}
	}
	
	// Update state based on commit results
	if commitSuccess {
		dtxn.State = DTxnCommitted
		log.Printf("Commit phase succeeded for distributed transaction %s", txnID)
	} else {
		// This is a serious error - some participants committed while others failed
		log.Printf("CRITICAL: Partial commit failure for distributed transaction %s", txnID)
		dtxn.State = DTxnCommitted // Mark as committed but log the inconsistency
	}
	
	// Clean up transaction
	delete(c.distributedTxns, txnID)
	
	return nil
}

// AbortDistributedTransaction aborts a distributed transaction
func (c *DistributedTransactionCoordinator) AbortDistributedTransaction(ctx context.Context, txnID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dtxn, exists := c.distributedTxns[txnID]
	if !exists {
		return fmt.Errorf("distributed transaction not found: %s", txnID)
	}
	
	return c.abortDistributedTransaction(ctx, dtxn)
}

// abortDistributedTransaction internal method to abort a distributed transaction
func (c *DistributedTransactionCoordinator) abortDistributedTransaction(ctx context.Context, dtxn *DistributedTransaction) error {
	dtxn.mu.Lock()
	defer dtxn.mu.Unlock()
	
	if dtxn.State == DTxnCommitted || dtxn.State == DTxnAborted {
		return nil // Already finished
	}
	
	// Change state to aborting
	dtxn.State = DTxnAborting
	dtxn.LastAccess = time.Now()
	
	log.Printf("Aborting distributed transaction %s", dtxn.ID)
	
	// Send abort messages to all participants
	for _, participant := range dtxn.Participants {
		manager, exists := c.participantManagers[participant]
		if !exists {
			log.Printf("No participant manager found for %s", participant)
			dtxn.abortResults[participant] = false
			continue
		}
		
		err := manager.Abort(ctx, dtxn.ID)
		if err != nil {
			log.Printf("Abort failed for participant %s: %v", participant, err)
			dtxn.abortResults[participant] = false
		} else {
			dtxn.abortResults[participant] = true
		}
	}
	
	// Update state to aborted
	dtxn.State = DTxnAborted
	
	// Clean up transaction
	delete(c.distributedTxns, dtxn.ID)
	
	return nil
}

// AddParticipantManager adds a participant manager for a node
func (c *DistributedTransactionCoordinator) AddParticipantManager(nodeID string, manager ParticipantManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.participantManagers[nodeID] = manager
	log.Printf("Added participant manager for node %s", nodeID)
}

// RemoveParticipantManager removes a participant manager
func (c *DistributedTransactionCoordinator) RemoveParticipantManager(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.participantManagers, nodeID)
	log.Printf("Removed participant manager for node %s", nodeID)
}

// cleanupExpiredDistributedTransactions cleans up expired distributed transactions
func (c *DistributedTransactionCoordinator) cleanupExpiredDistributedTransactions() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			
			for id, dtxn := range c.distributedTxns {
				if now.Sub(dtxn.StartTime) > dtxn.Timeout {
					c.abortDistributedTransaction(context.Background(), dtxn)
					log.Printf("Cleaned up expired distributed transaction: %s", id)
				}
			}
			
			c.mu.Unlock()
			
		case <-c.stopCh:
			return
		}
	}
}

// GetDistributedTransactionState returns the state of a distributed transaction
func (c *DistributedTransactionCoordinator) GetDistributedTransactionState(txnID string) (DistributedTransactionState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	dtxn, exists := c.distributedTxns[txnID]
	if !exists {
		return DTxnAborted, fmt.Errorf("distributed transaction not found: %s", txnID)
	}
	
	return dtxn.State, nil
}