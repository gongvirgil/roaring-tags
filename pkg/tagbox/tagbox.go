package tagbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/redis/go-redis/v9"
)

// TagSystem represents a high-performance object tagging system.
// It uses RoaringBitmap for efficient bitmap operations and Redis for persistence.
type TagSystem struct {
	mu    sync.RWMutex
	tags  map[string]*roaring.Bitmap
	redis *redis.Client
	ctx   context.Context
	config Config

	// For tracking unique objects across all tags
	allObjects *roaring.Bitmap

	// Snapshot management
	snapshotTicker *time.Ticker
	snapshotDone   chan struct{}
}

// New creates a new TagSystem with the given configuration.
func New(config Config) (*TagSystem, error) {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	ts := &TagSystem{
		tags:       make(map[string]*roaring.Bitmap),
		redis:      rdb,
		ctx:        ctx,
		config:     config,
		allObjects: roaring.NewBitmap(),
	}

	// Start background save worker if AutoSave is enabled
	if config.AutoSave {
		go ts.saveWorker()
	}

	return ts, nil
}

// RecoverFromRedis recovers tag data from Redis.
// This should be called after creating a new TagSystem to restore existing data.
func (ts *TagSystem) RecoverFromRedis() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Scan all tag keys
	iter := ts.redis.Scan(ts.ctx, 0, ts.config.KeyPrefix+"*", 0).Iterator()
	keys := make([]string, 0)

	for iter.Next(ts.ctx) {
		key := iter.Val()
		if key == ts.config.KeyPrefix+"_meta" {
			continue // Skip metadata key
		}
		keys = append(keys, key)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan failed: %w", err)
	}

	// Load each tag
	var errs []error
	for _, key := range keys {
		tag := key[len(ts.config.KeyPrefix):] // Remove prefix

		data, err := ts.redis.Get(ts.ctx, key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue // Key doesn't exist, skip
			}
			errs = append(errs, fmt.Errorf("tag %s: %w", tag, err))
			continue
		}

		bitmap := roaring.NewBitmap()
		if _, err := bitmap.ReadFrom(bytes.NewReader(data)); err != nil {
			errs = append(errs, fmt.Errorf("tag %s: %w", tag, err))
			continue
		}

		ts.tags[tag] = bitmap
		ts.allObjects.Or(bitmap)
	}

	if len(errs) > 0 {
		return fmt.Errorf("recover completed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// AddTag adds a tag to an object.
// If the tag doesn't exist, it will be created.
// If AutoSave is enabled, the tag will be asynchronously saved to Redis.
func (ts *TagSystem) AddTag(objectID uint32, tag string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		bitmap = roaring.NewBitmap()
		ts.tags[tag] = bitmap
	}

	bitmap.Add(objectID)
	ts.allObjects.Add(objectID)

	// Trigger async save
	if ts.config.AutoSave {
		select {
		case ts.config.SaveChan <- struct{}{}:
		default:
			// Channel is full, skip save trigger
		}
	}

	return nil
}

// RemoveTag removes a tag from an object.
func (ts *TagSystem) RemoveTag(objectID uint32, tag string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		return nil // Tag doesn't exist, nothing to remove
	}

	bitmap.Remove(objectID)

	// If bitmap is empty, remove the tag
	if bitmap.GetCardinality() == 0 {
		delete(ts.tags, tag)
		go ts.redis.Del(ts.ctx, ts.config.KeyPrefix+tag)
	} else if ts.config.AutoSave {
		select {
		case ts.config.SaveChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// BatchAddTags adds multiple tags to an object in a single operation.
func (ts *TagSystem) BatchAddTags(objectID uint32, tags []string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, tag := range tags {
		bitmap, exists := ts.tags[tag]
		if !exists {
			bitmap = roaring.NewBitmap()
			ts.tags[tag] = bitmap
		}
		bitmap.Add(objectID)
	}

	ts.allObjects.Add(objectID)

	if ts.config.AutoSave {
		select {
		case ts.config.SaveChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// BatchAddObjectsToTag adds multiple objects to a single tag.
func (ts *TagSystem) BatchAddObjectsToTag(objectIDs []uint32, tag string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		bitmap = roaring.NewBitmap()
		ts.tags[tag] = bitmap
	}

	for _, objectID := range objectIDs {
		bitmap.Add(objectID)
		ts.allObjects.Add(objectID)
	}

	if ts.config.AutoSave {
		select {
		case ts.config.SaveChan <- struct{}{}:
		default:
		}
	}

	return nil
}

// HasTag checks if an object has a specific tag.
func (ts *TagSystem) HasTag(objectID uint32, tag string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		return false
	}

	return bitmap.Contains(objectID)
}

// GetObjectTags returns all tags for a specific object.
func (ts *TagSystem) GetObjectTags(objectID uint32) ([]string, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var tags []string
	for tag, bitmap := range ts.tags {
		if bitmap.Contains(objectID) {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

// GetAllTags returns all tag names in the system.
func (ts *TagSystem) GetAllTags() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	tags := make([]string, 0, len(ts.tags))
	for tag := range ts.tags {
		tags = append(tags, tag)
	}

	return tags
}

// GetTagCount returns the number of objects with a specific tag.
func (ts *TagSystem) GetTagCount(tag string) (uint64, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		return 0, nil
	}

	return bitmap.GetCardinality(), nil
}

// GetStats returns statistics about the tag system.
func (ts *TagSystem) GetStats() Stats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	stats := Stats{
		TotalTags:     len(ts.tags),
		UniqueObjects: ts.allObjects.GetCardinality(),
	}

	var maxCardinality uint64

	for tag, bitmap := range ts.tags {
		cardinality := bitmap.GetCardinality()
		stats.TotalObjects += cardinality
		stats.MemoryUsage += bitmap.GetSizeInBytes()

		if cardinality > maxCardinality {
			maxCardinality = cardinality
			stats.LargestTag = tag
			stats.LargestTagSize = cardinality
		}
	}

	return stats
}

// Close closes the tag system and saves all data to Redis.
func (ts *TagSystem) Close() error {
	// Stop snapshot ticker if running
	if ts.snapshotTicker != nil {
		ts.snapshotTicker.Stop()
		close(ts.snapshotDone)
	}

	// Save all data to Redis
	if err := ts.SaveToRedis(); err != nil {
		return fmt.Errorf("save to redis failed: %w", err)
	}

	// Close Redis connection
	return ts.redis.Close()
}

// StartSnapshot enables periodic snapshot to disk.
func (ts *TagSystem) StartSnapshot() {
	if !ts.config.EnableSnapshot || ts.config.SnapshotPath == "" {
		return
	}

	ts.mu.Lock()
	if ts.snapshotTicker != nil {
		ts.mu.Unlock()
		return // Already started
	}
	ts.snapshotTicker = time.NewTicker(ts.config.SnapshotInterval)
	ts.snapshotDone = make(chan struct{})
	ts.mu.Unlock()

	go func() {
		for {
			select {
			case <-ts.snapshotTicker.C:
				if err := ts.SaveSnapshot(ts.config.SnapshotPath); err != nil {
					fmt.Printf("snapshot failed: %v\n", err)
				}
			case <-ts.snapshotDone:
				return
			}
		}
	}()
}

// SaveSnapshot saves all tags to a snapshot file.
func (ts *TagSystem) SaveSnapshot(filePath string) error {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	data := make(map[string][]byte)

	for tag, bitmap := range ts.tags {
		var buf bytes.Buffer
		if _, err := bitmap.WriteTo(&buf); err != nil {
			return fmt.Errorf("failed to serialize tag %s: %w", tag, err)
		}
		data[tag] = buf.Bytes()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadSnapshot loads all tags from a snapshot file.
func (ts *TagSystem) LoadSnapshot(filePath string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	data := make(map[string][]byte)
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	for tag, buf := range data {
		bitmap := roaring.NewBitmap()
		if _, err := bitmap.ReadFrom(bytes.NewReader(buf)); err != nil {
			return fmt.Errorf("load tag %s failed: %w", tag, err)
		}
		ts.tags[tag] = bitmap
		ts.allObjects.Or(bitmap)
	}

	return nil
}
