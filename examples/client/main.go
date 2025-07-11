package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samintheshell/rangekey/client"
)

func main() {
	// Create a client
	c, err := client.NewClient(&client.Config{
		Address:        "localhost:8080",
		RequestTimeout: 10 * time.Second,
		MaxRetries:     3,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect to the server
	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer c.Close()

	// Example 1: Basic key-value operations
	fmt.Println("=== Basic Key-Value Operations ===")

	// Put a key-value pair
	if err := c.Put(ctx, "hello", []byte("world")); err != nil {
		log.Printf("Failed to put key: %v", err)
	} else {
		fmt.Println("Put: hello -> world")
	}

	// Get the value
	value, err := c.Get(ctx, "hello")
	if err != nil {
		log.Printf("Failed to get key: %v", err)
	} else {
		fmt.Printf("Get: hello -> %s\n", string(value))
	}

	// Put more keys
	c.Put(ctx, "key1", []byte("value1"))
	c.Put(ctx, "key2", []byte("value2"))
	c.Put(ctx, "key3", []byte("value3"))

	// Example 2: Range query
	fmt.Println("\n=== Range Query ===")
	results, err := c.Range(ctx, "key", "keyz", 10)
	if err != nil {
		log.Printf("Failed to range query: %v", err)
	} else {
		for _, kv := range results {
			fmt.Printf("Range result: %s -> %s\n", kv.Key, string(kv.Value))
		}
	}

	// Example 3: Transactions
	fmt.Println("\n=== Transaction Example ===")

	// Begin transaction
	txn, err := c.BeginTransaction(ctx)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
	} else {
		fmt.Println("Started transaction")

		// In a real transaction, you would use transaction-specific operations
		// For now, we just demonstrate the transaction lifecycle

		// Commit the transaction
		if err := txn.Commit(ctx); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
		} else {
			fmt.Println("Transaction committed")
		}
	}

	// Example 4: Cluster information
	fmt.Println("\n=== Cluster Information ===")

	clusterInfo, err := c.GetClusterInfo(ctx)
	if err != nil {
		log.Printf("Failed to get cluster info: %v", err)
	} else {
		fmt.Printf("Cluster ID: %s\n", clusterInfo.ClusterId)
		fmt.Printf("Replication Factor: %d\n", clusterInfo.ReplicationFactor)
		fmt.Printf("Number of Partitions: %d\n", clusterInfo.NumPartitions)
		fmt.Printf("Nodes: %d\n", len(clusterInfo.Nodes))
	}

	// Example 5: Node information
	nodeInfo, err := c.GetNodeInfo(ctx)
	if err != nil {
		log.Printf("Failed to get node info: %v", err)
	} else {
		fmt.Printf("Node ID: %s\n", nodeInfo.NodeId)
		fmt.Printf("Peer Address: %s\n", nodeInfo.PeerAddress)
		fmt.Printf("Client Address: %s\n", nodeInfo.ClientAddress)
	}

	// Clean up
	c.Delete(ctx, "hello")
	c.Delete(ctx, "key1")
	c.Delete(ctx, "key2")
	c.Delete(ctx, "key3")
	fmt.Println("\nCleaned up test keys")
}
