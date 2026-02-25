package tagbox

import "time"

// Config represents the configuration for the tag system.
type Config struct {
	// Redis connection
	RedisAddr     string // Redis server address, e.g., "localhost:6379"
	RedisPassword string // Redis password (empty if no password)
	RedisDB       int    // Redis database number

	// Tag storage
	KeyPrefix string // Redis key prefix for tags, e.g., "tags:"

	// Persistence
	AutoSave bool          // AutoSave automatically saves tags to Redis after modifications
	SaveChan chan struct{} // Internal channel for triggering saves

	// Performance tuning
	EnableSnapshot    bool          // EnableSnapshot enables periodic snapshot to disk
	SnapshotPath      string        // SnapshotPath is the file path for snapshots
	SnapshotInterval  time.Duration // SnapshotInterval is the interval between snapshots

	// Query optimization
	CacheResults bool // CacheResults enables query result caching
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		KeyPrefix:        "tags:",
		AutoSave:         true,
		SaveChan:         make(chan struct{}, 100),
		EnableSnapshot:   false,
		SnapshotPath:     "",
		SnapshotInterval: 5 * time.Minute,
		CacheResults:     false,
	}
}

// Stats represents statistics about the tag system.
type Stats struct {
	TotalTags      int     // Total number of tags
	TotalObjects   uint64  // Total number of tagged objects (with duplicates)
	UniqueObjects  uint64  // Total number of unique objects across all tags
	MemoryUsage    uint64  // Total memory usage in bytes
	LargestTag     string  // The tag with the most objects
	LargestTagSize uint64  // Number of objects in the largest tag
}
