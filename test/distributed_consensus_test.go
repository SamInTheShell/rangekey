package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/server"
	"github.com/samintheshell/rangekey/client"
)

// TestDistributedConsensus tests actual leader election and log replication
func TestDistributedConsensus(t *testing.T) {
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
			HeartbeatTimeout: 1 * time.Second,
			ElectionTimeout:  5 * time.Second,
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
			HeartbeatTimeout: 1 * time.Second,
			ElectionTimeout:  5 * time.Second,
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
			HeartbeatTimeout: 1 * time.Second,
			ElectionTimeout:  5 * time.Second,
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

	// Wait for cluster to form and leader election
	time.Sleep(8 * time.Second)

	// Test leader election
	leaderCount := 0
	var leaderServer *server.Server
	for i, srv := range servers {
		if srv.IsLeader() {
			leaderCount++
			leaderServer = srv
			t.Logf("Node %d is the leader", i+1)
		}
	}

	if leaderCount != 1 {
		t.Errorf("Expected 1 leader, got %d", leaderCount)
	}

	if leaderServer == nil {
		t.Fatalf("No leader found")
	}

	// Test distributed writes through the leader
	t.Log("Testing distributed writes...")

	// Create a client to the leader
	leaderAddress := leaderServer.GetConfig().ClientAddress
	rangeClient, err := client.NewClient(&client.Config{
		Address:        leaderAddress,
		RequestTimeout: 5 * time.Second,
		MaxRetries:     3,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer rangeClient.Close()

	// Connect to the server
	if err := rangeClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to leader: %v", err)
	}

	// Test key-value operations
	testKey := "test-distributed-key"
	testValue := []byte("test-distributed-value")

	// PUT operation
	if err := rangeClient.Put(ctx, testKey, testValue); err != nil {
		t.Errorf("Failed to put key: %v", err)
	}

	// GET operation
	value, err := rangeClient.Get(ctx, testKey)
	if err != nil {
		t.Errorf("Failed to get key: %v", err)
	}

	if string(value) != string(testValue) {
		t.Errorf("Expected value %s, got %s", string(testValue), string(value))
	}

	// Verify data is replicated to all nodes
	t.Log("Verifying data replication...")
	time.Sleep(1 * time.Second) // Allow replication to complete

	for i, srv := range servers {
		clientAddr := srv.GetConfig().ClientAddress
		client, err := client.NewClient(&client.Config{
			Address:        clientAddr,
			RequestTimeout: 5 * time.Second,
			MaxRetries:     3,
		})
		if err != nil {
			t.Errorf("Failed to create client for node %d: %v", i+1, err)
			continue
		}

		if err := client.Connect(ctx); err != nil {
			t.Errorf("Failed to connect to node %d: %v", i+1, err)
			client.Close()
			continue
		}

		value, err := client.Get(ctx, testKey)
		if err != nil {
			t.Errorf("Failed to get key from node %d: %v", i+1, err)
		} else if string(value) != string(testValue) {
			t.Errorf("Node %d: Expected value %s, got %s", i+1, string(testValue), string(value))
		} else {
			t.Logf("Node %d: Successfully replicated data", i+1)
		}

		client.Close()
	}

	// Test multiple concurrent writes
	t.Log("Testing concurrent writes...")
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			key := fmt.Sprintf("concurrent-key-%d", idx)
			value := []byte(fmt.Sprintf("concurrent-value-%d", idx))

			if err := rangeClient.Put(ctx, key, value); err != nil {
				t.Errorf("Failed to put concurrent key %d: %v", idx, err)
			}

			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify concurrent writes
	time.Sleep(1 * time.Second) // Allow replication
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("concurrent-key-%d", i)
		expectedValue := []byte(fmt.Sprintf("concurrent-value-%d", i))

		value, err := rangeClient.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get concurrent key %d: %v", i, err)
		} else if string(value) != string(expectedValue) {
			t.Errorf("Concurrent key %d: Expected value %s, got %s", i, string(expectedValue), string(value))
		}
	}

	// Cleanup
	for i, srv := range servers {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("Failed to stop server %d: %v", i+1, err)
		}
	}
	
	// Give servers time to fully shut down and release ports
	time.Sleep(500 * time.Millisecond)

	t.Log("Distributed consensus test completed successfully")
}
