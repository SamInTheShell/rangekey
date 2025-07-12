# RangeDB API Documentation

## Overview
RangeDB is a distributed key-value database with ACID transactions, built on etcd Raft consensus.

## Core Operations

### Key-Value Operations

#### PUT
Store a key-value pair.

```bash
rangedb put <key> <value>
```

Example:
```bash
rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'
```

#### GET
Retrieve a value by key.

```bash
rangedb get <key>
```

Example:
```bash
rangedb get /user/123
```

#### DELETE
Delete a key.

```bash
rangedb delete <key>
```

Example:
```bash
rangedb delete /user/123
```

#### RANGE
Query a range of keys.

```bash
rangedb range <start-key> <end-key>
```

Example:
```bash
rangedb range /user/ /user/z
```

### Batch Operations

#### BATCH PUT
Perform multiple PUT operations atomically.

```bash
rangedb batch put <key1> <value1> [key2 value2 ...]
```

Example:
```bash
rangedb batch put /user/123 '{"name": "John"}' /user/456 '{"name": "Jane"}'
```

### Transaction Operations

#### BEGIN
Start a new transaction.

```bash
rangedb txn begin
```

Returns a transaction ID that can be used for subsequent operations.

#### COMMIT
Commit a transaction.

```bash
rangedb txn commit --id <transaction-id>
```

Example:
```bash
rangedb txn commit --id txn-12345
```

#### ROLLBACK
Rollback a transaction.

```bash
rangedb txn rollback --id <transaction-id>
```

Example:
```bash
rangedb txn rollback --id txn-12345
```

## Administrative Operations

### Cluster Management

#### CLUSTER STATUS
Show cluster information.

```bash
rangedb admin cluster status
```

### Configuration Management

#### CONFIG SET
Set a configuration value.

```bash
rangedb admin config set <key> <value>
```

Example:
```bash
rangedb admin config set _/cluster/config/replication_factor 3
```

#### CONFIG GET
Get a configuration value.

```bash
rangedb admin config get <key>
```

Example:
```bash
rangedb admin config get _/cluster/config/replication_factor
```

### Backup and Restore

#### BACKUP CREATE
Create a backup.

```bash
rangedb admin backup create <path>
```

Example:
```bash
rangedb admin backup create /backups/backup-2025-01-01.db
```

#### BACKUP RESTORE
Restore from a backup.

```bash
rangedb admin backup restore <path>
```

Example:
```bash
rangedb admin backup restore /backups/backup-2025-01-01.db
```

### Metadata Inspection

#### METADATA LIST
List metadata keys.

```bash
rangedb admin metadata list [prefix]
```

Example:
```bash
rangedb admin metadata list _/cluster/
```

#### METADATA GET
Get a metadata value.

```bash
rangedb admin metadata get <key>
```

Example:
```bash
rangedb admin metadata get _/cluster/nodes/node1
```

### Performance Tools

#### BENCHMARK
Run performance benchmarks.

```bash
rangedb admin performance benchmark [--operations 1000] [--concurrency 10]
```

Example:
```bash
rangedb admin performance benchmark --operations 10000 --concurrency 50
```

## gRPC API

RangeDB exposes a gRPC API for programmatic access. The service definition includes:

- **Core Operations**: Get, Put, Delete, Range, Batch
- **Transaction Operations**: BeginTransaction, CommitTransaction, RollbackTransaction
- **Cluster Operations**: GetClusterInfo, GetNodeInfo

### Go Client SDK

```go
import "github.com/samintheshell/rangekey/client"

// Create client
client, err := client.NewClient(&client.Config{
    Address: "localhost:8081",
})

// Connect
err = client.Connect(ctx)

// Basic operations
err = client.Put(ctx, "key", []byte("value"))
value, err := client.Get(ctx, "key")
err = client.Delete(ctx, "key")

// Range queries
results, err := client.Range(ctx, "start", "end", 100)

// Transactions
txn, err := client.BeginTransaction(ctx)
err = txn.Commit(ctx)
```

## Server Configuration

### Command Line Options

```bash
rangedb server [options]
```

Options:
- `--peer-address`: Address for peer communication (default: localhost:8080)
- `--client-address`: Address for client connections (default: localhost:8081)
- `--data-dir`: Directory for data storage (default: ./data)
- `--peers`: List of peer addresses for bootstrapping
- `--join`: List of existing cluster members to join
- `--cluster-init`: Initialize a new cluster
- `--raft-port`: Port for Raft communication (default: 8082)
- `--log-level`: Log level (debug, info, warn, error)

### Configuration Examples

#### Single Node
```bash
rangedb server --cluster-init
```

#### Multi-Node Cluster
```bash
# Node 1
rangedb server --peer-address=node1:8080 --peers=node1:8080,node2:8080,node3:8080 --cluster-init

# Node 2
rangedb server --peer-address=node2:8080 --peers=node1:8080,node2:8080,node3:8080

# Node 3
rangedb server --peer-address=node3:8080 --peers=node1:8080,node2:8080,node3:8080
```

## Data Model

### Key Namespaces

- **User Data**: `/...` - All application keys
- **System Metadata**: `_/...` - System configuration and metadata
  - `_/cluster/nodes/...` - Node registry
  - `_/cluster/config/...` - Cluster configuration
  - `_/partitions/...` - Partition metadata
  - `_/backup/...` - Backup metadata

### Data Types

RangeDB stores raw bytes. Applications are responsible for serialization (JSON, MessagePack, etc.).

## Error Handling

- **Key Not Found**: Returned when attempting to get a non-existent key
- **Transaction Errors**: Returned when transaction operations fail
- **Network Errors**: Connection and communication errors
- **Validation Errors**: Invalid parameters or configuration

## Best Practices

1. **Key Design**: Use hierarchical keys with consistent prefixes
2. **Transactions**: Keep transactions short and focused
3. **Batch Operations**: Use batch operations for multiple related changes
4. **Monitoring**: Monitor cluster status and performance metrics
5. **Backup**: Regular backups for data protection

## Examples

### Complete Application Example

```go
func main() {
    // Create client
    client, err := client.NewClient(&client.Config{
        Address: "localhost:8081",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Connect
    ctx := context.Background()
    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Store user data
    user := `{"name": "John", "email": "john@example.com"}`
    err = client.Put(ctx, "/user/123", []byte(user))
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve user data
    data, err := client.Get(ctx, "/user/123")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", string(data))

    // Transaction example
    txn, err := client.BeginTransaction(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Perform multiple operations in transaction
    // ... transaction operations ...

    err = txn.Commit(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```