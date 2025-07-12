# RangeDB Deployment Guide

## Overview
This guide covers deployment strategies for RangeDB, from single-node development setups to production multi-node clusters.

## Quick Start

### Single Node Deployment

1. **Build RangeDB**
   ```bash
   git clone https://github.com/samintheshell/rangekey
   cd rangekey
   make build
   ```

2. **Start Server**
   ```bash
   ./bin/rangedb server --cluster-init
   ```

3. **Test Connection**
   ```bash
   ./bin/rangedb put test-key "Hello, RangeDB!"
   ./bin/rangedb get test-key
   ```

## Production Deployment

### Multi-Node Cluster

#### Prerequisites
- 3+ nodes for high availability
- Network connectivity between nodes
- Persistent storage for each node

#### Configuration

1. **Node 1 (Leader)**
   ```bash
   ./bin/rangedb server \
     --peer-address=node1.example.com:8080 \
     --client-address=node1.example.com:8081 \
     --data-dir=/var/lib/rangedb \
     --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080 \
     --cluster-init
   ```

2. **Node 2**
   ```bash
   ./bin/rangedb server \
     --peer-address=node2.example.com:8080 \
     --client-address=node2.example.com:8081 \
     --data-dir=/var/lib/rangedb \
     --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080
   ```

3. **Node 3**
   ```bash
   ./bin/rangedb server \
     --peer-address=node3.example.com:8080 \
     --client-address=node3.example.com:8081 \
     --data-dir=/var/lib/rangedb \
     --peers=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080
   ```

### Docker Deployment

#### Single Node
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o rangedb ./cmd/rangedb

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rangedb .
EXPOSE 8080 8081 8082
CMD ["./rangedb", "server", "--cluster-init"]
```

#### Docker Compose
```yaml
version: '3.8'
services:
  rangedb-1:
    build: .
    ports:
      - "8081:8081"
      - "8080:8080"
    volumes:
      - rangedb-1-data:/var/lib/rangedb
    command: >
      ./rangedb server
      --peer-address=rangedb-1:8080
      --client-address=rangedb-1:8081
      --data-dir=/var/lib/rangedb
      --peers=rangedb-1:8080,rangedb-2:8080,rangedb-3:8080
      --cluster-init
    networks:
      - rangedb-network

  rangedb-2:
    build: .
    ports:
      - "8082:8081"
      - "8083:8080"
    volumes:
      - rangedb-2-data:/var/lib/rangedb
    command: >
      ./rangedb server
      --peer-address=rangedb-2:8080
      --client-address=rangedb-2:8081
      --data-dir=/var/lib/rangedb
      --peers=rangedb-1:8080,rangedb-2:8080,rangedb-3:8080
    networks:
      - rangedb-network
    depends_on:
      - rangedb-1

  rangedb-3:
    build: .
    ports:
      - "8084:8081"
      - "8085:8080"
    volumes:
      - rangedb-3-data:/var/lib/rangedb
    command: >
      ./rangedb server
      --peer-address=rangedb-3:8080
      --client-address=rangedb-3:8081
      --data-dir=/var/lib/rangedb
      --peers=rangedb-1:8080,rangedb-2:8080,rangedb-3:8080
    networks:
      - rangedb-network
    depends_on:
      - rangedb-1

volumes:
  rangedb-1-data:
  rangedb-2-data:
  rangedb-3-data:

networks:
  rangedb-network:
    driver: bridge
```

### Kubernetes Deployment

#### ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rangedb-config
data:
  cluster-peers: "rangedb-0.rangedb.default.svc.cluster.local:8080,rangedb-1.rangedb.default.svc.cluster.local:8080,rangedb-2.rangedb.default.svc.cluster.local:8080"
```

#### StatefulSet
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: rangedb
spec:
  serviceName: rangedb
  replicas: 3
  selector:
    matchLabels:
      app: rangedb
  template:
    metadata:
      labels:
        app: rangedb
    spec:
      containers:
      - name: rangedb
        image: rangedb:latest
        ports:
        - containerPort: 8080
          name: peer
        - containerPort: 8081
          name: client
        - containerPort: 8082
          name: raft
        env:
        - name: HOSTNAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: CLUSTER_PEERS
          valueFrom:
            configMapKeyRef:
              name: rangedb-config
              key: cluster-peers
        command:
        - "./rangedb"
        - "server"
        - "--peer-address=$(HOSTNAME).rangedb.default.svc.cluster.local:8080"
        - "--client-address=$(HOSTNAME).rangedb.default.svc.cluster.local:8081"
        - "--data-dir=/var/lib/rangedb"
        - "--peers=$(CLUSTER_PEERS)"
        - "--cluster-init"
        volumeMounts:
        - name: data
          mountPath: /var/lib/rangedb
        livenessProbe:
          exec:
            command:
            - "./rangedb"
            - "admin"
            - "cluster"
            - "status"
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - "./rangedb"
            - "admin"
            - "cluster"
            - "status"
          initialDelaySeconds: 5
          periodSeconds: 5
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - rangedb
            topologyKey: kubernetes.io/hostname
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

#### Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: rangedb
spec:
  ports:
  - port: 8080
    name: peer
  - port: 8081
    name: client
  - port: 8082
    name: raft
  clusterIP: None
  selector:
    app: rangedb
---
apiVersion: v1
kind: Service
metadata:
  name: rangedb-client
spec:
  ports:
  - port: 8081
    name: client
  selector:
    app: rangedb
  type: LoadBalancer
```

## Configuration Management

### Environment Variables
```bash
export RANGEDB_PEER_ADDRESS=localhost:8080
export RANGEDB_CLIENT_ADDRESS=localhost:8081
export RANGEDB_DATA_DIR=/var/lib/rangedb
export RANGEDB_LOG_LEVEL=info
```

### Configuration File
```yaml
# rangedb.yaml
server:
  peer_address: "localhost:8080"
  client_address: "localhost:8081"
  data_dir: "/var/lib/rangedb"
  log_level: "info"
  
cluster:
  replication_factor: 3
  peers:
    - "node1.example.com:8080"
    - "node2.example.com:8080"
    - "node3.example.com:8080"
    
storage:
  max_batch_size: 1000
  flush_interval: "100ms"
  compaction_level: 1
```

## Monitoring and Observability

### Health Checks
```bash
# Check cluster status
./rangedb admin cluster status

# Check node health
curl -f http://localhost:8081/health || exit 1
```

### Metrics Collection
RangeDB exposes Prometheus metrics on the client address:
```bash
curl http://localhost:8081/metrics
```

### Logging
Configure structured logging:
```bash
./rangedb server --log-level debug
```

## Backup and Recovery

### Automated Backups
```bash
#!/bin/bash
# backup.sh
BACKUP_DIR="/backups/rangedb"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/rangedb_backup_$DATE.db"

mkdir -p $BACKUP_DIR
./rangedb admin backup create $BACKUP_FILE

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "rangedb_backup_*.db" -mtime +7 -delete
```

### Recovery
```bash
# Stop RangeDB
sudo systemctl stop rangedb

# Restore from backup
./rangedb admin backup restore /backups/rangedb_backup_20250101_120000.db

# Start RangeDB
sudo systemctl start rangedb
```

## Security

### Network Security
- Use TLS for client connections
- Implement firewall rules
- Use VPN for cross-datacenter communication

### Authentication
```bash
# Generate TLS certificates
openssl req -x509 -newkey rsa:4096 -keyout rangedb.key -out rangedb.crt -days 365 -nodes
```

### Access Control
- Implement network-level access control
- Use API keys for application access
- Regular security audits

## Performance Tuning

### Storage Optimization
```bash
# Adjust BadgerDB settings
./rangedb server \
  --data-dir=/fast-ssd/rangedb \
  --max-batch-size=10000 \
  --flush-interval=50ms
```

### Network Optimization
- Use dedicated network for cluster communication
- Optimize TCP settings
- Consider network compression

### Resource Allocation
```yaml
# Kubernetes resource limits
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

## Troubleshooting

### Common Issues

1. **Split Brain Prevention**
   - Always use odd number of nodes
   - Monitor cluster membership
   - Implement proper network partitioning

2. **Performance Issues**
   - Monitor disk I/O
   - Check network latency
   - Optimize key design

3. **Memory Usage**
   - Monitor BadgerDB memory usage
   - Adjust cache sizes
   - Consider data retention policies

### Debugging Commands
```bash
# Check cluster health
./rangedb admin cluster status

# View metadata
./rangedb admin metadata list

# Performance benchmark
./rangedb admin performance benchmark --operations 10000

# Check logs
tail -f /var/log/rangedb/rangedb.log
```

## Maintenance

### Regular Tasks
1. **Backup Verification**
   - Test backup integrity
   - Practice restore procedures
   - Document recovery procedures

2. **Monitoring**
   - Set up alerts for cluster health
   - Monitor disk usage
   - Track performance metrics

3. **Updates**
   - Plan rolling updates
   - Test in staging environment
   - Maintain backward compatibility

### Scaling Operations

#### Adding Nodes
```bash
# New node joins existing cluster
./rangedb server \
  --peer-address=node4.example.com:8080 \
  --join=node1.example.com:8080,node2.example.com:8080,node3.example.com:8080
```

#### Removing Nodes
```bash
# Graceful node removal
./rangedb admin node decommission node3.example.com:8080
```

## Best Practices

1. **High Availability**
   - Use at least 3 nodes
   - Distribute across availability zones
   - Implement proper load balancing

2. **Data Management**
   - Regular backups
   - Monitor data growth
   - Implement data retention policies

3. **Security**
   - Use TLS everywhere
   - Regular security updates
   - Network segmentation

4. **Performance**
   - Use SSD storage
   - Optimize network configuration
   - Monitor and tune regularly

5. **Operations**
   - Automate deployments
   - Document procedures
   - Regular disaster recovery drills