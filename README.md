# RangeDB

A distributed key-value database built with Go, featuring automatic partitioning, multi-raft consensus, ACID transactions, and high availability.

## Features

- **Distributed Architecture**: Multi-node cluster with automatic failover
- **Multi-Raft Consensus**: Each partition uses its own Raft group for optimal performance
- **ACID Transactions**: Full transactional support with strong consistency
- **Auto-Partitioning**: Automatic data distribution and rebalancing
- **High Availability**: Configurable replication factor and automatic recovery
- **gRPC API**: Efficient binary protocol with streaming support
- **Single Binary**: Easy deployment with embedded services
- **CLI Interface**: Comprehensive command-line tool for all operations

## Quick Start

### Prerequisites

- Go 1.24.2 or later
- Make (optional, for build automation)

### Building

```bash
# Clone the repository
git clone https://github.com/samintheshell/rangekey.git
cd rangekey

# Build the project
make build

# Or build manually
go build -o build/rangedb ./cmd/rangedb
```

### Running a Single Node

```bash
# Start a single node cluster
./build/rangedb server --cluster-init

# In another terminal, try some operations
./build/rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'
./build/rangedb get /user/123
./build/rangedb range /user/ /user/z
```

### Running a 3-Node Cluster

```bash
# Node 1 (initialize cluster)
./build/rangedb server --peer-address=localhost:8080 \
                       --client-address=localhost:8081 \
                       --peers=localhost:8080,localhost:8082,localhost:8084 \
                       --cluster-init

# Node 2 (join cluster)
./build/rangedb server --peer-address=localhost:8082 \
                       --client-address=localhost:8083 \
                       --join=localhost:8080,localhost:8082,localhost:8084

# Node 3 (join cluster)
./build/rangedb server --peer-address=localhost:8084 \
                       --client-address=localhost:8085 \
                       --join=localhost:8080,localhost:8082,localhost:8084
```

## Command-Line Interface

### Basic Operations

```bash
# Put a key-value pair
rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'

# Get a value
rangedb get /user/123

# Delete a key
rangedb delete /user/123

# Range query
rangedb range /user/ /user/z --limit 100
```

### Transactions

```bash
# Begin a transaction
rangedb txn begin

# Perform operations within transaction
rangedb txn put /user/123 '{"name": "John"}'
rangedb txn put /user/456 '{"name": "Jane"}'

# Commit the transaction
rangedb txn commit
```

### Batch Operations

```bash
# Batch put operations
rangedb batch put /user/123 '{"name": "John"}' /user/456 '{"name": "Jane"}'
```

### Administrative Commands

```bash
# Cluster status
rangedb admin cluster status

# Configuration management
rangedb admin config set _/cluster/config/replication_factor 3
rangedb admin config get _/cluster/config/replication_factor

# Backup operations
rangedb admin backup create /backups/$(date +%Y-%m-%d)
rangedb admin backup restore /backups/2025-07-01

# Metadata inspection
rangedb admin meta list _/partitions/
rangedb admin meta get _/cluster/nodes/node1
```

## Architecture

### Core Components

- **Storage Engine**: BadgerDB with Write-Ahead Logging (WAL)
- **Consensus**: etcd Raft per partition for distributed consensus
- **Partitioning**: Range-based partitioning with automatic splitting/merging
- **Networking**: gRPC for all client and inter-node communication
- **Metadata Store**: Dedicated storage for cluster metadata in `_/...` namespace

### Key-Value Namespace

- **User Data**: `/...` namespace for all application data
- **Metadata**: `_/...` namespace for system configuration
  - `_/cluster/nodes/...` - Node registry and membership
  - `_/cluster/config/...` - Cluster configuration
  - `_/partitions/...` - Partition metadata and assignments
  - `_/raft/groups/...` - Raft group information
  - `_/backups/...` - Backup metadata and schedules

### Deployment Model

RangeDB is designed for easy deployment:

- **Single Binary**: All services embedded in one executable
- **Multi-Peer Bootstrapping**: Kubernetes-compatible cluster formation
- **Configuration in Metadata**: Runtime config stored in `_/...` namespace
- **Automatic Discovery**: Nodes discover each other through metadata
- **Graceful Operations**: Shutdown, recovery, and node management

## Development

### Development Setup

```bash
# Install development tools
make dev-setup

# Run with hot reload
make dev

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format and lint code
make fmt
make vet
make lint
```

### Project Structure

```
rangekey/
├── cmd/rangedb/          # Main application entry point
├── internal/             # Private application code
│   ├── cli/             # Command-line interface
│   ├── config/          # Configuration management
│   ├── server/          # Server implementation
│   ├── storage/         # Storage engine (BadgerDB + WAL)
│   ├── raft/            # Raft consensus implementation
│   ├── metadata/        # Metadata store
│   ├── partition/       # Partition management
│   └── grpc/            # gRPC server
├── pkg/                 # Public API packages
├── client/              # Client SDK
├── api/                 # gRPC proto definitions
├── test/                # Integration tests
└── scripts/             # Build and deployment scripts
```

### Testing

```bash
# Run all tests
make test

# Run tests with race detection
go test -race ./...

# Run integration tests
go test -tags=integration ./test/...

# Run benchmarks
make bench
```

## Performance

RangeDB is designed for high performance:

- **Concurrent Operations**: Parallel processing across partitions
- **Efficient Storage**: BadgerDB with LSM-tree structure
- **Minimal Serialization**: Protocol Buffers for binary efficiency
- **Connection Pooling**: Reuse connections for reduced latency
- **Batch Operations**: Reduce network overhead with batching

## Monitoring

RangeDB provides comprehensive monitoring:

- **Metrics**: Prometheus-compatible metrics endpoint
- **Logging**: Structured JSON logging with configurable levels
- **Health Checks**: Built-in health and readiness checks
- **Tracing**: OpenTelemetry support for distributed tracing

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run the full test suite
6. Submit a pull request

## License

RangeDB is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Status

This project is under active development. The current implementation includes:

- ✅ Basic project structure and CLI
- ✅ Storage engine with BadgerDB and WAL
- ✅ Metadata store implementation
- ✅ Basic server framework
- ✅ gRPC server foundation
- ✅ Partition management structure
- ✅ Build system and tooling

### Coming Soon

- [ ] etcd Raft integration
- [ ] Multi-raft coordination
- [ ] Transaction support
- [ ] Automatic partitioning
- [ ] Data migration
- [ ] Client SDK
- [ ] Backup and recovery
- [ ] Performance optimization

See [.github/developer-plans/InitialVersion.md](.github/developer-plans/InitialVersion.md) for the complete development roadmap.
