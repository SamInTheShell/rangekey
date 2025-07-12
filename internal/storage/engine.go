package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Config holds the storage engine configuration
type Config struct {
	DataDir         string
	WALDir          string
	MaxBatchSize    int
	FlushInterval   time.Duration
	CompactionLevel int
}

// Engine represents the storage engine
type Engine struct {
	config *Config
	db     *badger.DB
	wal    *WAL

	// Backup manager
	backupManager *BackupManager

	// Lifecycle
	mu       sync.RWMutex
	started  bool
	stopping bool
	stopCh   chan struct{}

	// Background workers
	workers sync.WaitGroup

	// Metrics
	metrics *Metrics
}

// Metrics holds storage engine metrics
type Metrics struct {
	TotalOperations int64
	TotalBytes      int64
	LastFlush       time.Time
	LastCompaction  time.Time
}

// NewEngine creates a new storage engine
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		return nil, fmt.Errorf("storage config is required")
	}

	// Create directories
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.MkdirAll(config.WALDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	// Initialize BadgerDB
	opts := badger.DefaultOptions(config.DataDir)
	opts.Logger = &badgerLogger{}
	opts.SyncWrites = false // We'll use WAL for durability
	opts.CompactL0OnClose = true
	opts.NumVersionsToKeep = 1
	opts.NumGoroutines = 8

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	// Initialize WAL
	wal, err := NewWAL(config.WALDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize WAL: %w", err)
	}

	engine := &Engine{
		config:  config,
		db:      db,
		wal:     wal,
		stopCh:  make(chan struct{}),
		metrics: &Metrics{},
	}

	return engine, nil
}

// Start starts the storage engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started {
		return fmt.Errorf("storage engine is already started")
	}

	log.Println("Starting storage engine...")

	// Start WAL
	if err := e.wal.Start(ctx); err != nil {
		return fmt.Errorf("failed to start WAL: %w", err)
	}

	// Replay WAL if needed
	if err := e.replayWAL(ctx); err != nil {
		return fmt.Errorf("failed to replay WAL: %w", err)
	}

	// Start background workers
	e.startBackgroundWorkers(ctx)

	e.started = true
	log.Println("Storage engine started successfully")

	return nil
}

// Stop stops the storage engine
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started || e.stopping {
		return nil
	}

	e.stopping = true
	close(e.stopCh)

	log.Println("Stopping storage engine...")

	// Stop background workers
	e.workers.Wait()

	// Stop WAL
	if err := e.wal.Stop(ctx); err != nil {
		log.Printf("Error stopping WAL: %v", err)
	}

	// Close database
	if err := e.db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	e.started = false
	e.stopping = false

	log.Println("Storage engine stopped")
	return nil
}

// Get retrieves a value by key
func (e *Engine) Get(ctx context.Context, key []byte) ([]byte, error) {
	if !e.started {
		return nil, fmt.Errorf("storage engine is not started")
	}

	var value []byte
	err := e.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	return value, nil
}

// Put stores a key-value pair
func (e *Engine) Put(ctx context.Context, key, value []byte) error {
	if !e.started {
		return fmt.Errorf("storage engine is not started")
	}

	// Write to WAL first
	entry := &WALEntry{
		Type:      WALEntryTypePut,
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}

	if err := e.wal.Write(entry); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	// Write to storage
	err := e.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})

	if err != nil {
		return fmt.Errorf("failed to put key: %w", err)
	}

	// Update metrics
	e.metrics.TotalOperations++
	e.metrics.TotalBytes += int64(len(key) + len(value))

	return nil
}

// Delete removes a key
func (e *Engine) Delete(ctx context.Context, key []byte) error {
	if !e.started {
		return fmt.Errorf("storage engine is not started")
	}

	// Write to WAL first
	entry := &WALEntry{
		Type:      WALEntryTypeDelete,
		Key:       key,
		Timestamp: time.Now(),
	}

	if err := e.wal.Write(entry); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	// Delete from storage
	err := e.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})

	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	// Update metrics
	e.metrics.TotalOperations++

	return nil
}

// Range retrieves a range of keys
func (e *Engine) Range(ctx context.Context, startKey, endKey []byte, limit int) (map[string][]byte, error) {
	if !e.started {
		return nil, fmt.Errorf("storage engine is not started")
	}

	results := make(map[string][]byte)
	count := 0

	err := e.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(startKey); it.Valid() && count < limit; it.Next() {
			item := it.Item()
			key := item.Key()

			// Check if we've reached the end key
			if endKey != nil && string(key) >= string(endKey) {
				break
			}

			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			results[string(key)] = value
			count++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to perform range query: %w", err)
	}

	return results, nil
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Type  string
	Key   []byte
	Value []byte
}

// Batch performs multiple operations atomically
func (e *Engine) Batch(ctx context.Context, operations []BatchOperation) error {
	if !e.started {
		return fmt.Errorf("storage engine is not started")
	}

	if len(operations) == 0 {
		return nil
	}

	if len(operations) > e.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(operations), e.config.MaxBatchSize)
	}

	// Write all operations to WAL first
	entries := make([]*WALEntry, len(operations))
	for i, op := range operations {
		var entryType WALEntryType
		switch op.Type {
		case "put":
			entryType = WALEntryTypePut
		case "delete":
			entryType = WALEntryTypeDelete
		default:
			return fmt.Errorf("unsupported operation type: %s", op.Type)
		}

		entries[i] = &WALEntry{
			Type:      entryType,
			Key:       op.Key,
			Value:     op.Value,
			Timestamp: time.Now(),
		}
	}

	if err := e.wal.WriteBatch(entries); err != nil {
		return fmt.Errorf("failed to write WAL batch: %w", err)
	}

	// Apply operations to storage
	err := e.db.Update(func(txn *badger.Txn) error {
		for _, op := range operations {
			switch op.Type {
			case "put":
				if err := txn.Set(op.Key, op.Value); err != nil {
					return err
				}
			case "delete":
				if err := txn.Delete(op.Key); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to apply batch operations: %w", err)
	}

	// Update metrics
	e.metrics.TotalOperations += int64(len(operations))
	for _, op := range operations {
		e.metrics.TotalBytes += int64(len(op.Key) + len(op.Value))
	}

	return nil
}

// HealthCheck performs a health check
func (e *Engine) HealthCheck(ctx context.Context) error {
	if !e.started {
		return fmt.Errorf("storage engine is not started")
	}

	// Check if we can read/write
	testKey := []byte("__health_check__")
	testValue := []byte("ok")

	if err := e.Put(ctx, testKey, testValue); err != nil {
		return fmt.Errorf("health check write failed: %w", err)
	}

	value, err := e.Get(ctx, testKey)
	if err != nil {
		return fmt.Errorf("health check read failed: %w", err)
	}

	if string(value) != string(testValue) {
		return fmt.Errorf("health check value mismatch")
	}

	if err := e.Delete(ctx, testKey); err != nil {
		return fmt.Errorf("health check delete failed: %w", err)
	}

	return nil
}

// GetMetrics returns storage engine metrics
func (e *Engine) GetMetrics() *Metrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy of metrics
	return &Metrics{
		TotalOperations: e.metrics.TotalOperations,
		TotalBytes:      e.metrics.TotalBytes,
		LastFlush:       e.metrics.LastFlush,
		LastCompaction:  e.metrics.LastCompaction,
	}
}

// GetAllKeys returns all keys in the storage engine
func (e *Engine) GetAllKeys(ctx context.Context) ([][]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.started {
		return nil, fmt.Errorf("storage engine is not started")
	}

	var keys [][]byte
	err := e.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Only need keys
		iterator := txn.NewIterator(opts)
		defer iterator.Close()

		for iterator.Rewind(); iterator.Valid(); iterator.Next() {
			item := iterator.Item()
			key := item.KeyCopy(nil)
			keys = append(keys, key)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get all keys: %w", err)
	}

	return keys, nil
}

// Backup creates a backup of the storage engine
func (e *Engine) Backup(ctx context.Context, backupPath string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.started {
		return fmt.Errorf("storage engine is not started")
	}

	// Create backup manager if not exists
	if e.backupManager == nil {
		e.backupManager = NewBackupManager(e, nil)
		if err := e.backupManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start backup manager: %w", err)
		}
	}

	// Create full backup
	metadata, err := e.backupManager.CreateFullBackup(ctx, backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	log.Printf("Backup created with ID: %s", metadata.ID)
	return nil
}

// Restore restores the storage engine from a backup
func (e *Engine) Restore(ctx context.Context, backupPath string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.started {
		return fmt.Errorf("cannot restore while storage engine is running")
	}

	// Create backup manager if not exists
	if e.backupManager == nil {
		e.backupManager = NewBackupManager(e, nil)
		if err := e.backupManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start backup manager: %w", err)
		}
	}

	// Restore from backup
	if err := e.backupManager.RestoreFromBackup(ctx, backupPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	log.Printf("Restore initiated from: %s", backupPath)
	return nil
}

// GetBackupMetadata returns metadata about a backup
func (e *Engine) GetBackupMetadata(backupPath string) (*BackupMetadata, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create backup manager if not exists
	if e.backupManager == nil {
		e.backupManager = NewBackupManager(e, nil)
	}

	return e.backupManager.GetBackupMetadata(backupPath)
}

// replayWAL replays WAL entries to recover state
func (e *Engine) replayWAL(ctx context.Context) error {
	log.Println("Replaying WAL...")

	entries, err := e.wal.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read WAL entries: %w", err)
	}

	if len(entries) == 0 {
		log.Println("No WAL entries to replay")
		return nil
	}

	log.Printf("Replaying %d WAL entries", len(entries))

	err = e.db.Update(func(txn *badger.Txn) error {
		for _, entry := range entries {
			switch entry.Type {
			case WALEntryTypePut:
				if err := txn.Set(entry.Key, entry.Value); err != nil {
					return err
				}
			case WALEntryTypeDelete:
				if err := txn.Delete(entry.Key); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to replay WAL entries: %w", err)
	}

	// Clear WAL after successful replay
	if err := e.wal.Clear(); err != nil {
		return fmt.Errorf("failed to clear WAL: %w", err)
	}

	log.Printf("WAL replay completed successfully")
	return nil
}

// startBackgroundWorkers starts background maintenance workers
func (e *Engine) startBackgroundWorkers(ctx context.Context) {
	// Flush worker
	e.workers.Add(1)
	go e.flushWorker(ctx)

	// Compaction worker
	e.workers.Add(1)
	go e.compactionWorker(ctx)
}

// flushWorker periodically flushes data to disk
func (e *Engine) flushWorker(ctx context.Context) {
	defer e.workers.Done()

	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.flush()
		}
	}
}

// compactionWorker periodically compacts the database
func (e *Engine) compactionWorker(ctx context.Context) {
	defer e.workers.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.compact()
		}
	}
}

// flush flushes data to disk
func (e *Engine) flush() {
	if err := e.db.Sync(); err != nil {
		log.Printf("Error flushing database: %v", err)
		return
	}

	if err := e.wal.Flush(); err != nil {
		log.Printf("Error flushing WAL: %v", err)
		return
	}

	e.metrics.LastFlush = time.Now()
}

// compact performs database compaction
func (e *Engine) compact() {
	if err := e.db.Flatten(e.config.CompactionLevel); err != nil {
		log.Printf("Error compacting database: %v", err)
		return
	}

	e.metrics.LastCompaction = time.Now()
}

// Error definitions
var (
	ErrKeyNotFound = fmt.Errorf("key not found")
)

// badgerLogger implements badger.Logger
type badgerLogger struct{}

func (l *badgerLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[BADGER ERROR] "+format, args...)
}

func (l *badgerLogger) Warningf(format string, args ...interface{}) {
	log.Printf("[BADGER WARN] "+format, args...)
}

func (l *badgerLogger) Infof(format string, args ...interface{}) {
	log.Printf("[BADGER INFO] "+format, args...)
}

func (l *badgerLogger) Debugf(format string, args ...interface{}) {
	// Skip debug messages
}
