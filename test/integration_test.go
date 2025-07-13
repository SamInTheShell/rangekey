package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/server"
	"github.com/samintheshell/rangekey/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerLifecycle(t *testing.T) {
	// Create temporary data directory
	tempDir := t.TempDir()

	// Create server configuration
	cfg := config.DefaultServerConfig()
	cfg.DataDir = tempDir
	cfg.PeerAddress = "localhost:18080"
	cfg.ClientAddress = "localhost:18081"
	cfg.RaftPort = 18082
	cfg.ClusterInit = true

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify server is running
	if !srv.IsRunning() {
		t.Error("Server should be running")
	}

	// Give server time to initialize
	time.Sleep(100 * time.Millisecond)

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	if err := srv.Stop(stopCtx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Verify server is stopped
	if srv.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

func TestStorageEngineBasics(t *testing.T) {
	// Create temporary data directory
	tempDir := t.TempDir()

	// Create storage engine
	cfg := &config.ServerConfig{
		DataDir:           tempDir,
		ClientAddress:     "localhost:18081",
		PeerAddress:       "localhost:18080",
		RaftPort:          18082,
		LogLevel:          "info",
		RequestTimeout:    5 * time.Second,
		HeartbeatTimeout:  1 * time.Second,
		ElectionTimeout:   3 * time.Second,
		ReplicationFactor: 1,
		MaxBatchSize:      1000,
		FlushInterval:     100 * time.Millisecond,
		CompactionLevel:   1,
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		srv.Stop(stopCtx)
	}()

	// Basic functionality test would go here
	// For now, just verify the server started successfully
	if !srv.IsRunning() {
		t.Error("Server should be running for storage tests")
	}
}

func TestMetadataStore(t *testing.T) {
	// Create temporary data directory
	tempDir := t.TempDir()

	// Create server configuration
	cfg := config.DefaultServerConfig()
	cfg.DataDir = tempDir
	cfg.PeerAddress = "localhost:28080"
	cfg.ClientAddress = "localhost:28081"
	cfg.RaftPort = 28082
	cfg.ClusterInit = true

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		srv.Stop(stopCtx)
	}()

	// Verify cluster initialization
	if !srv.IsRunning() {
		t.Error("Server should be running for metadata tests")
	}

	// More detailed metadata tests would go here
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.ServerConfig
		shouldErr bool
	}{
		{
			name: "valid config",
			cfg: &config.ServerConfig{
				PeerAddress:       "localhost:8080",
				ClientAddress:     "localhost:8081",
				DataDir:           "/tmp/test",
				RaftPort:          8082,
				LogLevel:          "info",
				RequestTimeout:    5 * time.Second,
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   3 * time.Second,
				ReplicationFactor: 3,
				MaxBatchSize:      1000,
				FlushInterval:     100 * time.Millisecond,
				CompactionLevel:   1,
			},
			shouldErr: false,
		},
		{
			name: "missing peer address",
			cfg: &config.ServerConfig{
				ClientAddress:     "localhost:8081",
				DataDir:           "/tmp/test",
				RaftPort:          8082,
				LogLevel:          "info",
				RequestTimeout:    5 * time.Second,
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   3 * time.Second,
				ReplicationFactor: 3,
				MaxBatchSize:      1000,
				FlushInterval:     100 * time.Millisecond,
				CompactionLevel:   1,
			},
			shouldErr: true,
		},
		{
			name: "invalid replication factor",
			cfg: &config.ServerConfig{
				PeerAddress:       "localhost:8080",
				ClientAddress:     "localhost:8081",
				DataDir:           "/tmp/test",
				RaftPort:          8082,
				LogLevel:          "info",
				RequestTimeout:    5 * time.Second,
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   3 * time.Second,
				ReplicationFactor: 0,
				MaxBatchSize:      1000,
				FlushInterval:     100 * time.Millisecond,
				CompactionLevel:   1,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.shouldErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	// Setup test environment
	helper := NewTestHelper()
	if !helper.IsInitialized() {
		panic("test helper not initialized")
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestIntegrationBasicOperations(t *testing.T) {
	// Setup test server
	cfg := config.DefaultServerConfig()
	cfg.ClientAddress = "localhost:8091"
	cfg.PeerAddress = "localhost:8090"
	cfg.DataDir = t.TempDir()
	cfg.ClusterInit = true

	srv, err := server.NewServer(cfg)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := srv.Start(ctx)
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Create client
	client, err := client.NewClient(&client.Config{
		Address: "localhost:8091",
	})
	require.NoError(t, err)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close()

	// Test basic operations
	t.Run("PUT and GET", func(t *testing.T) {
		err := client.Put(ctx, "test-key", []byte("test-value"))
		assert.NoError(t, err)

		// Wait for the operation to be committed through Raft
		time.Sleep(1 * time.Second)

		value, err := client.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, []byte("test-value"), value)
	})

	t.Run("DELETE", func(t *testing.T) {
		err := client.Put(ctx, "delete-key", []byte("delete-value"))
		assert.NoError(t, err)

		// Wait for the PUT to be committed
		time.Sleep(1 * time.Second)

		err = client.Delete(ctx, "delete-key")
		assert.NoError(t, err)

		// Wait for the DELETE to be committed
		time.Sleep(1 * time.Second)

		_, err = client.Get(ctx, "delete-key")
		assert.Error(t, err) // Should not exist
	})

	t.Run("RANGE", func(t *testing.T) {
		// Setup range test data
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("range/key_%d", i)
			value := fmt.Sprintf("range_value_%d", i)
			err := client.Put(ctx, key, []byte(value))
			assert.NoError(t, err)
		}

		// Wait for all operations to be committed through Raft
		time.Sleep(2 * time.Second)

		// Test range query
		results, err := client.Range(ctx, "range/key_", "range/key_z", 10)
		assert.NoError(t, err)
		assert.Len(t, results, 5)
	})

	// Stop server
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	err = srv.Stop(shutdownCtx)
	assert.NoError(t, err)
}

func TestIntegrationTransactions(t *testing.T) {
	// Setup test server
	cfg := config.DefaultServerConfig()
	cfg.ClientAddress = "localhost:8093"
	cfg.PeerAddress = "localhost:8092"
	cfg.DataDir = t.TempDir()
	cfg.ClusterInit = true

	srv, err := server.NewServer(cfg)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := srv.Start(ctx)
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Create client
	client, err := client.NewClient(&client.Config{
		Address: "localhost:8093",
	})
	require.NoError(t, err)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close()

	// Test transaction operations
	t.Run("Begin and Commit Transaction", func(t *testing.T) {
		txn, err := client.BeginTransaction(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, txn.ID())

		err = txn.Commit(ctx)
		assert.NoError(t, err)
	})

	t.Run("Begin and Rollback Transaction", func(t *testing.T) {
		txn, err := client.BeginTransaction(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, txn.ID())

		err = txn.Rollback(ctx)
		assert.NoError(t, err)
	})

	// Stop server
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	err = srv.Stop(shutdownCtx)
	assert.NoError(t, err)
}

func TestIntegrationBenchmark(t *testing.T) {
	// Setup test server
	cfg := config.DefaultServerConfig()
	cfg.ClientAddress = "localhost:8095"
	cfg.PeerAddress = "localhost:8094"
	cfg.DataDir = t.TempDir()
	cfg.ClusterInit = true

	srv, err := server.NewServer(cfg)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := srv.Start(ctx)
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Create client
	client, err := client.NewClient(&client.Config{
		Address: "localhost:8095",
	})
	require.NoError(t, err)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close()

	// Run benchmark
	t.Run("Performance Benchmark", func(t *testing.T) {
		operations := 100
		start := time.Now()

		for i := 0; i < operations; i++ {
			key := fmt.Sprintf("benchmark/key_%d", i)
			value := fmt.Sprintf("benchmark_value_%d", i)
			err := client.Put(ctx, key, []byte(value))
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)
		opsPerSec := float64(operations) / elapsed.Seconds()

		t.Logf("Benchmark Results:")
		t.Logf("  Operations: %d", operations)
		t.Logf("  Time: %v", elapsed)
		t.Logf("  Ops/sec: %.2f", opsPerSec)

		// Assert minimum performance
		assert.Greater(t, opsPerSec, 10.0, "Should achieve at least 10 ops/sec")
	})

	// Stop server
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	
	err = srv.Stop(shutdownCtx)
	assert.NoError(t, err)
}
