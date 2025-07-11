# RangeKey Distributed Database - Initial Version Plan

## Project Overview
Build a distributed key-value database using etcd v3 as foundation with multi-raft consensus, automatic data distribution and migration, transactional support, and comprehensive backup/recovery capabilities. Single binary deployment model.

## Core Architecture

### Phase 1: Foundation & Core Components
- [ ] **Project Structure Setup**
  - [ ] Initialize Go module structure
  - [ ] Setup build system and Makefile
  - [ ] Configure CI/CD pipeline
  - [ ] Setup testing framework

- [ ] **Node Manager Implementation**
  - [ ] Node bootstrap and initialization
  - [ ] Multi-peer discovery and bootstrapping
  - [ ] Service coordination and lifecycle management
  - [ ] Metadata store initialization for founding nodes
  - [ ] Cluster membership management and failure detection

- [ ] **Storage Engine Foundation**
  - [ ] Integrate BadgerDB as storage backend
  - [ ] Implement write-ahead logging (WAL)
  - [ ] Create key-value interface abstractions
  - [ ] Add data serialization/deserialization

- [ ] **Metadata Store System**
  - [ ] Implement `_/...` namespace for metadata storage
  - [ ] Create cluster configuration management in metadata
  - [ ] Build node registry and membership tracking
  - [ ] Implement partition metadata storage (`_/partitions/...`)
  - [ ] Add configuration versioning and updates

### Phase 2: Multi-Raft Implementation
- [ ] **Raft Group Management**
  - [ ] Integrate etcd's Raft library
  - [ ] Implement multi-raft group coordination
  - [ ] Create partition-to-raft-group mapping
  - [ ] Handle raft group lifecycle (create/destroy)

- [ ] **Partition System**
  - [ ] Implement range-based partitioning
  - [ ] Create partition metadata management in `_/partitions/...`
  - [ ] Build consistent hashing for initial distribution
  - [ ] Implement partition routing logic
  - [ ] Store partition assignments in metadata store

- [ ] **Consensus & Replication**
  - [ ] Configure Raft consensus per partition
  - [ ] Implement log replication
  - [ ] Handle leader election and failover
  - [ ] Create snapshot mechanism per partition

### Phase 3: Networking & API
- [ ] **Communication Layer**
  - [ ] Implement gRPC server for client API
  - [ ] Create inter-node communication protocol
  - [ ] Design Protocol Buffers schema for all operations
  - [ ] Implement connection pooling and load balancing

- [ ] **Client Interface & CLI**
  - [ ] Integrate urfave CLI v3 for command-line interface
  - [ ] Basic key-value operations via gRPC (GET, PUT, DELETE)
  - [ ] Range query support through gRPC
  - [ ] Batch operations and transactions via gRPC
  - [ ] Client SDK development (gRPC-based)
  - [ ] CLI commands for all database operations

- [ ] **Client SDK & Libraries**
  - [ ] Go client SDK with transaction support
  - [ ] Connection pooling and load balancing in client
  - [ ] Automatic retry and failover logic
  - [ ] Client-side transaction management
  - [ ] Examples and documentation for SDK usage

### Phase 4: Distributed Transactions
- [ ] **Transaction Coordinator**
  - [ ] Implement two-phase commit (2PC) protocol
  - [ ] Create distributed transaction manager
  - [ ] Handle cross-partition transaction coordination
  - [ ] Implement transaction isolation levels

- [ ] **Consistency & Isolation**
  - [ ] Implement timestamp ordering
  - [ ] Add conflict detection and resolution
  - [ ] Create deadlock detection and prevention
  - [ ] Handle transaction timeouts and cleanup

### Phase 5: Auto-Migration & Rebalancing
- [ ] **Partition Management**
  - [ ] Implement automatic partition splitting
  - [ ] Create partition merging logic
  - [ ] Build load monitoring and metrics
  - [ ] Handle partition size thresholds

- [ ] **Data Migration**
  - [ ] Implement consistent data movement
  - [ ] Create migration coordination protocol
  - [ ] Handle migration rollback scenarios
  - [ ] Ensure zero-downtime migrations

- [ ] **Load Balancing**
  - [ ] Monitor partition load and distribution
  - [ ] Implement automatic rebalancing triggers
  - [ ] Create rebalancing algorithms
  - [ ] Handle node addition/removal scenarios
  - [ ] Implement graceful node decommissioning
  - [ ] Data migration during node removal

### Phase 6: Backup & Recovery
- [ ] **Backup System**
  - [ ] Implement full cluster backup
  - [ ] Create incremental backup capability
  - [ ] Add partition-level backup support
  - [ ] Build backup scheduling system

- [ ] **Recovery Mechanisms**
  - [ ] Implement WAL replay for node recovery
  - [ ] Create cluster-wide recovery procedures
  - [ ] Handle partial failure scenarios
  - [ ] Build backup verification and integrity checks

- [ ] **Disaster Recovery**
  - [ ] Implement restore from backup
  - [ ] Create point-in-time recovery
  - [ ] Handle split-brain prevention
  - [ ] Build corruption detection and repair

### Phase 7: Monitoring & Operations
- [ ] **Observability**
  - [ ] Implement metrics collection (Prometheus format)
  - [ ] Add structured logging
  - [ ] Create health check endpoints
  - [ ] Build distributed tracing support

- [ ] **Administration Tools**
  - [ ] Create cluster status and monitoring commands via CLI
  - [ ] Implement configuration management via metadata store
  - [ ] Add performance profiling tools through CLI
  - [ ] Build debugging and diagnostic utilities
  - [ ] Create metadata store inspection tools
  - [ ] CLI-based backup and restore operations
  - [ ] Node join/leave/decommission commands

### Phase 8: Performance & Optimization
- [ ] **Performance Tuning**
  - [ ] Optimize storage layer performance
  - [ ] Implement connection pooling
  - [ ] Add request batching and pipelining
  - [ ] Optimize network serialization

- [ ] **Caching & Optimization**
  - [ ] Implement read caching strategies
  - [ ] Add write batching for better throughput
  - [ ] Optimize memory usage
  - [ ] Create performance benchmarking suite

### Phase 9: Testing & Validation
- [ ] **Unit Testing**
  - [ ] Core component unit tests
  - [ ] Raft integration tests
  - [ ] Transaction system tests
  - [ ] Backup/recovery tests

- [ ] **Integration Testing**
  - [ ] Multi-node cluster tests
  - [ ] Failure scenario testing
  - [ ] Performance and load testing
  - [ ] Chaos engineering tests

- [ ] **System Testing**
  - [ ] End-to-end system validation
  - [ ] Long-running stability tests
  - [ ] Data consistency validation
  - [ ] Backup/restore validation

### Phase 10: Documentation & Deployment
- [ ] **Documentation**
  - [ ] API documentation
  - [ ] Administration guide
  - [ ] Architecture documentation
  - [ ] Performance tuning guide

- [ ] **Deployment & Packaging**
  - [ ] Single binary build system
  - [ ] Docker containerization
  - [ ] Deployment scripts and examples
  - [ ] Configuration templates
  - [ ] Client SDK package distribution (Go modules)
  - [ ] Multi-language client SDK planning

## Technical Specifications

### Core Features
- **Storage**: BadgerDB backend with WAL
- **Consensus**: etcd Raft per partition
- **Networking**: gRPC-only APIs (no HTTP REST)
- **Serialization**: Protocol Buffers
- **CLI**: urfave CLI v3 for all user interactions
- **Monitoring**: Prometheus metrics
- **Logging**: Structured JSON logging

### Deployment Model
- Single binary with embedded services
- Multi-peer bootstrapping for Kubernetes compatibility
- Configuration stored in metadata store (`_/...` namespace)
- Automatic service discovery through metadata
- Graceful shutdown and recovery
- Dynamic cluster membership management

### Key-Value Namespace Design
- **User Data**: `/...` namespace for all application keys
- **Metadata**: `_/...` namespace for system configuration
  - `_/cluster/nodes/...` - Node registry and membership
  - `_/cluster/config/...` - Cluster configuration
  - `_/partitions/...` - Partition metadata and assignments
  - `_/raft/groups/...` - Raft group information
  - `_/backups/...` - Backup metadata and schedules

### Backup Strategy
- Monthly scheduled backups (user configurable)
- Full and incremental backup support
- Point-in-time recovery
- Distributed snapshot coordination

### Recovery Capabilities
- Individual node recovery with WAL replay
- Cluster-wide recovery from total shutdown
- Partition-level recovery
- Split-brain prevention and detection

### Cluster Peering Examples
```bash
# Kubernetes-friendly multi-peer bootstrapping
./rangedb server --peer-address=node1.example.com:8080 \
                 --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080 \
                 --cluster-init

# Any node can bootstrap with any available peer
./rangedb server --peer-address=node2.example.com:8080 \
                 --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080

# New nodes join by connecting to any existing member
./rangedb server --peer-address=node4.example.com:8080 \
                 --join=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080

# All other configuration managed in metadata store
# Example: Update replication factor
./rangedb admin set _/cluster/config/replication_factor 3
```

### CLI Interface Examples
```bash
# Basic key-value operations
./rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'
./rangedb get /user/123
./rangedb delete /user/123

# Range queries
./rangedb range /user/ /user/z

# Transactions
./rangedb txn begin
./rangedb txn put /user/123 '{"name": "John"}'
./rangedb txn put /user/456 '{"name": "Jane"}'
./rangedb txn commit

# Batch operations
./rangedb batch put /user/123 '{"name": "John"}' /user/456 '{"name": "Jane"}'

# Cluster administration
./rangedb admin cluster status
./rangedb admin config set _/cluster/config/replication_factor 3
./rangedb admin backup create /backups/$(date +%Y-%m-%d)
./rangedb admin restore /backups/2025-07-01

# Metadata inspection
./rangedb admin meta list _/partitions/
./rangedb admin meta get _/cluster/nodes/node1
```

### CLI Command Structure
```bash
# Show help and available commands (prevents accidental execution)
./rangedb
./rangedb --help

# Server subcommand for running the database node
./rangedb server --help
./rangedb server --peer-address=node1:8080 --cluster-init

# Client operations
./rangedb get /user/123
./rangedb put /user/123 '{"name": "John"}'
./rangedb delete /user/123
./rangedb range /user/ /user/z

# Transaction operations
./rangedb txn begin
./rangedb txn put /user/123 '{"name": "John"}'
./rangedb txn commit

# Batch operations
./rangedb batch put /user/123 '{"name": "John"}' /user/456 '{"name": "Jane"}'

# Administrative operations
./rangedb admin cluster status
./rangedb admin config set _/cluster/config/replication_factor 3
./rangedb admin backup create /backups/$(date +%Y-%m-%d)
./rangedb admin restore /backups/2025-07-01
./rangedb admin node decommission node3.example.com:8080
./rangedb admin meta list _/partitions/
```

### Safety Features
- **Explicit Server Command**: Prevents accidental server startup
- **Help System**: Running `./rangedb` shows available commands
- **Confirmation Prompts**: Destructive operations require confirmation
- **Dry Run Mode**: Preview operations before execution

### gRPC Protocol Design
- **Transactional Support**: All operations support ACID transactions
- **Streaming**: Support for streaming large results and batch operations
- **Error Handling**: Rich error codes and messages via gRPC status
- **Authentication**: Built-in support for gRPC authentication
- **Compression**: Automatic compression for large payloads
- **Multiplexing**: Single connection for multiple concurrent operations

### Metadata Store Management
- **Bootstrap**: Initial nodes create metadata store structure
- **Configuration**: All runtime config stored in `_/...` namespace
- **Versioning**: Configuration changes tracked with versions
- **Consistency**: Metadata updates use same consensus as user data
- **Separation**: Clear boundary between user (`/...`) and system (`_/...`) data

## Success Criteria
- [ ] Successfully deploy 3-node cluster
- [ ] Demonstrate automatic partition splitting
- [ ] Execute cross-partition transactions
- [ ] Perform backup and restore operations
- [ ] Handle node failures gracefully
- [ ] Achieve target performance benchmarks
- [ ] Validate client SDK with real applications
- [ ] Demonstrate transaction consistency under load

## Timeline Estimate
- **Phase 1-2**: 4-6 weeks (Foundation + Multi-Raft)
- **Phase 3-4**: 4-5 weeks (Networking + Transactions)
- **Phase 5-6**: 5-6 weeks (Auto-Migration + Backup)
- **Phase 7-8**: 3-4 weeks (Monitoring + Performance)
- **Phase 9-10**: 3-4 weeks (Testing + Documentation)

**Total Estimated Timeline**: 19-25 weeks

## Risk Mitigation
- Start with proven libraries (etcd Raft, BadgerDB)
- Implement comprehensive testing at each phase
- Create rollback procedures for all operations
- Build monitoring and alerting from day one
- Document all design decisions and trade-offs

# Comprehensive Cluster Membership and Node Lifecycle Management

## Cluster Membership & Node Lifecycle

#### **Bootstrap Strategy (Kubernetes-Compatible)**
- **Multi-Peer Discovery**: Nodes attempt to connect to multiple peers
- **Bootstrap Quorum**: Any majority of initial peers can form cluster
- **Retry Logic**: Nodes retry connection attempts with exponential backoff
- **Split-Brain Prevention**: Requires majority consensus for cluster formation

#### **Node Failure Recovery**
```bash
# Scenario: node1 goes down, node2 and node3 continue
# When node1 restarts, it reconnects to existing cluster
./rangedb server --peer-address=node1.example.com:8080 \
                 --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080
# Node1 will discover node2/node3 are alive and rejoin
```

#### **Adding New Nodes**
```bash
# Step 1: Start new node pointing to existing cluster members
./rangedb server --peer-address=node4.example.com:8080 \
                 --join=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080

# Step 2: Cluster automatically rebalances partitions to include new node
# Step 3: Monitor rebalancing progress
./rangedb admin cluster status
./rangedb admin partition list --show-migrations
```

#### **Node Decommissioning**
```bash
# Step 1: Initiate graceful decommission (data migration)
./rangedb admin node decommission node3.example.com:8080 --timeout=30m

# Step 2: Monitor migration progress
./rangedb admin node decommission-status node3.example.com:8080

# Step 3: Confirm and remove node from cluster
./rangedb admin node remove node3.example.com:8080 --confirm

# If decommissioning an initial peer:
# - System automatically updates peer lists in metadata store
# - Remaining nodes continue operating
# - New nodes use updated peer list from metadata
```

#### **Disaster Recovery Scenarios**
- **Majority Nodes Lost**: Restore from backup to new cluster
- **All Nodes Lost**: Bootstrap new cluster from most recent backup
- **Peer Node Permanently Lost**: System removes from peer list automatically
- **Network Partition**: Majority partition continues, minority waits for reunion

### Kubernetes Deployment Strategy
```yaml
# StatefulSet with anti-affinity for peer distribution
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: rangedb
spec:
  serviceName: rangedb
  replicas: 3
  template:
    spec:
      containers:
      - name: rangedb
        image: rangedb:latest
        command:
        - ./rangedb
        - server
        - --peer-address=$(HOSTNAME).rangedb.default.svc.cluster.local:8080
        - --peers=rangedb-0.rangedb.default.svc.cluster.local:8080,rangedb-1.rangedb.default.svc.cluster.local:8080,rangedb-2.rangedb.default.svc.cluster.local:8080
        - --cluster-init
        env:
        - name: HOSTNAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
```

### Operational Safeguards
- **Quorum Protection**: Prevent cluster formation without majority
- **Data Replication**: Ensure data survives node failures
- **Graceful Shutdown**: Proper cleanup before node termination
- **Migration Monitoring**: Track data movement during rebalancing
- **Rollback Capability**: Ability to cancel decommissioning if needed

# Client SDK Design and Usage

## Client SDK Overview
The Client SDK provides a high-level interface for applications to interact with the RangeDB database. It abstracts the complexities of gRPC communication, connection management, and error handling, offering a simple and efficient way to perform database operations.

### Client SDK Features
- **Connection Management**: Automatic connection pooling and load balancing
- **Fault Tolerance**: Automatic retry with exponential backoff
- **Transaction Support**: Full ACID transaction capabilities
- **Batch Operations**: Efficient bulk operations
- **Streaming**: Support for large result sets
- **Type Safety**: Strongly typed interfaces with generics
- **Context Support**: Full context.Context integration
- **Monitoring**: Built-in metrics and tracing

## Client SDK Examples

#### **Go Client SDK Usage**
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/samintheshell/rangekey/client"
)

func main() {
    // Connect to RangeDB cluster
    config := &client.Config{
        Endpoints: []string{
            "node1.example.com:8080",
            "node2.example.com:8080", 
            "node3.example.com:8080",
        },
        DialTimeout: 5 * time.Second,
    }
    
    db, err := client.Connect(config)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    ctx := context.Background()
    
    // Basic key-value operations
    err = db.Put(ctx, "/user/123", `{"name": "John", "email": "john@example.com"}`)
    if err != nil {
        log.Fatal(err)
    }
    
    value, err := db.Get(ctx, "/user/123")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", value)
    
    // Range queries
    results, err := db.Range(ctx, "/user/", "/user/z")
    if err != nil {
        log.Fatal(err)
    }
    for key, value := range results {
        fmt.Printf("%s: %s\n", key, value)
    }
    
    // Transactions
    txn := db.NewTransaction()
    
    // Read-write transaction
    err = txn.Begin(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check if user exists
    exists, err := txn.Exists(ctx, "/user/456")
    if err != nil {
        txn.Rollback(ctx)
        log.Fatal(err)
    }
    
    if !exists {
        err = txn.Put(ctx, "/user/456", `{"name": "Jane", "email": "jane@example.com"}`)
        if err != nil {
            txn.Rollback(ctx)
            log.Fatal(err)
        }
        
        // Update counter
        err = txn.Put(ctx, "/stats/user_count", "2")
        if err != nil {
            txn.Rollback(ctx)
            log.Fatal(err)
        }
    }
    
    err = txn.Commit(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Batch operations
    batch := db.NewBatch()
    batch.Put("/user/789", `{"name": "Bob"}`)
    batch.Put("/user/101", `{"name": "Alice"}`)
    batch.Delete("/user/old")
    
    err = batch.Execute(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### **Advanced Transaction Usage**
```go
// Read-only transaction (optimized)
txn := db.NewReadOnlyTransaction()
err := txn.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

// Read multiple keys atomically
user, err := txn.Get(ctx, "/user/123")
if err != nil {
    txn.Rollback(ctx)
    log.Fatal(err)
}

profile, err := txn.Get(ctx, "/profile/123")
if err != nil {
    txn.Rollback(ctx)
    log.Fatal(err)
}

// Read-only transactions don't need explicit commit
txn.Close()

// Conditional updates
txn = db.NewTransaction()
err = txn.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

// Compare-and-swap operation
success, err := txn.CompareAndSwap(ctx, "/counter", "10", "11")
if err != nil {
    txn.Rollback(ctx)
    log.Fatal(err)
}

if success {
    err = txn.Put(ctx, "/last_update", time.Now().Format(time.RFC3339))
    if err != nil {
        txn.Rollback(ctx)
        log.Fatal(err)
    }
}

err = txn.Commit(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Client SDK Architecture
- **gRPC Transport**: Efficient binary protocol with HTTP/2
- **Connection Pooling**: Reuse connections across requests
- **Load Balancing**: Distribute requests across cluster nodes
- **Failure Detection**: Automatic failover to healthy nodes
- **Retry Logic**: Configurable retry policies with backoff
- **Circuit Breaker**: Prevent cascading failures
- **Metrics Integration**: Prometheus metrics for monitoring
- **Distributed Tracing**: OpenTelemetry integration
