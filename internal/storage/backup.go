package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"encoding/json"
	"archive/tar"
	"compress/gzip"
	"io"
	"strings"
	"sort"
)

// BackupManager handles backup and recovery operations
type BackupManager struct {
	engine *Engine
	config *BackupConfig
	
	// Active backup operations
	mu              sync.RWMutex
	activeBackups   map[string]*BackupOperation
	activeRestores  map[string]*RestoreOperation
	
	// Lifecycle
	started bool
	stopCh  chan struct{}
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	BackupDir           string
	MaxBackupsToRetain  int
	CompressionEnabled  bool
	BackupSchedule      string // Cron expression for scheduled backups
	IncrementalEnabled  bool
}

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"` // "full" or "incremental"
	Path              string    `json:"path"`
	Size              int64     `json:"size"`
	CreatedAt         time.Time `json:"created_at"`
	CompletedAt       time.Time `json:"completed_at"`
	Status            string    `json:"status"` // "in_progress", "completed", "failed"
	BaseBackupID      string    `json:"base_backup_id,omitempty"` // For incremental backups
	KeyCount          int64     `json:"key_count"`
	CompressionRatio  float64   `json:"compression_ratio"`
	Checksum          string    `json:"checksum"`
	Version           string    `json:"version"`
	NodeID            string    `json:"node_id"`
	ClusterID         string    `json:"cluster_id"`
}

// BackupOperation represents an active backup operation
type BackupOperation struct {
	ID        string
	Type      string
	Path      string
	StartTime time.Time
	Status    string
	Progress  float64
	Error     error
}

// RestoreOperation represents an active restore operation
type RestoreOperation struct {
	ID        string
	Path      string
	StartTime time.Time
	Status    string
	Progress  float64
	Error     error
}

// NewBackupManager creates a new backup manager
func NewBackupManager(engine *Engine, config *BackupConfig) *BackupManager {
	if config == nil {
		config = &BackupConfig{
			BackupDir:           "./backups",
			MaxBackupsToRetain:  10,
			CompressionEnabled:  true,
			IncrementalEnabled:  false,
		}
	}
	
	return &BackupManager{
		engine:          engine,
		config:          config,
		activeBackups:   make(map[string]*BackupOperation),
		activeRestores:  make(map[string]*RestoreOperation),
		stopCh:          make(chan struct{}),
	}
}

// Start starts the backup manager
func (b *BackupManager) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.started {
		return fmt.Errorf("backup manager is already started")
	}
	
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(b.config.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	log.Printf("Starting backup manager with backup directory: %s", b.config.BackupDir)
	
	// Start background cleanup goroutine
	go b.cleanupOldBackups()
	
	b.started = true
	log.Println("Backup manager started successfully")
	
	return nil
}

// Stop stops the backup manager
func (b *BackupManager) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.started {
		return nil
	}
	
	log.Println("Stopping backup manager...")
	
	// Cancel all active operations
	for id := range b.activeBackups {
		log.Printf("Cancelling active backup: %s", id)
	}
	for id := range b.activeRestores {
		log.Printf("Cancelling active restore: %s", id)
	}
	
	// Signal stop to background goroutines
	close(b.stopCh)
	
	b.started = false
	log.Println("Backup manager stopped")
	
	return nil
}

// CreateFullBackup creates a full backup of the database
func (b *BackupManager) CreateFullBackup(ctx context.Context, backupPath string) (*BackupMetadata, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.started {
		return nil, fmt.Errorf("backup manager is not started")
	}
	
	// Generate backup ID
	backupID := fmt.Sprintf("backup-%d", time.Now().Unix())
	
	// Create backup operation
	operation := &BackupOperation{
		ID:        backupID,
		Type:      "full",
		Path:      backupPath,
		StartTime: time.Now(),
		Status:    "in_progress",
		Progress:  0.0,
	}
	
	b.activeBackups[backupID] = operation
	
	log.Printf("Starting full backup %s to %s", backupID, backupPath)
	
	// Create backup metadata
	metadata := &BackupMetadata{
		ID:        backupID,
		Type:      "full",
		Path:      backupPath,
		CreatedAt: time.Now(),
		Status:    "in_progress",
		Version:   "1.0.0",
		NodeID:    "node-1", // TODO: Get actual node ID
		ClusterID: "cluster-1", // TODO: Get actual cluster ID
	}
	
	// Start backup in background
	go b.performFullBackup(ctx, operation, metadata)
	
	return metadata, nil
}

// performFullBackup performs the actual backup operation
func (b *BackupManager) performFullBackup(ctx context.Context, operation *BackupOperation, metadata *BackupMetadata) {
	defer func() {
		b.mu.Lock()
		delete(b.activeBackups, operation.ID)
		b.mu.Unlock()
	}()
	
	// Create backup directory
	backupDir := filepath.Join(b.config.BackupDir, operation.ID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		operation.Error = fmt.Errorf("failed to create backup directory: %w", err)
		operation.Status = "failed"
		return
	}
	
	// Create backup file
	backupFile := filepath.Join(backupDir, "backup.tar.gz")
	
	file, err := os.Create(backupFile)
	if err != nil {
		operation.Error = fmt.Errorf("failed to create backup file: %w", err)
		operation.Status = "failed"
		return
	}
	defer file.Close()
	
	// Create gzip writer if compression is enabled
	var writer io.Writer = file
	if b.config.CompressionEnabled {
		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()
		writer = gzipWriter
	}
	
	// Create tar writer
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()
	
	// Backup all data
	operation.Progress = 0.1
	if err := b.backupStorageData(ctx, tarWriter, operation); err != nil {
		operation.Error = fmt.Errorf("failed to backup storage data: %w", err)
		operation.Status = "failed"
		return
	}
	
	// Update progress
	operation.Progress = 0.9
	
	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		operation.Error = fmt.Errorf("failed to get backup file info: %w", err)
		operation.Status = "failed"
		return
	}
	
	// Complete metadata
	metadata.Size = fileInfo.Size()
	metadata.CompletedAt = time.Now()
	metadata.Status = "completed"
	metadata.Path = backupFile
	
	// Save metadata
	if err := b.saveBackupMetadata(metadata); err != nil {
		operation.Error = fmt.Errorf("failed to save backup metadata: %w", err)
		operation.Status = "failed"
		return
	}
	
	// Complete operation
	operation.Progress = 1.0
	operation.Status = "completed"
	
	log.Printf("Full backup %s completed successfully", operation.ID)
}

// backupStorageData backs up the storage data to a tar writer
func (b *BackupManager) backupStorageData(ctx context.Context, tarWriter *tar.Writer, operation *BackupOperation) error {
	// Get all keys
	keys, err := b.engine.GetAllKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all keys: %w", err)
	}
	
	totalKeys := len(keys)
	processed := 0
	
	// Create a data file in the tar
	dataHeader := &tar.Header{
		Name: "data.json",
		Mode: 0644,
		Size: 0, // Will be updated after writing
	}
	
	// We'll write the data as JSON lines
	var dataBuffer []byte
	for _, key := range keys {
		value, err := b.engine.Get(ctx, key)
		if err != nil {
			log.Printf("Failed to get key %s during backup: %v", string(key), err)
			continue
		}
		
		// Create data entry
		entry := map[string]interface{}{
			"key":   string(key),
			"value": string(value),
		}
		
		entryBytes, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Failed to marshal entry for key %s: %v", string(key), err)
			continue
		}
		
		dataBuffer = append(dataBuffer, entryBytes...)
		dataBuffer = append(dataBuffer, '\n')
		
		processed++
		operation.Progress = 0.1 + (0.8 * float64(processed) / float64(totalKeys))
	}
	
	// Update header size
	dataHeader.Size = int64(len(dataBuffer))
	
	// Write header
	if err := tarWriter.WriteHeader(dataHeader); err != nil {
		return fmt.Errorf("failed to write data header: %w", err)
	}
	
	// Write data
	if _, err := tarWriter.Write(dataBuffer); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	log.Printf("Backed up %d keys", processed)
	return nil
}

// saveBackupMetadata saves backup metadata to disk
func (b *BackupManager) saveBackupMetadata(metadata *BackupMetadata) error {
	metadataFile := filepath.Join(filepath.Dir(metadata.Path), "metadata.json")
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return nil
}

// RestoreFromBackup restores the database from a backup
func (b *BackupManager) RestoreFromBackup(ctx context.Context, backupPath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.started {
		return fmt.Errorf("backup manager is not started")
	}
	
	// Generate restore ID
	restoreID := fmt.Sprintf("restore-%d", time.Now().Unix())
	
	// Create restore operation
	operation := &RestoreOperation{
		ID:        restoreID,
		Path:      backupPath,
		StartTime: time.Now(),
		Status:    "in_progress",
		Progress:  0.0,
	}
	
	b.activeRestores[restoreID] = operation
	
	log.Printf("Starting restore %s from %s", restoreID, backupPath)
	
	// Start restore in background
	go b.performRestore(ctx, operation)
	
	return nil
}

// performRestore performs the actual restore operation
func (b *BackupManager) performRestore(ctx context.Context, operation *RestoreOperation) {
	defer func() {
		b.mu.Lock()
		delete(b.activeRestores, operation.ID)
		b.mu.Unlock()
	}()
	
	// Open backup file
	file, err := os.Open(operation.Path)
	if err != nil {
		operation.Error = fmt.Errorf("failed to open backup file: %w", err)
		operation.Status = "failed"
		return
	}
	defer file.Close()
	
	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		operation.Error = fmt.Errorf("failed to create gzip reader: %w", err)
		operation.Status = "failed"
		return
	}
	defer gzipReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	
	// Read and restore data
	operation.Progress = 0.1
	if err := b.restoreStorageData(ctx, tarReader, operation); err != nil {
		operation.Error = fmt.Errorf("failed to restore storage data: %w", err)
		operation.Status = "failed"
		return
	}
	
	// Complete operation
	operation.Progress = 1.0
	operation.Status = "completed"
	
	log.Printf("Restore %s completed successfully", operation.ID)
}

// restoreStorageData restores storage data from a tar reader
func (b *BackupManager) restoreStorageData(ctx context.Context, tarReader *tar.Reader, operation *RestoreOperation) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		
		if header.Name == "data.json" {
			// Read the data
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}
			
			// Parse and restore entries
			lines := string(data)
			entries := make([]map[string]interface{}, 0)
			
			// Split by newlines and parse each entry
			for _, line := range strings.Split(lines, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				
				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					log.Printf("Failed to parse entry: %v", err)
					continue
				}
				
				entries = append(entries, entry)
			}
			
			// Restore entries
			totalEntries := len(entries)
			processed := 0
			
			for _, entry := range entries {
				key := entry["key"].(string)
				value := entry["value"].(string)
				
				if err := b.engine.Put(ctx, []byte(key), []byte(value)); err != nil {
					log.Printf("Failed to restore key %s: %v", key, err)
					continue
				}
				
				processed++
				operation.Progress = 0.1 + (0.8 * float64(processed) / float64(totalEntries))
			}
			
			log.Printf("Restored %d entries", processed)
		}
	}
	
	return nil
}

// cleanupOldBackups removes old backups to maintain the retention policy
func (b *BackupManager) cleanupOldBackups() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			b.performCleanup()
		case <-b.stopCh:
			return
		}
	}
}

// performCleanup removes old backups
func (b *BackupManager) performCleanup() {
	// Get all backup directories
	entries, err := os.ReadDir(b.config.BackupDir)
	if err != nil {
		log.Printf("Failed to read backup directory: %v", err)
		return
	}
	
	// Sort by modification time (oldest first)
	type backupInfo struct {
		name    string
		modTime time.Time
	}
	
	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupInfo{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}
	
	// Sort by modification time
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})
	
	// Remove excess backups
	if len(backups) > b.config.MaxBackupsToRetain {
		toRemove := len(backups) - b.config.MaxBackupsToRetain
		for i := 0; i < toRemove; i++ {
			backupPath := filepath.Join(b.config.BackupDir, backups[i].name)
			if err := os.RemoveAll(backupPath); err != nil {
				log.Printf("Failed to remove old backup %s: %v", backupPath, err)
			} else {
				log.Printf("Removed old backup: %s", backupPath)
			}
		}
	}
}

// GetBackupMetadata returns metadata about a backup
func (b *BackupManager) GetBackupMetadata(backupPath string) (*BackupMetadata, error) {
	metadataFile := filepath.Join(filepath.Dir(backupPath), "metadata.json")
	
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	return &metadata, nil
}

// ListBackups returns a list of all available backups
func (b *BackupManager) ListBackups() ([]*BackupMetadata, error) {
	entries, err := os.ReadDir(b.config.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	var backups []*BackupMetadata
	for _, entry := range entries {
		if entry.IsDir() {
			metadataFile := filepath.Join(b.config.BackupDir, entry.Name(), "metadata.json")
			if _, err := os.Stat(metadataFile); err == nil {
				metadata, err := b.GetBackupMetadata(filepath.Join(b.config.BackupDir, entry.Name(), "backup.tar.gz"))
				if err != nil {
					log.Printf("Failed to read metadata for backup %s: %v", entry.Name(), err)
					continue
				}
				backups = append(backups, metadata)
			}
		}
	}
	
	return backups, nil
}

// GetBackupStatus returns the status of a backup operation
func (b *BackupManager) GetBackupStatus(backupID string) (*BackupOperation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	operation, exists := b.activeBackups[backupID]
	if !exists {
		return nil, fmt.Errorf("backup operation not found: %s", backupID)
	}
	
	return operation, nil
}

// GetRestoreStatus returns the status of a restore operation
func (b *BackupManager) GetRestoreStatus(restoreID string) (*RestoreOperation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	operation, exists := b.activeRestores[restoreID]
	if !exists {
		return nil, fmt.Errorf("restore operation not found: %s", restoreID)
	}
	
	return operation, nil
}