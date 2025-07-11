package config

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// ServerConfig holds the server configuration
type ServerConfig struct {
	// Network configuration
	PeerAddress   string   `json:"peer_address"`
	ClientAddress string   `json:"client_address"`
	Peers         []string `json:"peers"`
	JoinAddresses []string `json:"join_addresses"`
	RaftPort      int      `json:"raft_port"`

	// Cluster configuration
	ClusterInit bool   `json:"cluster_init"`
	NodeID      string `json:"node_id"`

	// Storage configuration
	DataDir string `json:"data_dir"`

	// Logging configuration
	LogLevel string `json:"log_level"`

	// Timeouts and limits
	RequestTimeout    time.Duration `json:"request_timeout"`
	HeartbeatTimeout  time.Duration `json:"heartbeat_timeout"`
	ElectionTimeout   time.Duration `json:"election_timeout"`
	ReplicationFactor int           `json:"replication_factor"`

	// Performance tuning
	MaxBatchSize    int           `json:"max_batch_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	CompactionLevel int           `json:"compaction_level"`
}

// DefaultServerConfig returns a server configuration with sensible defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		PeerAddress:       "localhost:8080",
		ClientAddress:     "localhost:8081",
		RaftPort:          8082,
		DataDir:           "./data",
		LogLevel:          "info",
		RequestTimeout:    5 * time.Second,
		HeartbeatTimeout:  1 * time.Second,
		ElectionTimeout:   3 * time.Second,
		ReplicationFactor: 3,
		MaxBatchSize:      1000,
		FlushInterval:     100 * time.Millisecond,
		CompactionLevel:   1,
	}
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.PeerAddress == "" {
		return errors.New("peer address is required")
	}

	if c.ClientAddress == "" {
		return errors.New("client address is required")
	}

	if c.DataDir == "" {
		return errors.New("data directory is required")
	}

	// Validate addresses
	if err := c.validateAddress(c.PeerAddress); err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	if err := c.validateAddress(c.ClientAddress); err != nil {
		return fmt.Errorf("invalid client address: %w", err)
	}

	// Validate peer addresses
	for _, peer := range c.Peers {
		if err := c.validateAddress(peer); err != nil {
			return fmt.Errorf("invalid peer address %s: %w", peer, err)
		}
	}

	// Validate join addresses
	for _, addr := range c.JoinAddresses {
		if err := c.validateAddress(addr); err != nil {
			return fmt.Errorf("invalid join address %s: %w", addr, err)
		}
	}

	// Validate port ranges
	if c.RaftPort < 1 || c.RaftPort > 65535 {
		return errors.New("raft port must be between 1 and 65535")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, c.LogLevel) {
		return fmt.Errorf("invalid log level: %s (must be one of: %s)", c.LogLevel, strings.Join(validLogLevels, ", "))
	}

	// Validate replication factor
	if c.ReplicationFactor < 1 {
		return errors.New("replication factor must be at least 1")
	}

	// Validate timeouts
	if c.RequestTimeout <= 0 {
		return errors.New("request timeout must be positive")
	}

	if c.HeartbeatTimeout <= 0 {
		return errors.New("heartbeat timeout must be positive")
	}

	if c.ElectionTimeout <= 0 {
		return errors.New("election timeout must be positive")
	}

	if c.ElectionTimeout <= c.HeartbeatTimeout {
		return errors.New("election timeout must be greater than heartbeat timeout")
	}

	// Validate batch size
	if c.MaxBatchSize <= 0 {
		return errors.New("max batch size must be positive")
	}

	return nil
}

// validateAddress validates a network address
func (c *ServerConfig) validateAddress(addr string) error {
	if addr == "" {
		return errors.New("address cannot be empty")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	if host == "" {
		return errors.New("host cannot be empty")
	}

	if port == "" {
		return errors.New("port cannot be empty")
	}

	return nil
}

// GetRaftAddress returns the Raft address for this node
func (c *ServerConfig) GetRaftAddress() string {
	host, _, err := net.SplitHostPort(c.PeerAddress)
	if err != nil {
		return c.PeerAddress
	}
	return fmt.Sprintf("%s:%d", host, c.RaftPort)
}

// GetDataPath returns the full path for a data file
func (c *ServerConfig) GetDataPath(filename string) string {
	return filepath.Join(c.DataDir, filename)
}

// GetNodeID returns the node ID for this server
func (c *ServerConfig) GetNodeID() string {
	if c.NodeID != "" {
		return c.NodeID
	}
	// Use peer address as node ID if not specified
	return c.PeerAddress
}

// IsBootstrap returns true if this node should initialize the cluster
func (c *ServerConfig) IsBootstrap() bool {
	return c.ClusterInit
}

// IsJoinMode returns true if this node should join an existing cluster
func (c *ServerConfig) IsJoinMode() bool {
	return len(c.JoinAddresses) > 0
}

// GetPeerList returns the list of peer addresses
func (c *ServerConfig) GetPeerList() []string {
	if len(c.Peers) > 0 {
		return c.Peers
	}
	return []string{c.PeerAddress}
}

// GetJoinList returns the list of join addresses
func (c *ServerConfig) GetJoinList() []string {
	return c.JoinAddresses
}

// ClientConfig holds the client configuration
type ClientConfig struct {
	Endpoints    []string      `json:"endpoints"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	RequestTimeout time.Duration `json:"request_timeout"`
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
}

// DefaultClientConfig returns a client configuration with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Endpoints:      []string{"localhost:8081"},
		DialTimeout:    5 * time.Second,
		RequestTimeout: 10 * time.Second,
		MaxRetries:     3,
		RetryDelay:     100 * time.Millisecond,
	}
}

// Validate validates the client configuration
func (c *ClientConfig) Validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("at least one endpoint is required")
	}

	for _, endpoint := range c.Endpoints {
		if err := validateAddress(endpoint); err != nil {
			return fmt.Errorf("invalid endpoint %s: %w", endpoint, err)
		}
	}

	if c.DialTimeout <= 0 {
		return errors.New("dial timeout must be positive")
	}

	if c.RequestTimeout <= 0 {
		return errors.New("request timeout must be positive")
	}

	if c.MaxRetries < 0 {
		return errors.New("max retries must be non-negative")
	}

	if c.RetryDelay < 0 {
		return errors.New("retry delay must be non-negative")
	}

	return nil
}

// validateAddress validates a network address
func validateAddress(addr string) error {
	if addr == "" {
		return errors.New("address cannot be empty")
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	if host == "" {
		return errors.New("host cannot be empty")
	}

	if port == "" {
		return errors.New("port cannot be empty")
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
