package transaction

import (
	"context"
	"testing"
	"time"

	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"github.com/samintheshell/rangekey/internal/storage"
)

func TestTransactionManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create storage engine
	storageEngine, err := storage.NewEngine(&storage.Config{
		DataDir:       tempDir,
		WALDir:        tempDir + "/wal",
		FlushInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}

	// Start storage engine
	ctx := context.Background()
	if err := storageEngine.Start(ctx); err != nil {
		t.Fatalf("Failed to start storage engine: %v", err)
	}
	defer storageEngine.Stop(ctx)

	// Create transaction manager
	txnManager, err := NewManager(&Config{
		DefaultTimeout:  5 * time.Second,
		MaxTransactions: 10,
		Storage:        storageEngine,
	})
	if err != nil {
		t.Fatalf("Failed to create transaction manager: %v", err)
	}

	// Start transaction manager
	if err := txnManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start transaction manager: %v", err)
	}
	defer txnManager.Stop(ctx)

	// Test basic transaction operations
	t.Run("BeginTransaction", func(t *testing.T) {
		req := &v1.BeginTransactionRequest{
			TimeoutMs: func() *int64 { ms := int64(5000); return &ms }(),
		}

		resp, err := txnManager.BeginTransaction(ctx, req)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		if resp.TransactionId == "" {
			t.Error("Transaction ID should not be empty")
		}

		if resp.Timestamp == 0 {
			t.Error("Transaction timestamp should not be zero")
		}

		// Rollback the transaction
		rollbackReq := &v1.RollbackTransactionRequest{
			TransactionId: resp.TransactionId,
		}

		rollbackResp, err := txnManager.RollbackTransaction(ctx, rollbackReq)
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		if !rollbackResp.Success {
			t.Error("Rollback should succeed")
		}
	})

	t.Run("CommitTransaction", func(t *testing.T) {
		// Begin transaction
		req := &v1.BeginTransactionRequest{}
		resp, err := txnManager.BeginTransaction(ctx, req)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		txnID := resp.TransactionId

		// Write some data in the transaction
		if err := txnManager.TransactionPut(ctx, txnID, "test-key", []byte("test-value")); err != nil {
			t.Fatalf("Failed to put in transaction: %v", err)
		}

		// Read back the data
		value, err := txnManager.TransactionGet(ctx, txnID, "test-key")
		if err != nil {
			t.Fatalf("Failed to get from transaction: %v", err)
		}

		if string(value) != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", string(value))
		}

		// Commit the transaction
		commitReq := &v1.CommitTransactionRequest{
			TransactionId: txnID,
		}

		commitResp, err := txnManager.CommitTransaction(ctx, commitReq)
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		if !commitResp.Success {
			t.Error("Commit should succeed")
		}

		// Verify the data is persisted
		persistedValue, err := storageEngine.Get(ctx, []byte("test-key"))
		if err != nil {
			t.Fatalf("Failed to get persisted value: %v", err)
		}

		if string(persistedValue) != "test-value" {
			t.Errorf("Expected persisted value 'test-value', got '%s'", string(persistedValue))
		}
	})

	t.Run("TransactionDelete", func(t *testing.T) {
		// First, put some data
		if err := storageEngine.Put(ctx, []byte("delete-key"), []byte("delete-value")); err != nil {
			t.Fatalf("Failed to put initial data: %v", err)
		}

		// Begin transaction
		req := &v1.BeginTransactionRequest{}
		resp, err := txnManager.BeginTransaction(ctx, req)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		txnID := resp.TransactionId

		// Delete the key in the transaction
		if err := txnManager.TransactionDelete(ctx, txnID, "delete-key"); err != nil {
			t.Fatalf("Failed to delete in transaction: %v", err)
		}

		// Try to read the deleted key - should fail
		_, err = txnManager.TransactionGet(ctx, txnID, "delete-key")
		if err == nil {
			t.Error("Expected error when reading deleted key")
		}

		// Commit the transaction
		commitReq := &v1.CommitTransactionRequest{
			TransactionId: txnID,
		}

		commitResp, err := txnManager.CommitTransaction(ctx, commitReq)
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		if !commitResp.Success {
			t.Error("Commit should succeed")
		}

		// Verify the key is deleted from storage
		_, err = storageEngine.Get(ctx, []byte("delete-key"))
		if err == nil {
			t.Error("Expected error when reading deleted key from storage")
		}
	})
}
