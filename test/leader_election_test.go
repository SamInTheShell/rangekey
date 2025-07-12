package test

import (
	"context"
	"testing"
	"time"

	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/server"
)

// TestLeaderElection tests just the leader election process
func TestLeaderElection(t *testing.T) {
	ctx := context.Background()

	// Create temporary directories for each node
	tempDirs := make([]string, 3)
	for i := range tempDirs {
		tempDir := t.TempDir()
		tempDirs[i] = tempDir
	}

	// Node configurations
	nodeConfigs := []*config.ServerConfig{
		{
			PeerAddress:     "localhost:19090",
			ClientAddress:   "localhost:19091",
			RaftPort:        19092,
			DataDir:         tempDirs[0],
			ClusterInit:     true,
			NodeID:          "1",
			Peers:           []string{"localhost:19090", "localhost:20090", "localhost:21090"},
			LogLevel:        "info",
			RequestTimeout:  5 * time.Second,
			HeartbeatTimeout: 500 * time.Millisecond,
			ElectionTimeout:  2 * time.Second,
			ReplicationFactor: 3,
			MaxBatchSize:    1000,
			FlushInterval:   100 * time.Millisecond,
			CompactionLevel: 1,
		},
		{
			PeerAddress:     "localhost:20090",
			ClientAddress:   "localhost:20091",
			RaftPort:        20092,
			DataDir:         tempDirs[1],
			ClusterInit:     false,
			NodeID:          "2",
			Peers:           []string{"localhost:19090", "localhost:20090", "localhost:21090"},
			LogLevel:        "info",
			RequestTimeout:  5 * time.Second,
			HeartbeatTimeout: 500 * time.Millisecond,
			ElectionTimeout:  2 * time.Second,
			ReplicationFactor: 3,
			MaxBatchSize:    1000,
			FlushInterval:   100 * time.Millisecond,
			CompactionLevel: 1,
		},
		{
			PeerAddress:     "localhost:21090",
			ClientAddress:   "localhost:21091",
			RaftPort:        21092,
			DataDir:         tempDirs[2],
			ClusterInit:     false,
			NodeID:          "3",
			Peers:           []string{"localhost:19090", "localhost:20090", "localhost:21090"},
			LogLevel:        "info",
			RequestTimeout:  5 * time.Second,
			HeartbeatTimeout: 500 * time.Millisecond,
			ElectionTimeout:  2 * time.Second,
			ReplicationFactor: 3,
			MaxBatchSize:    1000,
			FlushInterval:   100 * time.Millisecond,
			CompactionLevel: 1,
		},
	}

	// Start all servers
	servers := make([]*server.Server, 3)
	for i, config := range nodeConfigs {
		srv, err := server.NewServer(config)
		if err != nil {
			t.Fatalf("Failed to create server %d: %v", i+1, err)
		}

		if err := srv.Start(ctx); err != nil {
			t.Fatalf("Failed to start server %d: %v", i+1, err)
		}

		servers[i] = srv
		t.Logf("Server %d is running", i+1)
	}

	// Wait for leader election with incremental checks
	for attempt := 0; attempt < 20; attempt++ {
		time.Sleep(1 * time.Second)

		leaderCount := 0
		for i, srv := range servers {
			if srv.IsLeader() {
				leaderCount++
				t.Logf("Attempt %d: Node %d is the leader", attempt+1, i+1)
			}
		}

		if leaderCount == 1 {
			t.Logf("Leader election successful after %d seconds", attempt+1)
			break
		} else if leaderCount > 1 {
			t.Errorf("Multiple leaders found: %d", leaderCount)
			break
		}

		if attempt == 19 {
			t.Errorf("No leader elected after 20 seconds")
		}
	}

	// Cleanup
	for i, srv := range servers {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("Failed to stop server %d: %v", i+1, err)
		}
	}
}
