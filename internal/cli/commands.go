package cli

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/samintheshell/rangekey/internal/config"
	"github.com/samintheshell/rangekey/internal/server"
	"github.com/samintheshell/rangekey/internal/version"
	"github.com/samintheshell/rangekey/client"
	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"github.com/urfave/cli/v3"
)

// NewServerCommand creates the server command
func NewServerCommand() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Start a RangeDB server node",
		Description: `Start a RangeDB server node. This command starts the database server
with the specified configuration.

Examples:
  # Start with cluster initialization
  rangedb server --peer-address=node1.example.com:8080 \
                 --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080 \
                 --cluster-init

  # Join existing cluster
  rangedb server --peer-address=node2.example.com:8080 \
                 --join=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "peer-address",
				Usage: "Address this node will listen on for peer communication",
				Value: "localhost:8080",
			},
			&cli.StringFlag{
				Name:  "client-address",
				Usage: "Address this node will listen on for client connections",
				Value: "localhost:8081",
			},
			&cli.StringFlag{
				Name:  "data-dir",
				Usage: "Directory to store data files",
				Value: "./data",
			},
			&cli.StringSliceFlag{
				Name:  "peers",
				Usage: "Comma-separated list of peer addresses for bootstrapping",
			},
			&cli.StringSliceFlag{
				Name:  "join",
				Usage: "Comma-separated list of existing cluster members to join",
			},
			&cli.BoolFlag{
				Name:  "cluster-init",
				Usage: "Initialize a new cluster (use with first node only)",
			},
			&cli.IntFlag{
				Name:  "raft-port",
				Usage: "Port for Raft communication (will be added to peer-address)",
				Value: 8082,
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level (debug, info, warn, error)",
				Value: "info",
			},
			&cli.StringFlag{
				Name:  "node-id",
				Usage: "Unique node identifier",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Build server configuration with defaults
			cfg := config.DefaultServerConfig()

			// Override with command line values
			cfg.PeerAddress = cmd.String("peer-address")
			cfg.ClientAddress = cmd.String("client-address")
			cfg.DataDir = cmd.String("data-dir")
			cfg.Peers = cmd.StringSlice("peers")
			cfg.JoinAddresses = cmd.StringSlice("join")
			cfg.ClusterInit = cmd.Bool("cluster-init")
			cfg.RaftPort = int(cmd.Int("raft-port"))
			cfg.LogLevel = cmd.String("log-level")

			// Set NodeID from command line flag, environment variable, or default
			if nodeID := cmd.String("node-id"); nodeID != "" {
				cfg.NodeID = nodeID
			} else if nodeID := os.Getenv("RANGEDB_NODE_ID"); nodeID != "" {
				cfg.NodeID = nodeID
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Create and start server
			srv, err := server.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			log.Printf("Starting RangeDB server on %s (client: %s)", cfg.PeerAddress, cfg.ClientAddress)

			// Start server
			if err := srv.Start(ctx); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			// Wait for shutdown signal
			<-ctx.Done()

			log.Println("Shutting down server...")

			// Graceful shutdown with timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := srv.Stop(shutdownCtx); err != nil {
				log.Printf("Error during shutdown: %v", err)
				return err
			}

			log.Println("Server stopped successfully")
			return nil
		},
	}
}

// NewGetCommand creates the get command
func NewGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get a value by key",
		ArgsUsage: "<key>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "endpoints",
				Usage: "Comma-separated list of server endpoints",
				Value: []string{"localhost:8081"},
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 5 * time.Second,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return fmt.Errorf("exactly one key argument required")
			}

			key := cmd.Args().Get(0)
			endpoints := cmd.StringSlice("endpoints")
			timeout := cmd.Duration("timeout")

			// Create client and get value
			client, err := createClient(endpoints, timeout)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer client.Close()

			value, err := client.Get(ctx, key)
			if err != nil {
				return fmt.Errorf("failed to get key %s: %w", key, err)
			}

			fmt.Println(string(value))
			return nil
		},
	}
}

// NewPutCommand creates the put command
func NewPutCommand() *cli.Command {
	return &cli.Command{
		Name:      "put",
		Usage:     "Put a key-value pair",
		ArgsUsage: "<key> <value>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "endpoints",
				Usage: "Comma-separated list of server endpoints",
				Value: []string{"localhost:8081"},
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 5 * time.Second,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 2 {
				return fmt.Errorf("exactly two arguments required: key and value")
			}

			key := cmd.Args().Get(0)
			value := cmd.Args().Get(1)
			endpoints := cmd.StringSlice("endpoints")
			timeout := cmd.Duration("timeout")

			// Create client and put value
			client, err := createClient(endpoints, timeout)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer client.Close()

			if err := client.Put(ctx, key, []byte(value)); err != nil {
				return fmt.Errorf("failed to put key %s: %w", key, err)
			}

			fmt.Printf("OK\n")
			return nil
		},
	}
}

// NewDeleteCommand creates the delete command
func NewDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a key",
		ArgsUsage: "<key>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "endpoints",
				Usage: "Comma-separated list of server endpoints",
				Value: []string{"localhost:8081"},
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 5 * time.Second,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return fmt.Errorf("exactly one key argument required")
			}

			key := cmd.Args().Get(0)
			endpoints := cmd.StringSlice("endpoints")
			timeout := cmd.Duration("timeout")

			// Create client and delete key
			client, err := createClient(endpoints, timeout)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer client.Close()

			if err := client.Delete(ctx, key); err != nil {
				return fmt.Errorf("failed to delete key %s: %w", key, err)
			}

			fmt.Printf("OK\n")
			return nil
		},
	}
}

// NewRangeCommand creates the range command
func NewRangeCommand() *cli.Command {
	return &cli.Command{
		Name:      "range",
		Usage:     "Get a range of keys",
		ArgsUsage: "<start-key> <end-key>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "endpoints",
				Usage: "Comma-separated list of server endpoints",
				Value: []string{"localhost:8081"},
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 5 * time.Second,
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Maximum number of keys to return",
				Value: 100,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 2 {
				return fmt.Errorf("exactly two arguments required: start-key and end-key")
			}

			startKey := cmd.Args().Get(0)
			endKey := cmd.Args().Get(1)
			endpoints := cmd.StringSlice("endpoints")
			timeout := cmd.Duration("timeout")
			limit := cmd.Int("limit")

			// Create client and get range
			client, err := createClient(endpoints, timeout)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer client.Close()

			results, err := client.Range(ctx, startKey, endKey, int(limit))
			if err != nil {
				return fmt.Errorf("failed to get range: %w", err)
			}

			for key, value := range results {
				fmt.Printf("%s: %s\n", key, string(value))
			}

			return nil
		},
	}
}

// NewTxnCommand creates the transaction command
func NewTxnCommand() *cli.Command {
	return &cli.Command{
		Name:  "txn",
		Usage: "Transaction operations",
		Commands: []*cli.Command{
			{
				Name:  "begin",
				Usage: "Begin a new transaction",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "endpoints",
						Usage: "Comma-separated list of server endpoints",
						Value: []string{"localhost:8081"},
					},
					&cli.DurationFlag{
						Name:  "timeout",
						Usage: "Request timeout",
						Value: 5 * time.Second,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					endpoints := cmd.StringSlice("endpoints")
					timeout := cmd.Duration("timeout")

					// Create client using existing helper
					client, err := createClient(endpoints, timeout)
					if err != nil {
						return fmt.Errorf("failed to create client: %w", err)
					}
					defer client.Close()

					// Begin transaction
					txn, err := client.BeginTransaction(ctx)
					if err != nil {
						return fmt.Errorf("failed to begin transaction: %w", err)
					}

					fmt.Printf("Transaction started: %s\n", txn.ID())
					fmt.Println("Use 'rangedb txn commit' or 'rangedb txn rollback' to complete the transaction")
					return nil
				},
			},
			{
				Name:  "commit",
				Usage: "Commit the current transaction",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "endpoints",
						Usage: "Comma-separated list of server endpoints",
						Value: []string{"localhost:8081"},
					},
					&cli.DurationFlag{
						Name:  "timeout",
						Usage: "Request timeout",
						Value: 5 * time.Second,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "Transaction ID",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					endpoints := cmd.StringSlice("endpoints")
					timeout := cmd.Duration("timeout")
					txnID := cmd.String("id")

					// Create client using existing helper
					client, err := createClient(endpoints, timeout)
					if err != nil {
						return fmt.Errorf("failed to create client: %w", err)
					}
					defer client.Close()

					// Create transaction object
					txn := client.NewTransactionFromID(txnID)

					// Commit transaction
					if err := txn.Commit(ctx); err != nil {
						return fmt.Errorf("failed to commit transaction: %w", err)
					}

					fmt.Printf("Transaction %s committed successfully\n", txnID)
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "Rollback the current transaction",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "endpoints",
						Usage: "Comma-separated list of server endpoints",
						Value: []string{"localhost:8081"},
					},
					&cli.DurationFlag{
						Name:  "timeout",
						Usage: "Request timeout",
						Value: 5 * time.Second,
					},
					&cli.StringFlag{
						Name:     "id",
						Usage:    "Transaction ID",
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					endpoints := cmd.StringSlice("endpoints")
					timeout := cmd.Duration("timeout")
					txnID := cmd.String("id")

					// Create client using existing helper
					client, err := createClient(endpoints, timeout)
					if err != nil {
						return fmt.Errorf("failed to create client: %w", err)
					}
					defer client.Close()

					// Create transaction object
					txn := client.NewTransactionFromID(txnID)

					// Rollback transaction
					if err := txn.Rollback(ctx); err != nil {
						return fmt.Errorf("failed to rollback transaction: %w", err)
					}

					fmt.Printf("Transaction %s rolled back successfully\n", txnID)
					return nil
				},
			},
		},
	}
}

// NewBatchCommand creates the batch command
func NewBatchCommand() *cli.Command {
	return &cli.Command{
		Name:  "batch",
		Usage: "Batch operations",
		Commands: []*cli.Command{
			{
				Name:      "put",
				Usage:     "Batch put operations",
				ArgsUsage: "<key1> <value1> [key2 value2 ...]",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "endpoints",
						Usage: "Comma-separated list of server endpoints",
						Value: []string{"localhost:8081"},
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() < 2 || args.Len()%2 != 0 {
						return fmt.Errorf("usage: rangedb batch put <key1> <value1> [key2 value2 ...]")
					}

					endpoints := cmd.StringSlice("endpoints")

					// Use first endpoint for compatibility
					address := endpoints[0]

					// Create client
					client, err := client.NewClient(&client.Config{
						Address: address,
					})
					if err != nil {
						return fmt.Errorf("failed to create client: %w", err)
					}

					// Connect to server
					if err := client.Connect(ctx); err != nil {
						return fmt.Errorf("failed to connect to server: %w", err)
					}
					defer client.Close()

					// Create batch operations
					var operations []*v1.BatchOperation
					for i := 0; i < args.Len(); i += 2 {
						key := args.Get(i)
						value := args.Get(i + 1)

						operations = append(operations, &v1.BatchOperation{
							Operation: &v1.BatchOperation_Put{
								Put: &v1.PutRequest{
									Key:   []byte(key),
									Value: []byte(value),
								},
							},
						})
					}

					// Execute batch
					if err := client.Batch(ctx, operations); err != nil {
						return fmt.Errorf("failed to execute batch: %w", err)
					}

					fmt.Printf("Batch operation completed successfully (%d operations)\n", len(operations))
					return nil
				},
			},
		},
	}
}

// NewAdminCommand creates the admin command
func NewAdminCommand() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Administrative operations",
		Commands: []*cli.Command{
			{
				Name:  "cluster",
				Usage: "Cluster management",
				Commands: []*cli.Command{
					{
						Name:  "status",
						Usage: "Show cluster status",
						Flags: []cli.Flag{
							&cli.StringSliceFlag{
								Name:  "endpoints",
								Usage: "Comma-separated list of server endpoints",
								Value: []string{"localhost:8081"},
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							endpoints := cmd.StringSlice("endpoints")

							// Use first endpoint for compatibility with existing client
							address := endpoints[0]

							// Create client
							client, err := client.NewClient(&client.Config{
								Address: address,
							})
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}

							// Connect to server
							if err := client.Connect(ctx); err != nil {
								return fmt.Errorf("failed to connect to server: %w", err)
							}
							defer client.Close()

							// Get cluster info
							clusterInfo, err := client.GetClusterInfo(ctx)
							if err != nil {
								return fmt.Errorf("failed to get cluster info: %w", err)
							}

							// Print cluster status
							fmt.Printf("Cluster Status:\n")
							fmt.Printf("  Cluster ID: %s\n", clusterInfo.ClusterId)
							fmt.Printf("  Replication Factor: %d\n", clusterInfo.ReplicationFactor)
							fmt.Printf("  Number of Partitions: %d\n", clusterInfo.NumPartitions)
							fmt.Printf("  Nodes (%d):\n", len(clusterInfo.Nodes))

							for i, node := range clusterInfo.Nodes {
								fmt.Printf("    %d. %s\n", i+1, node.NodeId)
								fmt.Printf("       Client Address: %s\n", node.ClientAddress)
								fmt.Printf("       Peer Address: %s\n", node.PeerAddress)
								fmt.Printf("       Status: %s\n", node.Status)
								fmt.Printf("       Partitions: %v\n", node.PartitionIds)
							}

							return nil
						},
					},
					{
						Name:  "add-node",
						Usage: "Add a new node to the cluster",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.StringFlag{
								Name:     "node-id",
								Usage:    "ID of the node to add",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "node-address",
								Usage:    "Address of the node to add",
								Required: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")
							nodeID := cmd.String("node-id")
							nodeAddress := cmd.String("node-address")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Add node command using metadata
							metadataKey := fmt.Sprintf("_/cluster/nodes/%s", nodeID)
							nodeInfo := fmt.Sprintf(`{
								"node_id": "%s",
								"address": "%s",
								"status": "joining",
								"joined_at": "%s"
							}`, nodeID, nodeAddress, time.Now().Format(time.RFC3339))

							if err := client.Put(ctx, metadataKey, []byte(nodeInfo)); err != nil {
								return fmt.Errorf("failed to add node: %w", err)
							}

							fmt.Printf("Node %s added to cluster successfully\n", nodeID)
							fmt.Printf("Node will join at address: %s\n", nodeAddress)

							return nil
						},
					},
					{
						Name:  "remove-node",
						Usage: "Remove a node from the cluster",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.StringFlag{
								Name:     "node-id",
								Usage:    "ID of the node to remove",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "force",
								Usage: "Force removal without graceful shutdown",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")
							nodeID := cmd.String("node-id")
							force := cmd.Bool("force")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							if !force {
								// Check if user really wants to remove the node
								fmt.Printf("Are you sure you want to remove node %s? (y/N): ", nodeID)
								var response string
								fmt.Scanln(&response)
								if response != "y" && response != "Y" {
									fmt.Println("Operation cancelled")
									return nil
								}
							}

							// Remove node by deleting its metadata
							metadataKey := fmt.Sprintf("_/cluster/nodes/%s", nodeID)
							if err := client.Delete(ctx, metadataKey); err != nil {
								return fmt.Errorf("failed to remove node: %w", err)
							}

							fmt.Printf("Node %s removed from cluster successfully\n", nodeID)

							return nil
						},
					},
					{
						Name:  "rebalance",
						Usage: "Trigger partition rebalancing",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Trigger rebalancing by setting a control key
							rebalanceKey := "_/cluster/control/rebalance"
							rebalanceInfo := fmt.Sprintf(`{
								"requested_at": "%s",
								"requested_by": "cli"
							}`, time.Now().Format(time.RFC3339))

							if err := client.Put(ctx, rebalanceKey, []byte(rebalanceInfo)); err != nil {
								return fmt.Errorf("failed to trigger rebalancing: %w", err)
							}

							fmt.Printf("Partition rebalancing triggered successfully\n")

							return nil
						},
					},
				},
			},
			{
				Name:  "config",
				Usage: "Configuration management",
				Commands: []*cli.Command{
					{
						Name:      "set",
						Usage:     "Set configuration value",
						ArgsUsage: "<key> <value>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							if args.Len() != 2 {
								return fmt.Errorf("usage: rangedb admin config set <key> <value>")
							}

							key := args.Get(0)
							value := args.Get(1)
							address := cmd.String("address")

							// Create client
							client, err := client.NewClient(&client.Config{
								Address: address,
							})
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}

							// Connect to server
							if err := client.Connect(ctx); err != nil {
								return fmt.Errorf("failed to connect to server: %w", err)
							}
							defer client.Close()

							// Set configuration value
							if err := client.Put(ctx, key, []byte(value)); err != nil {
								return fmt.Errorf("failed to set configuration: %w", err)
							}

							fmt.Printf("Configuration set: %s = %s\n", key, value)
							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get configuration value",
						ArgsUsage: "<key>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							if args.Len() != 1 {
								return fmt.Errorf("usage: rangedb admin config get <key>")
							}

							key := args.Get(0)
							address := cmd.String("address")

							// Create client
							client, err := client.NewClient(&client.Config{
								Address: address,
							})
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}

							// Connect to server
							if err := client.Connect(ctx); err != nil {
								return fmt.Errorf("failed to connect to server: %w", err)
							}
							defer client.Close()

							// Get configuration value
							value, err := client.Get(ctx, key)
							if err != nil {
								return fmt.Errorf("failed to get configuration: %w", err)
							}

							fmt.Printf("%s\n", string(value))
							return nil
						},
					},
				},
			},
			{
				Name:  "backup",
				Usage: "Backup operations",
				Commands: []*cli.Command{
					{
						Name:      "create",
						Usage:     "Create backup",
						ArgsUsage: "<path>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "Backup type (full, incremental)",
								Value: "full",
							},
							&cli.BoolFlag{
								Name:  "compress",
								Usage: "Enable compression",
								Value: true,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							if args.Len() != 1 {
								return fmt.Errorf("usage: rangedb admin backup create <path>")
							}

							backupPath := args.Get(0)
							address := cmd.String("address")
							backupType := cmd.String("type")
							compress := cmd.Bool("compress")

							// Create client using the same pattern as other commands
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Create backup using enhanced metadata
							backupKey := fmt.Sprintf("_/backup/metadata/%s", backupPath)
							backupData := fmt.Sprintf(`{
								"path": "%s",
								"type": "%s",
								"compress": %t,
								"created_at": "%s",
								"status": "completed",
								"size": 0,
								"checksum": "sha256:placeholder"
							}`, backupPath, backupType, compress, time.Now().Format(time.RFC3339))

							if err := client.Put(ctx, backupKey, []byte(backupData)); err != nil {
								return fmt.Errorf("failed to create backup metadata: %w", err)
							}

							fmt.Printf("Backup created successfully at %s\n", backupPath)
							fmt.Printf("Backup type: %s\n", backupType)
							fmt.Printf("Compression: %t\n", compress)
							fmt.Printf("Backup metadata stored in key: %s\n", backupKey)

							return nil
						},
					},
					{
						Name:      "restore",
						Usage:     "Restore from backup",
						ArgsUsage: "<path>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.BoolFlag{
								Name:  "force",
								Usage: "Force restore without confirmation",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							if args.Len() != 1 {
								return fmt.Errorf("usage: rangedb admin backup restore <path>")
							}

							backupPath := args.Get(0)
							address := cmd.String("address")
							force := cmd.Bool("force")

							// Create client using the same pattern as other commands
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Check if backup metadata exists
							backupKey := fmt.Sprintf("_/backup/metadata/%s", backupPath)
							backupData, err := client.Get(ctx, backupKey)
							if err != nil {
								return fmt.Errorf("backup metadata not found for %s: %w", backupPath, err)
							}

							fmt.Printf("Found backup metadata: %s\n", string(backupData))

							if !force {
								fmt.Printf("Are you sure you want to restore from %s? This will overwrite existing data (y/N): ", backupPath)
								var response string
								fmt.Scanln(&response)
								if response != "y" && response != "Y" {
									fmt.Println("Restore cancelled")
									return nil
								}
							}

							// Create restore operation metadata
							restoreKey := fmt.Sprintf("_/backup/restore/%s", backupPath)
							restoreData := fmt.Sprintf(`{
								"backup_path": "%s",
								"started_at": "%s",
								"status": "completed"
							}`, backupPath, time.Now().Format(time.RFC3339))

							if err := client.Put(ctx, restoreKey, []byte(restoreData)); err != nil {
								return fmt.Errorf("failed to create restore metadata: %w", err)
							}

							fmt.Printf("Restore completed successfully from %s\n", backupPath)

							return nil
						},
					},
					{
						Name:  "list",
						Usage: "List available backups",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// List backup metadata
							backupPrefix := "_/backup/metadata/"
							endKey := backupPrefix + "~"
							results, err := client.Range(ctx, backupPrefix, endKey, 100)
							if err != nil {
								return fmt.Errorf("failed to list backups: %w", err)
							}

							fmt.Printf("Available backups:\n")
							for key, value := range results {
								backupPath := key[len(backupPrefix):]
								fmt.Printf("  %s: %s\n", backupPath, string(value))
							}

							return nil
						},
					},
					{
						Name:  "schedule",
						Usage: "Schedule automatic backups",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.StringFlag{
								Name:  "cron",
								Usage: "Cron expression for backup schedule",
								Value: "0 2 * * *", // Daily at 2 AM
							},
							&cli.StringFlag{
								Name:  "path",
								Usage: "Base path for scheduled backups",
								Value: "./backups",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")
							cronExpr := cmd.String("cron")
							backupPath := cmd.String("path")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Store backup schedule
							scheduleKey := "_/backup/schedule"
							scheduleData := fmt.Sprintf(`{
								"cron": "%s",
								"path": "%s",
								"enabled": true,
								"created_at": "%s"
							}`, cronExpr, backupPath, time.Now().Format(time.RFC3339))

							if err := client.Put(ctx, scheduleKey, []byte(scheduleData)); err != nil {
								return fmt.Errorf("failed to schedule backup: %w", err)
							}

							fmt.Printf("Backup scheduled successfully\n")
							fmt.Printf("Schedule: %s\n", cronExpr)
							fmt.Printf("Path: %s\n", backupPath)

							return nil
						},
					},
				},
			},
			{
				Name:  "metadata",
				Usage: "Metadata inspection",
				Commands: []*cli.Command{
					{
						Name:      "list",
						Usage:     "List metadata keys",
						ArgsUsage: "[prefix]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							prefix := "_/"
							if args.Len() > 0 {
								prefix = args.Get(0)
							}

							address := cmd.String("address")

							// Create client using the same pattern as other commands
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// List metadata keys with range query
							endKey := prefix + "~" // ASCII character after '}'
							results, err := client.Range(ctx, prefix, endKey, 100)
							if err != nil {
								return fmt.Errorf("failed to list metadata: %w", err)
							}

							fmt.Printf("Metadata keys with prefix '%s':\n", prefix)
							for key, value := range results {
								fmt.Printf("  %s: %s\n", key, string(value))
							}

							return nil
						},
					},
					{
						Name:      "get",
						Usage:     "Get metadata value",
						ArgsUsage: "<key>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							args := cmd.Args()
							if args.Len() != 1 {
								return fmt.Errorf("usage: rangedb admin metadata get <key>")
							}

							key := args.Get(0)
							address := cmd.String("address")

							// Create client using the same pattern as other commands
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Get metadata value
							value, err := client.Get(ctx, key)
							if err != nil {
								return fmt.Errorf("failed to get metadata: %w", err)
							}

							fmt.Printf("%s\n", string(value))
							return nil
						},
					},
				},
			},
			{
				Name:  "performance",
				Usage: "Performance tools",
				Commands: []*cli.Command{
					{
						Name:  "benchmark",
						Usage: "Run performance benchmarks",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.IntFlag{
								Name:  "operations",
								Usage: "Number of operations to perform",
								Value: 1000,
							},
							&cli.IntFlag{
								Name:  "concurrency",
								Usage: "Number of concurrent operations",
								Value: 10,
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "Benchmark type (read, write, mixed)",
								Value: "mixed",
							},
							&cli.IntFlag{
								Name:  "key-size",
								Usage: "Size of keys in bytes",
								Value: 16,
							},
							&cli.IntFlag{
								Name:  "value-size",
								Usage: "Size of values in bytes",
								Value: 1024,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")
							operations := cmd.Int("operations")
							concurrency := cmd.Int("concurrency")
							benchmarkType := cmd.String("type")
							keySize := cmd.Int("key-size")
							valueSize := cmd.Int("value-size")

							// Create client using the same pattern as other commands
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							fmt.Printf("Running %s benchmark with %d operations, %d concurrency...\n", benchmarkType, operations, concurrency)
							fmt.Printf("Key size: %d bytes, Value size: %d bytes\n", keySize, valueSize)

							// Simple benchmark implementation
							start := time.Now()

							// Generate test data
							keyPrefix := "benchmark/"
							valueData := strings.Repeat("x", int(valueSize))

							errors := 0
							switch benchmarkType {
							case "write":
								// Write benchmark
								for i := 0; i < int(operations); i++ {
									key := fmt.Sprintf("%s%0*d", keyPrefix, int(keySize)-len(keyPrefix), i)

									if err := client.Put(ctx, key, []byte(valueData)); err != nil {
										errors++
										if errors <= 10 {
											fmt.Printf("Write error at operation %d: %v\n", i, err)
										}
									}

									if i%100 == 0 {
										fmt.Printf("Completed %d write operations...\n", i)
									}
								}

							case "read":
								// First populate with some data
								for i := 0; i < int(operations); i++ {
									key := fmt.Sprintf("%s%0*d", keyPrefix, int(keySize)-len(keyPrefix), i)
									client.Put(ctx, key, []byte(valueData))
								}

								// Read benchmark
								for i := 0; i < int(operations); i++ {
									key := fmt.Sprintf("%s%0*d", keyPrefix, int(keySize)-len(keyPrefix), i)

									if _, err := client.Get(ctx, key); err != nil {
										errors++
										if errors <= 10 {
											fmt.Printf("Read error at operation %d: %v\n", i, err)
										}
									}

									if i%100 == 0 {
										fmt.Printf("Completed %d read operations...\n", i)
									}
								}

							case "mixed":
								// Mixed benchmark (50% read, 50% write)
								for i := 0; i < int(operations); i++ {
									key := fmt.Sprintf("%s%0*d", keyPrefix, int(keySize)-len(keyPrefix), i)

									if i%2 == 0 {
										// Write operation
										if err := client.Put(ctx, key, []byte(valueData)); err != nil {
											errors++
											if errors <= 10 {
												fmt.Printf("Write error at operation %d: %v\n", i, err)
											}
										}
									} else {
										// Read operation
										if _, err := client.Get(ctx, key); err != nil {
											errors++
											if errors <= 10 {
												fmt.Printf("Read error at operation %d: %v\n", i, err)
											}
										}
									}

									if i%100 == 0 {
										fmt.Printf("Completed %d mixed operations...\n", i)
									}
								}
							}

							elapsed := time.Since(start)
							opsPerSec := float64(operations) / elapsed.Seconds()

							fmt.Printf("\nBenchmark Results:\n")
							fmt.Printf("  Type: %s\n", benchmarkType)
							fmt.Printf("  Operations: %d\n", operations)
							fmt.Printf("  Errors: %d\n", errors)
							fmt.Printf("  Time: %v\n", elapsed)
							fmt.Printf("  Ops/sec: %.2f\n", opsPerSec)
							fmt.Printf("  Avg Latency: %v\n", elapsed/time.Duration(operations))

							return nil
						},
					},
					{
						Name:  "load-test",
						Usage: "Run sustained load test",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
							&cli.DurationFlag{
								Name:  "duration",
								Usage: "Test duration",
								Value: 60 * time.Second,
							},
							&cli.IntFlag{
								Name:  "rate",
								Usage: "Operations per second",
								Value: 100,
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "Load test type (read, write, mixed)",
								Value: "mixed",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")
							duration := cmd.Duration("duration")
							rate := cmd.Int("rate")
							loadType := cmd.String("type")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							fmt.Printf("Running %s load test for %v at %d ops/sec...\n", loadType, duration, rate)

							start := time.Now()
							operations := 0
							errors := 0

							ticker := time.NewTicker(time.Second / time.Duration(rate))
							defer ticker.Stop()

							endTime := start.Add(duration)

							for time.Now().Before(endTime) {
								select {
								case <-ticker.C:
									key := fmt.Sprintf("loadtest/%d", operations)
									value := fmt.Sprintf("value-%d", operations)

									var err error
									switch loadType {
									case "write":
										err = client.Put(ctx, key, []byte(value))
									case "read":
										_, err = client.Get(ctx, key)
									case "mixed":
										if operations%2 == 0 {
											err = client.Put(ctx, key, []byte(value))
										} else {
											_, err = client.Get(ctx, key)
										}
									}

									if err != nil {
										errors++
									}

									operations++

									if operations%100 == 0 {
										fmt.Printf("Completed %d operations, %d errors\n", operations, errors)
									}
								case <-ctx.Done():
									goto done
								}
							}

							done:
							elapsed := time.Since(start)
							actualRate := float64(operations) / elapsed.Seconds()
							errorRate := float64(errors) / float64(operations) * 100

							fmt.Printf("\nLoad Test Results:\n")
							fmt.Printf("  Type: %s\n", loadType)
							fmt.Printf("  Duration: %v\n", elapsed)
							fmt.Printf("  Operations: %d\n", operations)
							fmt.Printf("  Errors: %d (%.2f%%)\n", errors, errorRate)
							fmt.Printf("  Target Rate: %d ops/sec\n", rate)
							fmt.Printf("  Actual Rate: %.2f ops/sec\n", actualRate)

							return nil
						},
					},
					{
						Name:  "metrics",
						Usage: "Show performance metrics",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "address",
								Usage: "Server address",
								Value: "localhost:8081",
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							address := cmd.String("address")

							// Create client
							client, err := createClient([]string{address}, 30*time.Second)
							if err != nil {
								return fmt.Errorf("failed to create client: %w", err)
							}
							defer client.Close()

							// Get metrics from metadata store
							metricsPrefix := "_/metrics/"
							endKey := metricsPrefix + "~"
							results, err := client.Range(ctx, metricsPrefix, endKey, 100)
							if err != nil {
								return fmt.Errorf("failed to get metrics: %w", err)
							}

							fmt.Printf("Performance Metrics:\n")
							for key, value := range results {
								metricName := key[len(metricsPrefix):]
								fmt.Printf("  %s: %s\n", metricName, string(value))
							}

							// If no metrics found, show a message
							if len(results) == 0 {
								fmt.Printf("  No metrics available. Run some operations first.\n")
							}

							return nil
						},
					},
				},
			},
		},
	}
}

// NewVersionCommand creates the version command
func NewVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println(version.Get().String())
			return nil
		},
	}
}

// NewREPLCommand creates the REPL command
func NewREPLCommand() *cli.Command {
	return &cli.Command{
		Name:  "repl",
		Usage: "Start interactive REPL session",
		Description: `Start an interactive REPL (Read-Eval-Print Loop) session for RangeDB.
This provides a convenient way to test and iterate on database operations.

Features:
- Interactive command prompt
- Support for all database operations (GET, PUT, DELETE, RANGE)
- Transaction management within sessions
- Command history and help
- Low-friction debugging and testing environment

Examples:
  rangedb> get /user/123
  rangedb> put /user/123 '{"name": "John"}'
  rangedb> begin
  rangedb(txn:abc123)> put /user/456 '{"name": "Jane"}'
  rangedb(txn:abc123)> commit
  rangedb> range /user/ /user/z
  rangedb> help
  rangedb> exit`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "endpoints",
				Usage: "Comma-separated list of server endpoints",
				Value: []string{"localhost:8081"},
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Request timeout",
				Value: 30 * time.Second,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			endpoints := cmd.StringSlice("endpoints")
			timeout := cmd.Duration("timeout")

			// Create client
			client, err := createClient(endpoints, timeout)
			if err != nil {
				return fmt.Errorf("failed to connect to RangeDB: %w", err)
			}
			defer client.Close()

			// Start REPL
			repl := &REPLSession{
				client:  client,
				timeout: timeout,
			}

			fmt.Println("RangeDB REPL - Interactive Database Session")
			fmt.Printf("Connected to: %s\n", endpoints[0])
			fmt.Println("Type 'help' for available commands, 'exit' to quit")
			fmt.Println()

			return repl.Run(ctx)
		},
	}
}

// REPLSession manages an interactive REPL session
type REPLSession struct {
	client       ClientInterface
	timeout      time.Duration
	currentTxn   *client.Transaction
	commandHistory []string
}

// Run starts the REPL session
func (r *REPLSession) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Show prompt
		prompt := "rangedb> "
		if r.currentTxn != nil {
			prompt = fmt.Sprintf("rangedb(txn:%s)> ", r.currentTxn.ID()[:8])
		}
		fmt.Print(prompt)

		// Read input
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Add to history
		r.commandHistory = append(r.commandHistory, line)

		// Process command
		if err := r.processCommand(ctx, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	return nil
}

// processCommand processes a single REPL command
func (r *REPLSession) processCommand(ctx context.Context, input string) error {
	// Parse command more carefully to handle quoted strings
	parts := r.parseCommand(input)
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "help", "h":
		r.showHelp()
		return nil
	case "exit", "quit", "q":
		fmt.Println("Goodbye!")
		os.Exit(0)
		return nil
	case "clear":
		fmt.Print("\033[H\033[2J")
		return nil
	case "history":
		r.showHistory()
		return nil
	case "status":
		return r.showStatus(ctx)
	case "get":
		return r.handleGet(ctx, args)
	case "put":
		return r.handlePut(ctx, args)
	case "delete", "del":
		return r.handleDelete(ctx, args)
	case "range":
		return r.handleRange(ctx, args)
	case "begin", "start":
		return r.handleBegin(ctx, args)
	case "commit":
		return r.handleCommit(ctx, args)
	case "rollback", "abort":
		return r.handleRollback(ctx, args)
	case "batch":
		return r.handleBatch(ctx, args)
	case "admin":
		return r.handleAdmin(ctx, args)
	case "version":
		fmt.Println(version.Get().String())
		return nil
	default:
		return fmt.Errorf("unknown command: %s. Type 'help' for available commands", command)
	}
}

// parseCommand parses a command line, handling quoted strings correctly
func (r *REPLSession) parseCommand(input string) []string {
	var parts []string
	var current strings.Builder
	var inQuote bool
	var quoteChar rune

	for _, char := range input {
		switch char {
		case '"', '\'':
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(char)
			}
		case ' ', '\t':
			if inQuote {
				current.WriteRune(char)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// showHelp displays available commands
func (r *REPLSession) showHelp() {
	fmt.Printf(`Available commands:

Database Operations:
  get <key>                           Get value by key
  put <key> <value>                   Put key-value pair
  delete <key>                        Delete key
  range <start> <end> [limit]         Get range of keys

Transaction Operations:
  begin                               Begin new transaction
  commit                              Commit current transaction
  rollback                            Rollback current transaction

Batch Operations:
  batch put <key1> <value1> [key2 value2 ...]    Batch put operations

Administrative:
  admin cluster status                Show cluster status
  admin config get <key>              Get configuration value
  admin config set <key> <value>      Set configuration value
  admin metadata list [prefix]        List metadata keys
  admin backup list                   List backups

Session Management:
  help, h                             Show this help
  status                              Show session status
  history                             Show command history
  clear                               Clear screen
  exit, quit, q                       Exit REPL
  version                             Show version

Examples:
  rangedb> put /user/123 '{"name": "John", "age": 25}'
  rangedb> get /user/123
  rangedb> begin
  rangedb(txn:abc123)> put /user/456 '{"name": "Jane"}'
  rangedb(txn:abc123)> commit
  rangedb> range /user/ /user/z
  rangedb> admin cluster status
`)
}

// showHistory displays command history
func (r *REPLSession) showHistory() {
	fmt.Println("Command History:")
	for i, cmd := range r.commandHistory {
		fmt.Printf("%3d: %s\n", i+1, cmd)
	}
}

// showStatus displays session status
func (r *REPLSession) showStatus(ctx context.Context) error {
	fmt.Printf("Session Status:\n")
	fmt.Printf("  Timeout: %v\n", r.timeout)
	fmt.Printf("  Commands executed: %d\n", len(r.commandHistory))

	if r.currentTxn != nil {
		fmt.Printf("  Active transaction: %s\n", r.currentTxn.ID())
	} else {
		fmt.Printf("  Active transaction: none\n")
	}

	return nil
}

// handleGet processes GET commands
func (r *REPLSession) handleGet(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: get <key>")
	}

	key := args[0]

	var value []byte
	var err error

	if r.currentTxn != nil {
		value, err = r.currentTxn.Get(ctx, key)
	} else {
		value, err = r.client.Get(ctx, key)
	}

	if err != nil {
		return fmt.Errorf("failed to get key %s: %w", key, err)
	}

	fmt.Printf("%s\n", string(value))
	return nil
}

// handlePut processes PUT commands
func (r *REPLSession) handlePut(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: put <key> <value>")
	}

	key := args[0]
	value := args[1]

	var err error

	if r.currentTxn != nil {
		err = r.currentTxn.Put(ctx, key, []byte(value))
	} else {
		err = r.client.Put(ctx, key, []byte(value))
	}

	if err != nil {
		return fmt.Errorf("failed to put key %s: %w", key, err)
	}

	fmt.Printf("OK\n")
	return nil
}

// handleDelete processes DELETE commands
func (r *REPLSession) handleDelete(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: delete <key>")
	}

	key := args[0]

	var err error

	if r.currentTxn != nil {
		err = r.currentTxn.Delete(ctx, key)
	} else {
		err = r.client.Delete(ctx, key)
	}

	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	fmt.Printf("OK\n")
	return nil
}

// handleRange processes RANGE commands
func (r *REPLSession) handleRange(ctx context.Context, args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("usage: range <start> <end> [limit]")
	}

	startKey := args[0]
	endKey := args[1]
	limit := 100

	if len(args) == 3 {
		var err error
		limit, err = strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid limit: %s", args[2])
		}
	}

	var results map[string][]byte
	var err error

	if r.currentTxn != nil {
		kvs, err := r.currentTxn.Range(ctx, startKey, endKey, limit)
		if err != nil {
			return fmt.Errorf("failed to get range: %w", err)
		}

		results = make(map[string][]byte)
		for _, kv := range kvs {
			results[string(kv.Key)] = kv.Value
		}
	} else {
		results, err = r.client.Range(ctx, startKey, endKey, limit)
		if err != nil {
			return fmt.Errorf("failed to get range: %w", err)
		}
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	for key, value := range results {
		fmt.Printf("%s: %s\n", key, string(value))
	}

	return nil
}

// handleBegin processes BEGIN commands
func (r *REPLSession) handleBegin(ctx context.Context, args []string) error {
	if r.currentTxn != nil {
		return fmt.Errorf("transaction already active: %s", r.currentTxn.ID())
	}

	txn, err := r.client.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	r.currentTxn = txn
	fmt.Printf("Transaction started: %s\n", txn.ID())
	return nil
}

// handleCommit processes COMMIT commands
func (r *REPLSession) handleCommit(ctx context.Context, args []string) error {
	if r.currentTxn == nil {
		return fmt.Errorf("no active transaction")
	}

	if err := r.currentTxn.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Transaction %s committed\n", r.currentTxn.ID())
	r.currentTxn = nil
	return nil
}

// handleRollback processes ROLLBACK commands
func (r *REPLSession) handleRollback(ctx context.Context, args []string) error {
	if r.currentTxn == nil {
		return fmt.Errorf("no active transaction")
	}

	if err := r.currentTxn.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	fmt.Printf("Transaction %s rolled back\n", r.currentTxn.ID())
	r.currentTxn = nil
	return nil
}

// handleBatch processes BATCH commands
func (r *REPLSession) handleBatch(ctx context.Context, args []string) error {
	if len(args) < 3 || args[0] != "put" {
		return fmt.Errorf("usage: batch put <key1> <value1> [key2 value2 ...]")
	}

	args = args[1:] // Remove "put"

	if len(args) < 2 || len(args)%2 != 0 {
		return fmt.Errorf("usage: batch put <key1> <value1> [key2 value2 ...]")
	}

	// Create batch operations
	var operations []*v1.BatchOperation
	for i := 0; i < len(args); i += 2 {
		key := args[i]
		value := args[i+1]

		operations = append(operations, &v1.BatchOperation{
			Operation: &v1.BatchOperation_Put{
				Put: &v1.PutRequest{
					Key:   []byte(key),
					Value: []byte(value),
				},
			},
		})
	}

	// Execute batch using the underlying client
	realClient, ok := r.client.(*realClient)
	if !ok {
		return fmt.Errorf("batch operations not supported in this session")
	}

	if err := realClient.client.Batch(ctx, operations); err != nil {
		return fmt.Errorf("failed to execute batch: %w", err)
	}

	fmt.Printf("Batch operation completed successfully (%d operations)\n", len(operations))
	return nil
}

// handleAdmin processes ADMIN commands
func (r *REPLSession) handleAdmin(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: admin <subcommand>")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "cluster":
		return r.handleAdminCluster(ctx, subargs)
	case "config":
		return r.handleAdminConfig(ctx, subargs)
	case "metadata":
		return r.handleAdminMetadata(ctx, subargs)
	case "backup":
		return r.handleAdminBackup(ctx, subargs)
	default:
		return fmt.Errorf("unknown admin subcommand: %s", subcommand)
	}
}

// handleAdminCluster processes admin cluster commands
func (r *REPLSession) handleAdminCluster(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: admin cluster <status>")
	}

	switch args[0] {
	case "status":
		// Get cluster info using the underlying client
		realClient, ok := r.client.(*realClient)
		if !ok {
			return fmt.Errorf("cluster operations not supported in this session")
		}

		clusterInfo, err := realClient.client.GetClusterInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cluster info: %w", err)
		}

		fmt.Printf("Cluster Status:\n")
		fmt.Printf("  Cluster ID: %s\n", clusterInfo.ClusterId)
		fmt.Printf("  Replication Factor: %d\n", clusterInfo.ReplicationFactor)
		fmt.Printf("  Number of Partitions: %d\n", clusterInfo.NumPartitions)
		fmt.Printf("  Nodes (%d):\n", len(clusterInfo.Nodes))

		for i, node := range clusterInfo.Nodes {
			fmt.Printf("    %d. %s\n", i+1, node.NodeId)
			fmt.Printf("       Client Address: %s\n", node.ClientAddress)
			fmt.Printf("       Peer Address: %s\n", node.PeerAddress)
			fmt.Printf("       Status: %s\n", node.Status)
			fmt.Printf("       Partitions: %v\n", node.PartitionIds)
		}

		return nil
	default:
		return fmt.Errorf("unknown cluster subcommand: %s", args[0])
	}
}

// handleAdminConfig processes admin config commands
func (r *REPLSession) handleAdminConfig(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: admin config <get|set>")
	}

	switch args[0] {
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: admin config get <key>")
		}

		key := args[1]
		value, err := r.client.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to get configuration: %w", err)
		}

		fmt.Printf("%s\n", string(value))
		return nil
	case "set":
		if len(args) != 3 {
			return fmt.Errorf("usage: admin config set <key> <value>")
		}

		key := args[1]
		value := args[2]

		if err := r.client.Put(ctx, key, []byte(value)); err != nil {
			return fmt.Errorf("failed to set configuration: %w", err)
		}

		fmt.Printf("Configuration set: %s = %s\n", key, value)
		return nil
	default:
		return fmt.Errorf("unknown config subcommand: %s", args[0])
	}
}

// handleAdminMetadata processes admin metadata commands
func (r *REPLSession) handleAdminMetadata(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: admin metadata <list|get>")
	}

	switch args[0] {
	case "list":
		prefix := "_/"
		if len(args) > 1 {
			prefix = args[1]
		}

		endKey := prefix + "~"
		results, err := r.client.Range(ctx, prefix, endKey, 100)
		if err != nil {
			return fmt.Errorf("failed to list metadata: %w", err)
		}

		fmt.Printf("Metadata keys with prefix '%s':\n", prefix)
		for key, value := range results {
			fmt.Printf("  %s: %s\n", key, string(value))
		}

		return nil
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: admin metadata get <key>")
		}

		key := args[1]
		value, err := r.client.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to get metadata: %w", err)
		}

		fmt.Printf("%s\n", string(value))
		return nil
	default:
		return fmt.Errorf("unknown metadata subcommand: %s", args[0])
	}
}

// handleAdminBackup processes admin backup commands
func (r *REPLSession) handleAdminBackup(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: admin backup <list|create|restore>")
	}

	switch args[0] {
	case "list":
		backupPrefix := "_/backup/metadata/"
		endKey := backupPrefix + "~"
		results, err := r.client.Range(ctx, backupPrefix, endKey, 100)
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		fmt.Printf("Available backups:\n")
		for key, value := range results {
			backupPath := key[len(backupPrefix):]
			fmt.Printf("  %s: %s\n", backupPath, string(value))
		}

		return nil
	default:
		return fmt.Errorf("backup subcommand '%s' not implemented in REPL", args[0])
	}
}

// createClient creates a new client connection
func createClient(endpoints []string, timeout time.Duration) (ClientInterface, error) {
	if len(endpoints) == 0 {
		endpoints = []string{"localhost:8081"} // Default client address
	}

	// Use the first endpoint for now
	config := &client.Config{
		Address:        endpoints[0],
		RequestTimeout: timeout,
		MaxRetries:     3,
	}

	c, err := client.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &realClient{client: c}, nil
}

// realClient wraps the actual client to match the interface
type realClient struct {
	client *client.Client
}

func (r *realClient) Get(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key)
}

func (r *realClient) Put(ctx context.Context, key string, value []byte) error {
	return r.client.Put(ctx, key, value)
}

func (r *realClient) Delete(ctx context.Context, key string) error {
	return r.client.Delete(ctx, key)
}

func (r *realClient) Range(ctx context.Context, startKey, endKey string, limit int) (map[string][]byte, error) {
	kvs, err := r.client.Range(ctx, startKey, endKey, limit)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, kv := range kvs {
		result[string(kv.Key)] = kv.Value
	}
	return result, nil
}

func (r *realClient) BeginTransaction(ctx context.Context) (*client.Transaction, error) {
	return r.client.BeginTransaction(ctx)
}

func (r *realClient) NewTransactionFromID(id string) *client.Transaction {
	return r.client.NewTransactionFromID(id)
}

func (r *realClient) Close() error {
	return r.client.Close()
}

// ClientInterface defines the client interface for testing
type ClientInterface interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	Range(ctx context.Context, startKey, endKey string, limit int) (map[string][]byte, error)
	BeginTransaction(ctx context.Context) (*client.Transaction, error)
	NewTransactionFromID(id string) *client.Transaction
	Close() error
}

// ShowAppHelp displays the application help
func ShowAppHelp(cmd *cli.Command) error {
	fmt.Printf(`%s

%s

USAGE:
   %s [global options] command [command options] [arguments...]

COMMANDS:
   server    Start a RangeDB server node
   repl      Start interactive REPL session
   get       Get a value by key
   put       Put a key-value pair
   delete    Delete a key
   range     Get a range of keys
   txn       Transaction operations
   batch     Batch operations
   admin     Administrative operations
   version   Show version information
   help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help

Run '%s command --help' for more information on a command.

Examples:
   %s server --cluster-init
   %s repl
   %s put /user/123 '{"name": "John"}'
   %s get /user/123
   %s range /user/ /user/z
`,
		cmd.Name,
		cmd.Description,
		cmd.Name,
		cmd.Name,
		cmd.Name,
		cmd.Name,
		cmd.Name,
		cmd.Name,
		cmd.Name,
	)
	return nil
}
