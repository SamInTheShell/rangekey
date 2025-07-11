package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/server"
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

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}
