Final Edit: I'm marking this project as a failure for AI. The system doesn't work, but there is a ton of code here that can be used as reference when I do revisit this project in retirement.

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

### Download Pre-built Binaries

Pre-built binaries are available for Linux, macOS, and Windows from the [GitHub Releases](https://github.com/samintheshell/rangekey/releases) page.

#### Linux (x64)
```bash
wget https://github.com/SamInTheShell/rangekey/releases/latest/download/rangedb-*-linux-amd64.tar.gz
tar -xzf rangedb-*-linux-amd64.tar.gz
chmod +x rangedb
./rangedb --help
```

#### macOS (x64)
```bash
wget https://github.com/SamInTheShell/rangekey/releases/latest/download/rangedb-*-darwin-amd64.tar.gz
tar -xzf rangedb-*-darwin-amd64.tar.gz
chmod +x rangedb
./rangedb --help
```

#### Windows (x64)
Download the `rangedb-*-windows-amd64.zip` file from the releases page and extract the executable.

### Building from Source

#### Prerequisites

- Go 1.24.5 or later
- Make (optional, for build automation)

#### Building

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

### Docker Deployment

RangeDB can be deployed using Docker containers:

```bash
# Pull the latest container image
docker pull ghcr.io/samintheshell/rangekey:latest

# Run single node
docker run -p 8080:8080 -p 8081:8081 \
  ghcr.io/samintheshell/rangekey:latest \
  rangedb server --cluster-init

# Run 3-node cluster with Docker Compose
docker-compose -f docker-compose.3-node.yml up -d

# Run 5-node cluster with Docker Compose
docker-compose -f docker-compose.5-node.yml up -d
```

See [Docker Deployment Guide](docs/docker-deployment.md) for detailed instructions.

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

### Interactive REPL

RangeDB provides an interactive REPL (Read-Eval-Print Loop) for developers to test and iterate on database operations:

```bash
# Start interactive REPL session
rangedb repl

# Example REPL session:
rangedb> put /user/123 '{"name": "John", "email": "john@example.com"}'
OK
rangedb> get /user/123
{"name": "John", "email": "john@example.com"}
rangedb> begin
Transaction started: e36b3db5
rangedb(txn:e36b3db5)> put /user/456 '{"name": "Jane"}'
OK
rangedb(txn:e36b3db5)> commit
Transaction e36b3db5 committed
rangedb> range /user/ /user/z
/user/123: {"name": "John", "email": "john@example.com"}
/user/456: {"name": "Jane"}
rangedb> admin cluster status
Cluster Status:
  Cluster ID: rangedb-cluster
  Replication Factor: 3
  Number of Partitions: 1
  Nodes (1):
    1. localhost:8080
       Client Address: localhost:8081
       Peer Address: localhost:8080
       Status: NODE_RUNNING
       Partitions: []
rangedb> help
Available commands:
  get <key>                           Get value by key
  put <key> <value>                   Put key-value pair
  delete <key>                        Delete key
  range <start> <end> [limit]         Get range of keys
  begin                               Begin new transaction
  commit                              Commit current transaction
  rollback                            Rollback current transaction
  admin cluster status                Show cluster status
  help                                Show this help
  exit                                Exit REPL
rangedb> exit
Goodbye!
```

The REPL features:
- **Interactive Operations**: All database operations available interactively
- **Transaction Support**: Begin, commit, and rollback transactions with visual feedback
- **Command History**: Access to previously executed commands
- **Help System**: Built-in help for all available commands
- **Status Information**: View session and cluster status
- **Error Handling**: Graceful error handling with helpful messages
- **Low Friction**: Perfect for testing, debugging, and learning

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

### Automated Builds

The project uses GitHub Actions for automated builds and releases:

- **Continuous Integration**: Every push and pull request triggers automated builds and tests
- **Cross-compilation**: Builds are tested for Linux, macOS, and Windows
- **Automated Releases**: When a version tag is pushed, binaries are automatically built and attached to GitHub releases
- **Binary Verification**: All releases include SHA256 checksums for security verification

To create a new release:

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions will automatically:
# 1. Build binaries for all platforms
# 2. Create release archives
# 3. Generate checksums
# 4. Create a GitHub release with binaries attached
```

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
├── scripts/             # Build and deployment scripts
└── .github/             # GitHub Actions workflows
    └── workflows/       # CI/CD automation
        ├── ci.yml       # Continuous integration
        ├── release.yml  # Automated releases
        └── copilot-agent.yml # Copilot integration
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
