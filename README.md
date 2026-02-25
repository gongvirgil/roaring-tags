<div align="center">

# 🏷️ roaring-tags

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Redis](https://img.shields.io/badge/Redis-6.0+-DC382D?style=flat&logo=redis)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gongvirgil/roaring-tags)](https://goreportcard.com/report/github.com/gongvirgil/roaring-tags)
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)]()

**高性能对象标签系统 | High-Performance Object Tagging System**

基于 RoaringBitmap + Redis 构建的千万级对象标签解决方案

Built with RoaringBitmap + Redis for massive-scale object tagging

[Features](#-features) • [Quick Start](#-quick-start) • [Benchmarks](#-benchmarks) • [Documentation](#-documentation)

</div>

---

## ✨ Features

- ⚡️ **Millisecond Queries** - Complex tag combinations (AND/OR/NOT) in milliseconds
- 🗜️ **Memory Efficient** - 80%+ memory savings compared to traditional solutions
- 📦 **Production Ready** - Battle-tested with comprehensive test coverage
- 🔒 **Thread Safe** - Built-in concurrency protection with RWMutex
- 💾 **Persistent Storage** - Automatic Redis persistence with recovery
- 🔄 **Auto Recovery** - Service restart recovery from Redis snapshots
- 📊 **Scalable** - Tested with 10M+ objects, 100+ tags
- 🎯 **Simple API** - Clean, intuitive interface

## 🚀 Quick Start

### Installation

```bash
go get github.com/gongvirgil/roaring-tags
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/gongvirgil/roaring-tags/pkg/tagbox"
)

func main() {
    // Create a new tag system
    config := tagbox.DefaultConfig()
    config.RedisAddr = "localhost:6379"

    ts, err := tagbox.New(config)
    if err != nil {
        panic(err)
    }
    defer ts.Close()

    // Add tags to objects
    ts.AddTag(1, "vip")
    ts.AddTag(1, "male")
    ts.AddTag(2, "vip")
    ts.AddTag(2, "female")

    // Query: Find VIP AND male users
    result, _ := ts.QueryAnd([]string{"vip", "male"})
    fmt.Println(result.ToArray()) // [1]

    // Count: Get VIP user count
    count, _ := ts.GetTagCount("vip")
    fmt.Println(count) // 2
}
```

## 📖 Core Concepts

### Bitmap Indexing

`roaring-tags` uses RoaringBitmap to efficiently store and query object-tag relationships:

```
Objects:
┌────┬───────┬─────────┐
│ ID │ Gender│  City   │
├────┼───────┼─────────┤
│ 1  │ Male  │ Beijing │
│ 2  │ Female│ Shanghai│
│ 3  │ Male  │ Beijing │
└────┴───────┴─────────┘

Bitmap Index:
Gender:Male    → [1,0,1]
Gender:Female  → [0,1,0]
City:Beijing   → [1,0,1]
City:Shanghai  → [0,1,0]

Query: Male AND Beijing
[1,0,1] AND [1,0,1] = [1,0,1] → Objects [1,3]
```

### Why RoaringBitmap?

| Solution | Memory Usage | Query Speed | Complex Queries |
|----------|-------------|-------------|-----------------|
| **Database JOIN** | ❌ High | ❌ Slow | ⚠️ Complex |
| **HashSet** | ❌ High | ✅ Fast | ⚠️ Slow |
| **Redis SET** | ❌ High | ⚠️ Network | ❌ Slow |
| **roaring-tags** | ✅ Low | ✅ Fast | ✅ Milliseconds |

## 💡 Use Cases

```go
// 1. User Profiling
result, _ := ts.QueryAnd([]string{"active", "paid", "30days"})

// 2. Audience Segmentation
result, _ := ts.QueryOr([]string{"high_value", "potential", "churned"})

// 3. Content Recommendations
result, _ := ts.QueryAnd([]string{"tech", "ai", "subscribed"})

// 4. Permission Management
result, _ := ts.QueryAnd([]string{"admin", "active", "verified"})

// 5. Complex Queries
result, _ := ts.ComplexQuery([]tagbox.QueryOp{
    {Type: "AND", Tags: []string{"vip", "active"}},
    {Type: "OR", Tags: []string{"new_user", "referred"}},
})
```

## 📊 Benchmarks

**Environment:** Apple M2, 16GB RAM, Go 1.21

```
BenchmarkTagSystem_AddTag-8      37,162,394    35.20 ns/op    0 B/op    0 allocs/op
BenchmarkTagSystem_Query-8         749,277   1661 ns/op    8404 B/op    7 allocs/op
BenchmarkTagSystem_QueryAnd-8      433,347   3143 ns/op    8404 B/op    7 allocs/op
BenchmarkTagSystem_HasTag-8     69,319,584    27.33 ns/op    0 B/op    0 allocs/op
```

**Memory Usage (10M objects, 100 tags):**
- Traditional approach: ~125 MB
- roaring-tags: ~15 MB
- **Savings: 88%**

## 🔧 Configuration

```go
type Config struct {
    // Redis connection
    RedisAddr     string        // Redis server address
    RedisPassword string        // Redis password
    RedisDB       int           // Redis database number

    // Tag storage
    KeyPrefix     string        // Redis key prefix (default: "tags:")

    // Persistence
    AutoSave      bool          // Auto-save to Redis (default: true)

    // Performance tuning
    EnableSnapshot    bool          // Enable disk snapshots
    SnapshotPath      string        // Snapshot file path
    SnapshotInterval  time.Duration // Snapshot interval
}
```

## 📚 API Reference

### Core Operations

```go
// Add tags
ts.AddTag(objectID uint32, tag string) error
ts.BatchAddTags(objectID uint32, tags []string) error
ts.BatchAddObjectsToTag(objectIDs []uint32, tag string) error

// Remove tags
ts.RemoveTag(objectID uint32, tag string) error

// Check tags
ts.HasTag(objectID uint32, tag string) bool
ts.GetObjectTags(objectID uint32) ([]string, error)

// Query operations
ts.Query(tag string) (*roaring.Bitmap, error)
ts.QueryAnd(tags []string) (*roaring.Bitmap, error)
ts.QueryOr(tags []string) (*roaring.Bitmap, error)
ts.QueryNot(tag string, allObjects *roaring.Bitmap) (*roaring.Bitmap, error)
ts.QueryDifference(tag1, tag2 string) (*roaring.Bitmap, error)
ts.QueryXor(tag1, tag2 string) (*roaring.Bitmap, error)
ts.ComplexQuery(ops []QueryOp) (*roaring.Bitmap, error)

// Statistics
ts.GetTagCount(tag string) (uint64, error)
ts.GetStats() Stats
ts.GetAllTags() []string

// Persistence
ts.SaveToRedis() error
ts.RecoverFromRedis() error
ts.SaveSnapshot(filePath string) error
ts.LoadSnapshot(filePath string) error
ts.Close() error
```

### Helper Functions

```go
// Convert bitmap to array
tagbox.GetObjectIDs(bitmap *roaring.Bitmap) []uint32

// Count objects in bitmap
tagbox.Count(bitmap *roaring.Bitmap) uint64

// Check if object ID is in bitmap
tagbox.Contains(bitmap *roaring.Bitmap, objectID uint32) bool
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## 📁 Examples

- [Basic Usage](examples/basic/main.go) - Getting started guide
- [User Profiling](examples/user_profiling/main.go) - Real-world user segmentation

## 🎯 Roadmap

- [x] Core bitmap indexing
- [x] Redis persistence
- [x] Complex queries (AND/OR/NOT)
- [x] Comprehensive tests
- [ ] Distributed sharding
- [ ] Tag hierarchy support
- [ ] Real-time tag computation
- [ ] Prometheus metrics
- [ ] Admin dashboard

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [RoaringBitmap](https://github.com/RoaringBitmap/roaring) - High-performance compressed bitmaps
- [go-redis](https://github.com/redis/go-redis) - Redis client for Go

## 📞 Support

- 📧 Email: gongvirgil@gmail.com
- 🐛 Issues: [GitHub Issues](https://github.com/gongvirgil/roaring-tags/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/gongvirgil/roaring-tags/discussions)

---

<div align="center">

**⭐️ If this project helps you, please consider giving it a star!**

Made with ❤️ by [@gongvirgil](https://github.com/gongvirgil)

</div>
