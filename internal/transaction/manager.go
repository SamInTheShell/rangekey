package transaction

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"github.com/samintheshell/rangekey/internal/storage"
)

// IsolationLevel represents the isolation level for transactions
type IsolationLevel int

const (
	ReadCommitted IsolationLevel = iota
	ReadUncommitted
	SerializableSnapshot
)

// TransactionState represents the state of a transaction
type TransactionState int

const (
	TxnActive TransactionState = iota
	TxnPrepared
	TxnCommitted
	TxnAborted
)

// Transaction represents a database transaction
type Transaction struct {
	ID            string
	State         TransactionState
	IsolationLevel IsolationLevel
	StartTime     time.Time
	Timeout       time.Duration

	// Transaction operations
	mu        sync.RWMutex
	reads     map[string][]byte  // key -> value at read time
	writes    map[string][]byte  // key -> value to write
	deletes   map[string]bool    // key -> true if deleted

	// Metadata
	LastAccess time.Time
}

// Config holds the transaction manager configuration
type Config struct {
	DefaultTimeout time.Duration
	MaxTransactions int
	Storage        *storage.Engine
}

// Manager manages transactions
type Manager struct {
	config *Config

	// Transaction tracking
	mu           sync.RWMutex
	transactions map[string]*Transaction

	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// NewManager creates a new transaction manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("transaction manager config is required")
	}

	if config.Storage == nil {
		return nil, fmt.Errorf("storage engine is required")
	}

	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}

	if config.MaxTransactions == 0 {
		config.MaxTransactions = 1000
	}

	return &Manager{
		config:       config,
		transactions: make(map[string]*Transaction),
		stopCh:       make(chan struct{}),
	}, nil
}

// Start starts the transaction manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("transaction manager is already started")
	}

	log.Println("Starting transaction manager...")

	// Start background cleanup goroutine
	go m.cleanupExpiredTransactions()

	m.started = true
	log.Println("Transaction manager started successfully")

	return nil
}

// Stop stops the transaction manager
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	log.Println("Stopping transaction manager...")

	// Signal stop to background goroutines
	close(m.stopCh)

	// Abort all active transactions
	for id, txn := range m.transactions {
		if txn.State == TxnActive {
			txn.State = TxnAborted
			log.Printf("Aborted transaction %s during shutdown", id)
		}
	}

	m.started = false
	log.Println("Transaction manager stopped")

	return nil
}

// BeginTransaction starts a new transaction
func (m *Manager) BeginTransaction(ctx context.Context, req *v1.BeginTransactionRequest) (*v1.BeginTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil, fmt.Errorf("transaction manager is not started")
	}

	// Check if we have too many transactions
	if len(m.transactions) >= m.config.MaxTransactions {
		return nil, fmt.Errorf("too many active transactions")
	}

	// Generate transaction ID
	txnID := uuid.New().String()

	// Set timeout
	timeout := m.config.DefaultTimeout
	if req.TimeoutMs != nil {
		timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
	}

	// Set isolation level
	isolationLevel := ReadCommitted
	if req.IsolationLevel != nil {
		switch *req.IsolationLevel {
		case v1.IsolationLevel_READ_UNCOMMITTED:
			isolationLevel = ReadUncommitted
		case v1.IsolationLevel_READ_COMMITTED:
			isolationLevel = ReadCommitted
		case v1.IsolationLevel_SERIALIZABLE:
			isolationLevel = SerializableSnapshot
		}
	}

	// Create transaction
	txn := &Transaction{
		ID:             txnID,
		State:          TxnActive,
		IsolationLevel: isolationLevel,
		StartTime:      time.Now(),
		Timeout:        timeout,
		reads:          make(map[string][]byte),
		writes:         make(map[string][]byte),
		deletes:        make(map[string]bool),
		LastAccess:     time.Now(),
	}

	m.transactions[txnID] = txn

	return &v1.BeginTransactionResponse{
		TransactionId: txnID,
		Timestamp:     txn.StartTime.UnixNano(),
	}, nil
}

// CommitTransaction commits a transaction
func (m *Manager) CommitTransaction(ctx context.Context, req *v1.CommitTransactionRequest) (*v1.CommitTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil, fmt.Errorf("transaction manager is not started")
	}

	// Find transaction
	txn, exists := m.transactions[req.TransactionId]
	if !exists {
		return &v1.CommitTransactionResponse{
			Success: false,
		}, fmt.Errorf("transaction not found: %s", req.TransactionId)
	}

	// Check transaction state
	if txn.State != TxnActive {
		return &v1.CommitTransactionResponse{
			Success: false,
		}, fmt.Errorf("transaction is not active: %s", req.TransactionId)
	}

	// Check timeout
	if time.Since(txn.StartTime) > txn.Timeout {
		txn.State = TxnAborted
		return &v1.CommitTransactionResponse{
			Success: false,
		}, fmt.Errorf("transaction timed out: %s", req.TransactionId)
	}

	// Apply writes and deletes to storage
	err := m.applyTransactionChanges(ctx, txn)
	if err != nil {
		txn.State = TxnAborted
		return &v1.CommitTransactionResponse{
			Success: false,
		}, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Mark as committed
	txn.State = TxnCommitted
	commitTime := time.Now()

	// Clean up transaction (remove from active list)
	delete(m.transactions, req.TransactionId)

	return &v1.CommitTransactionResponse{
		Success:         true,
		CommitTimestamp: commitTime.UnixNano(),
	}, nil
}

// RollbackTransaction rolls back a transaction
func (m *Manager) RollbackTransaction(ctx context.Context, req *v1.RollbackTransactionRequest) (*v1.RollbackTransactionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil, fmt.Errorf("transaction manager is not started")
	}

	// Find transaction
	txn, exists := m.transactions[req.TransactionId]
	if !exists {
		return &v1.RollbackTransactionResponse{
			Success: false,
		}, fmt.Errorf("transaction not found: %s", req.TransactionId)
	}

	// Mark as aborted
	txn.State = TxnAborted

	// Clean up transaction (remove from active list)
	delete(m.transactions, req.TransactionId)

	return &v1.RollbackTransactionResponse{
		Success: true,
	}, nil
}

// GetTransaction returns a transaction by ID
func (m *Manager) GetTransaction(ctx context.Context, txnID string) (*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return nil, fmt.Errorf("transaction manager is not started")
	}

	txn, exists := m.transactions[txnID]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	// Update last access time
	txn.LastAccess = time.Now()

	return txn, nil
}

// applyTransactionChanges applies the transaction changes to storage
func (m *Manager) applyTransactionChanges(ctx context.Context, txn *Transaction) error {
	// Apply deletes first
	for key := range txn.deletes {
		if err := m.config.Storage.Delete(ctx, []byte(key)); err != nil {
			return fmt.Errorf("failed to delete key %s: %v", key, err)
		}
	}

	// Apply writes
	for key, value := range txn.writes {
		if err := m.config.Storage.Put(ctx, []byte(key), value); err != nil {
			return fmt.Errorf("failed to write key %s: %v", key, err)
		}
	}

	return nil
}

// cleanupExpiredTransactions runs in background to clean up expired transactions
func (m *Manager) cleanupExpiredTransactions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()

			for id, txn := range m.transactions {
				if now.Sub(txn.StartTime) > txn.Timeout {
					txn.State = TxnAborted
					delete(m.transactions, id)
					log.Printf("Cleaned up expired transaction: %s", id)
				}
			}

			m.mu.Unlock()

		case <-m.stopCh:
			return
		}
	}
}

// TransactionGet reads a value within a transaction
func (m *Manager) TransactionGet(ctx context.Context, txnID string, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txn, exists := m.transactions[txnID]
	if !exists {
		return nil, fmt.Errorf("transaction not found: %s", txnID)
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	// Check if key was deleted in this transaction
	if txn.deletes[key] {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if key was written in this transaction
	if value, exists := txn.writes[key]; exists {
		return value, nil
	}

	// Check if key was already read in this transaction
	if value, exists := txn.reads[key]; exists {
		return value, nil
	}

	// Read from storage
	value, err := m.config.Storage.Get(ctx, []byte(key))
	if err != nil {
		return nil, err
	}

	// Remember this read for consistency
	txn.reads[key] = value
	txn.LastAccess = time.Now()

	return value, nil
}

// TransactionPut writes a value within a transaction
func (m *Manager) TransactionPut(ctx context.Context, txnID string, key string, value []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txn, exists := m.transactions[txnID]
	if !exists {
		return fmt.Errorf("transaction not found: %s", txnID)
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	// Remove from deletes if it was there
	delete(txn.deletes, key)

	// Add to writes
	txn.writes[key] = value
	txn.LastAccess = time.Now()

	return nil
}

// TransactionDelete deletes a key within a transaction
func (m *Manager) TransactionDelete(ctx context.Context, txnID string, key string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txn, exists := m.transactions[txnID]
	if !exists {
		return fmt.Errorf("transaction not found: %s", txnID)
	}

	txn.mu.Lock()
	defer txn.mu.Unlock()

	// Remove from writes if it was there
	delete(txn.writes, key)

	// Add to deletes
	txn.deletes[key] = true
	txn.LastAccess = time.Now()

	return nil
}
