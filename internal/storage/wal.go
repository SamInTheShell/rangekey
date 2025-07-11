package storage

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WALEntryType represents the type of WAL entry
type WALEntryType int

const (
	WALEntryTypePut WALEntryType = iota
	WALEntryTypeDelete
)

// WALEntry represents a single WAL entry
type WALEntry struct {
	Type      WALEntryType `json:"type"`
	Key       []byte       `json:"key"`
	Value     []byte       `json:"value,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// WAL represents the Write-Ahead Log
type WAL struct {
	dataDir string
	file    *os.File
	mu      sync.RWMutex

	// Current file info
	currentFile string
	currentSize int64

	// Configuration
	maxFileSize int64
	syncWrites  bool

	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// NewWAL creates a new WAL instance
func NewWAL(dataDir string) (*WAL, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	wal := &WAL{
		dataDir:     dataDir,
		maxFileSize: 64 * 1024 * 1024, // 64MB
		syncWrites:  true,
		stopCh:      make(chan struct{}),
	}

	return wal, nil
}

// Start starts the WAL
func (w *WAL) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return fmt.Errorf("WAL is already started")
	}

	// Find the latest WAL file or create a new one
	if err := w.openCurrentFile(); err != nil {
		return fmt.Errorf("failed to open WAL file: %w", err)
	}

	w.started = true
	log.Printf("WAL started with file: %s", w.currentFile)

	return nil
}

// Stop stops the WAL
func (w *WAL) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	close(w.stopCh)

	// Flush and close current file
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			log.Printf("Error syncing WAL file: %v", err)
		}
		if err := w.file.Close(); err != nil {
			log.Printf("Error closing WAL file: %v", err)
		}
		w.file = nil
	}

	w.started = false
	log.Println("WAL stopped")

	return nil
}

// Write writes a single entry to the WAL
func (w *WAL) Write(entry *WALEntry) error {
	return w.WriteBatch([]*WALEntry{entry})
}

// WriteBatch writes multiple entries to the WAL atomically
func (w *WAL) WriteBatch(entries []*WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return fmt.Errorf("WAL is not started")
	}

	// Serialize entries
	data, err := w.serializeEntries(entries)
	if err != nil {
		return fmt.Errorf("failed to serialize entries: %w", err)
	}

	// Check if we need to rotate the file
	if w.currentSize+int64(len(data)) > w.maxFileSize {
		if err := w.rotateFile(); err != nil {
			return fmt.Errorf("failed to rotate WAL file: %w", err)
		}
	}

	// Write to file
	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	w.currentSize += int64(n)

	// Sync if required
	if w.syncWrites {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("failed to sync WAL: %w", err)
		}
	}

	return nil
}

// ReadAll reads all entries from the WAL
func (w *WAL) ReadAll() ([]*WALEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Find all WAL files
	files, err := w.findWALFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find WAL files: %w", err)
	}

	var allEntries []*WALEntry

	// Read from all files in order
	for _, filename := range files {
		entries, err := w.readFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read WAL file %s: %w", filename, err)
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// Clear removes all WAL entries
func (w *WAL) Clear() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Close current file
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("failed to close WAL file: %w", err)
		}
		w.file = nil
	}

	// Remove all WAL files
	files, err := w.findWALFiles()
	if err != nil {
		return fmt.Errorf("failed to find WAL files: %w", err)
	}

	for _, filename := range files {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("failed to remove WAL file %s: %w", filename, err)
		}
	}

	// Create new file
	if err := w.openCurrentFile(); err != nil {
		return fmt.Errorf("failed to create new WAL file: %w", err)
	}

	return nil
}

// Flush flushes the WAL to disk
func (w *WAL) Flush() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.file == nil {
		return nil
	}

	return w.file.Sync()
}

// openCurrentFile opens the current WAL file
func (w *WAL) openCurrentFile() error {
	// Find the latest file or create a new one
	files, err := w.findWALFiles()
	if err != nil {
		return fmt.Errorf("failed to find WAL files: %w", err)
	}

	var filename string
	if len(files) == 0 {
		// Create first file
		filename = filepath.Join(w.dataDir, fmt.Sprintf("wal-%d.log", time.Now().UnixNano()))
	} else {
		// Use the latest file
		filename = files[len(files)-1]
	}

	// Open file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open WAL file: %w", err)
	}

	// Get current size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat WAL file: %w", err)
	}

	w.file = file
	w.currentFile = filename
	w.currentSize = info.Size()

	return nil
}

// rotateFile rotates the current WAL file
func (w *WAL) rotateFile() error {
	// Close current file
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("failed to sync WAL file: %w", err)
		}
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("failed to close WAL file: %w", err)
		}
	}

	// Create new file
	filename := filepath.Join(w.dataDir, fmt.Sprintf("wal-%d.log", time.Now().UnixNano()))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new WAL file: %w", err)
	}

	w.file = file
	w.currentFile = filename
	w.currentSize = 0

	log.Printf("WAL file rotated to: %s", filename)
	return nil
}

// findWALFiles finds all WAL files in the data directory
func (w *WAL) findWALFiles() ([]string, error) {
	entries, err := os.ReadDir(w.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".log" {
			files = append(files, filepath.Join(w.dataDir, entry.Name()))
		}
	}

	return files, nil
}

// readFile reads all entries from a single WAL file
func (w *WAL) readFile(filename string) ([]*WALEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	var entries []*WALEntry

	for {
		entry, err := w.readEntry(file)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read WAL entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// readEntry reads a single entry from a file
func (w *WAL) readEntry(file *os.File) (*WALEntry, error) {
	// Read length prefix
	var length uint32
	if err := binary.Read(file, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	// Read entry data
	data := make([]byte, length)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, err
	}

	// Deserialize entry
	var entry WALEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WAL entry: %w", err)
	}

	return &entry, nil
}

// serializeEntries serializes multiple entries into a byte slice
func (w *WAL) serializeEntries(entries []*WALEntry) ([]byte, error) {
	var result []byte

	for _, entry := range entries {
		// Serialize entry
		data, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal WAL entry: %w", err)
		}

		// Write length prefix
		length := uint32(len(data))
		lengthBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(lengthBytes, length)

		result = append(result, lengthBytes...)
		result = append(result, data...)
	}

	return result, nil
}
