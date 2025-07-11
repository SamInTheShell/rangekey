package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/samintheshell/rangekey/internal/storage"
)

// StateMachine handles applying committed entries to the storage layer
type StateMachine struct {
	storage *storage.Engine
	commitC chan *commit
	errorC  chan error
	stopCh  chan struct{}
}

// Operation represents a state machine operation
type Operation struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Value []byte `json:"value,omitempty"`
}

// Operation types
const (
	OpTypePut    = "PUT"
	OpTypeDelete = "DELETE"
)

// NewStateMachine creates a new state machine
func NewStateMachine(storage *storage.Engine, commitC chan *commit, errorC chan error) *StateMachine {
	return &StateMachine{
		storage: storage,
		commitC: commitC,
		errorC:  errorC,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the state machine
func (sm *StateMachine) Start(ctx context.Context) error {
	go sm.run()
	return nil
}

// Stop stops the state machine
func (sm *StateMachine) Stop(ctx context.Context) error {
	close(sm.stopCh)
	return nil
}

// run is the main state machine loop
func (sm *StateMachine) run() {
	for {
		select {
		case commit := <-sm.commitC:
			if err := sm.applyCommit(commit); err != nil {
				log.Printf("Failed to apply commit: %v", err)
				select {
				case sm.errorC <- err:
				case <-sm.stopCh:
					return
				}
			}
			close(commit.applyDoneC)

		case <-sm.stopCh:
			return
		}
	}
}

// applyCommit applies a committed entry to the storage
func (sm *StateMachine) applyCommit(commit *commit) error {
	// Check if the data looks like JSON (starts with '{')
	if len(commit.data) == 0 || commit.data[0] != '{' {
		// This is likely an internal Raft message, skip it
		log.Printf("Skipping non-JSON commit entry of length %d", len(commit.data))
		return nil
	}

	// Parse the operation
	var op Operation
	if err := json.Unmarshal(commit.data, &op); err != nil {
		// If it's not our operation format, skip it
		log.Printf("Skipping non-operation commit entry: %v", err)
		return nil
	}

	// Apply the operation
	ctx := context.Background()
	switch op.Type {
	case OpTypePut:
		if err := sm.storage.Put(ctx, []byte(op.Key), op.Value); err != nil {
			return fmt.Errorf("failed to put key %s: %w", op.Key, err)
		}
		log.Printf("Applied PUT: %s", op.Key)

	case OpTypeDelete:
		if err := sm.storage.Delete(ctx, []byte(op.Key)); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", op.Key, err)
		}
		log.Printf("Applied DELETE: %s", op.Key)

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return nil
}

// CreatePutOperation creates a PUT operation
func CreatePutOperation(key string, value []byte) ([]byte, error) {
	op := Operation{
		Type:  OpTypePut,
		Key:   key,
		Value: value,
	}
	return json.Marshal(op)
}

// CreateDeleteOperation creates a DELETE operation
func CreateDeleteOperation(key string) ([]byte, error) {
	op := Operation{
		Type: OpTypeDelete,
		Key:  key,
	}
	return json.Marshal(op)
}
