# Docker Deployment Guide

This guide explains how to deploy RangeDB using Docker containers.

## Prerequisites

- Docker Engine 20.10+ 
- Docker Compose 2.0+

## Container Images

RangeDB container images are available on GitHub Container Registry:

```bash
# Pull the latest version
docker pull ghcr.io/samintheshell/rangekey:v1.0.0

# Or pull a specific version
docker pull ghcr.io/samintheshell/rangekey:v1.0.0
```

## Quick Start - Single Node

```bash
# Run a single node cluster
docker run -p 8080:8080 -p 8081:8081 \
  --name rangedb-single \
  ghcr.io/samintheshell/rangekey:latest \
  rangedb server --cluster-init

# Test the deployment
docker exec rangedb-single rangedb put /test/key "test value"
docker exec rangedb-single rangedb get /test/key
```

## Multi-Node Clusters

### 3-Node Cluster

```bash
# Start a 3-node cluster
docker-compose -f docker-compose.3-node.yml up -d

# Check cluster status
docker exec rangedb-1 rangedb admin cluster status

# Test operations
docker exec rangedb-1 rangedb put /user/123 '{"name": "John", "email": "john@example.com"}'
docker exec rangedb-1 rangedb get /user/123
```

### 5-Node Cluster

```bash
# Start a 5-node cluster
docker-compose -f docker-compose.5-node.yml up -d

# Check cluster status
docker exec rangedb-1 rangedb admin cluster status

# Test operations
docker exec rangedb-1 rangedb put /user/456 '{"name": "Jane", "email": "jane@example.com"}'
docker exec rangedb-1 rangedb get /user/456
```

## Port Mapping

The Docker Compose configurations use the following port mappings:

### 3-Node Cluster
- rangedb-1: 8080 (peer), 8081 (client)
- rangedb-2: 8082 (peer), 8083 (client)
- rangedb-3: 8084 (peer), 8085 (client)

### 5-Node Cluster
- rangedb-1: 8080 (peer), 8081 (client)
- rangedb-2: 8082 (peer), 8083 (client)
- rangedb-3: 8084 (peer), 8085 (client)
- rangedb-4: 8086 (peer), 8087 (client)
- rangedb-5: 8088 (peer), 8089 (client)

## Data Persistence

Each node stores data in a Docker volume:
- rangedb_data_1, rangedb_data_2, etc.

To backup data:
```bash
# Create backup
docker exec rangedb-1 rangedb admin backup create /data/backup-$(date +%Y%m%d)

# Copy backup from container
docker cp rangedb-1:/data/backup-20241201 ./backup-20241201
```

## Health Checks

The containers include health checks that verify the node is responsive:

```bash
# Check health status
docker ps
docker inspect rangedb-1 | grep -A 10 "Health"
```

## Troubleshooting

### View logs
```bash
# View logs for a specific node
docker logs rangedb-1

# Follow logs
docker logs -f rangedb-1
```

### Connect to container
```bash
# Connect to running container
docker exec -it rangedb-1 /bin/sh

# Run commands inside container
docker exec rangedb-1 rangedb admin cluster status
```

### Reset cluster
```bash
# Stop all containers
docker-compose -f docker-compose.3-node.yml down

# Remove volumes (WARNING: This deletes all data)
docker volume prune

# Start fresh cluster
docker-compose -f docker-compose.3-node.yml up -d
```

## Configuration

### Environment Variables

- `RANGEDB_NODE_ID`: Unique identifier for the node
- `RANGEDB_CLUSTER_SIZE`: Total number of nodes in the cluster

### Custom Configuration

You can customize the configuration by mounting a config file:

```yaml
volumes:
  - ./config.yml:/config.yml
command: rangedb server --config /config.yml
```

## Production Considerations

1. **Data Persistence**: Use named volumes or bind mounts for production
2. **Resource Limits**: Set appropriate CPU and memory limits
3. **Network**: Use a custom network for better isolation
4. **Backup**: Implement regular backup procedures
5. **Monitoring**: Add monitoring and alerting
6. **Security**: Use proper firewall rules and authentication

## Building Custom Images

To build your own container image:

```bash
# Build the image
docker build -t rangedb:custom .

# Test the image
docker run --rm rangedb:custom rangedb --help
```