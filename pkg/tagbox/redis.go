package tagbox

import (
	"bytes"
	"fmt"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/redis/go-redis/v9"
)

// saveWorker runs in the background to save tags to Redis when triggered.
func (ts *TagSystem) saveWorker() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Batch save timer: save after no activity for 1 second
	var lastTrigger time.Time

	for range ticker.C {
		select {
		case <-ts.config.SaveChan:
			lastTrigger = time.Now()
		default:
			// Save if 1 second has passed since last trigger
			if !lastTrigger.IsZero() && time.Since(lastTrigger) >= time.Second {
				ts.SaveToRedis()
				lastTrigger = time.Time{}
			}
		}
	}
}

// SaveToRedis saves all tags to Redis.
func (ts *TagSystem) SaveToRedis() error {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var errs []error

	for tag, bitmap := range ts.tags {
		if err := ts.saveTagToRedis(tag, bitmap); err != nil {
			errs = append(errs, fmt.Errorf("tag %s: %w", tag, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("save completed with %d errors: %v", len(errs), errs)
	}

	return nil
}

// saveTagToRedis saves a single tag to Redis.
func (ts *TagSystem) saveTagToRedis(tag string, bitmap *roaring.Bitmap) error {
	// Serialize bitmap to bytes
	var buf bytes.Buffer
	_, err := bitmap.WriteTo(&buf)
	if err != nil {
		return err
	}

	// Save to Redis
	key := ts.config.KeyPrefix + tag
	return ts.redis.Set(ts.ctx, key, buf.Bytes(), 0).Err()
}

// SaveTagToRedis saves a specific tag to Redis immediately.
func (ts *TagSystem) SaveTagToRedis(tag string) error {
	ts.mu.RLock()
	bitmap, exists := ts.tags[tag]
	ts.mu.RUnlock()

	if !exists {
		return fmt.Errorf("tag not found: %s", tag)
	}

	return ts.saveTagToRedis(tag, bitmap)
}

// LoadTagFromRedis loads a specific tag from Redis.
func (ts *TagSystem) LoadTagFromRedis(tag string) error {
	key := ts.config.KeyPrefix + tag

	data, err := ts.redis.Get(ts.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("tag not found: %s", tag)
		}
		return err
	}

	bitmap := roaring.NewBitmap()
	if _, err := bitmap.ReadFrom(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("deserialization failed: %w", err)
	}

	ts.mu.Lock()
	ts.tags[tag] = bitmap
	ts.allObjects.Or(bitmap)
	ts.mu.Unlock()

	return nil
}
