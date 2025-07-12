package client

import (
	"context"
	"fmt"
	"time"

	"github.com/samintheshell/rangekey/api/rangedb/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds the client configuration
type Config struct {
	Address        string
	RequestTimeout time.Duration
	MaxRetries     int
}

// Client represents the RangeDB client
type Client struct {
	config *Config
	conn   *grpc.ClientConn
	client v1.RangeDBClient
}

// NewClient creates a new RangeDB client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("client config is required")
	}

	if config.Address == "" {
		return nil, fmt.Errorf("server address is required")
	}

	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &Client{
		config: config,
	}, nil
}

// Connect establishes a connection to the RangeDB server
func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		return fmt.Errorf("client is already connected")
	}

	// Connect to the server
	conn, err := grpc.NewClient(c.config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.conn = conn
	c.client = v1.NewRangeDBClient(conn)

	return nil
}

// Close closes the connection to the RangeDB server
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, &v1.GetRequest{
		Key: []byte(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if !resp.Found {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return resp.Kv.Value, nil
}

// Put stores a key-value pair
func (c *Client) Put(ctx context.Context, key string, value []byte) error {
	if c.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	_, err := c.client.Put(ctx, &v1.PutRequest{
		Key:   []byte(key),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("failed to put key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key
func (c *Client) Delete(ctx context.Context, key string) error {
	if c.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	_, err := c.client.Delete(ctx, &v1.DeleteRequest{
		Key: []byte(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// Range retrieves multiple key-value pairs
func (c *Client) Range(ctx context.Context, startKey, endKey string, limit int) ([]*v1.KeyValue, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	limit32 := int32(limit)
	stream, err := c.client.Range(ctx, &v1.RangeRequest{
		StartKey: []byte(startKey),
		EndKey:   []byte(endKey),
		Limit:    &limit32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start range query: %w", err)
	}

	var results []*v1.KeyValue
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Check if we've reached the end
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to receive range response: %w", err)
		}

		if resp.Kv != nil {
			results = append(results, resp.Kv)
		}

		// Check if there are more results
		if !resp.HasMore {
			break
		}
	}

	return results, nil
}

// Batch executes multiple operations in a single request
func (c *Client) Batch(ctx context.Context, operations []*v1.BatchOperation) error {
	if c.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	_, err := c.client.Batch(ctx, &v1.BatchRequest{
		Operations: operations,
	})
	if err != nil {
		return fmt.Errorf("failed to execute batch operations: %w", err)
	}

	return nil
}

// Transaction represents a database transaction
type Transaction struct {
	client *Client
	id     string
}

// ID returns the transaction ID
func (t *Transaction) ID() string {
	return t.id
}

// NewTransactionFromID creates a transaction with an existing ID
func (c *Client) NewTransactionFromID(id string) *Transaction {
	return &Transaction{
		client: c,
		id:     id,
	}
}

// BeginTransaction starts a new transaction
func (c *Client) BeginTransaction(ctx context.Context) (*Transaction, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.BeginTransaction(ctx, &v1.BeginTransactionRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{
		client: c,
		id:     resp.TransactionId,
	}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit(ctx context.Context) error {
	if t.client.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	_, err := t.client.client.CommitTransaction(ctx, &v1.CommitTransactionRequest{
		TransactionId: t.id,
	})
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback(ctx context.Context) error {
	if t.client.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	_, err := t.client.client.RollbackTransaction(ctx, &v1.RollbackTransactionRequest{
		TransactionId: t.id,
	})
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// Get retrieves a value by key within the transaction
func (t *Transaction) Get(ctx context.Context, key string) ([]byte, error) {
	if t.client.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	resp, err := t.client.client.Get(ctx, &v1.GetRequest{
		Key:           []byte(key),
		TransactionId: &t.id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if !resp.Found {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return resp.Kv.Value, nil
}

// Put stores a key-value pair within the transaction
func (t *Transaction) Put(ctx context.Context, key string, value []byte) error {
	if t.client.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	_, err := t.client.client.Put(ctx, &v1.PutRequest{
		Key:           []byte(key),
		Value:         value,
		TransactionId: &t.id,
	})
	if err != nil {
		return fmt.Errorf("failed to put key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key within the transaction
func (t *Transaction) Delete(ctx context.Context, key string) error {
	if t.client.client == nil {
		return fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	_, err := t.client.client.Delete(ctx, &v1.DeleteRequest{
		Key:           []byte(key),
		TransactionId: &t.id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// Range retrieves multiple key-value pairs within the transaction
func (t *Transaction) Range(ctx context.Context, startKey, endKey string, limit int) ([]*v1.KeyValue, error) {
	if t.client.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.client.config.RequestTimeout)
	defer cancel()

	limit32 := int32(limit)
	stream, err := t.client.client.Range(ctx, &v1.RangeRequest{
		StartKey:      []byte(startKey),
		EndKey:        []byte(endKey),
		Limit:         &limit32,
		TransactionId: &t.id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start range query: %w", err)
	}

	var results []*v1.KeyValue
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Check if we've reached the end
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to receive range response: %w", err)
		}

		if resp.Kv != nil {
			results = append(results, resp.Kv)
		}

		// Check if there are more results
		if !resp.HasMore {
			break
		}
	}

	return results, nil
}

// GetClusterInfo retrieves cluster information
func (c *Client) GetClusterInfo(ctx context.Context) (*v1.GetClusterInfoResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.GetClusterInfo(ctx, &v1.GetClusterInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	return resp, nil
}

// GetNodeInfo retrieves node information
func (c *Client) GetNodeInfo(ctx context.Context) (*v1.GetNodeInfoResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	resp, err := c.client.GetNodeInfo(ctx, &v1.GetNodeInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	return resp, nil
}
