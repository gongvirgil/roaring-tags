package main

import (
	"fmt"

	"github.com/gongvirgil/roaring-tags/roaring-tags/pkg/tagbox"
)

func main() {
	// Create a new tag system with default configuration
	config := tagbox.DefaultConfig()
	config.RedisAddr = "localhost:6379" // Change this to your Redis address

	ts, err := tagbox.New(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create tag system: %v", err))
	}
	defer ts.Close()

	// Example 1: Add tags to objects
	fmt.Println("=== Example 1: Add tags ===")
	ts.AddTag(1, "vip")
	ts.AddTag(1, "male")
	ts.AddTag(1, "active")

	ts.AddTag(2, "vip")
	ts.AddTag(2, "female")
	ts.AddTag(2, "active")

	ts.AddTag(3, "regular")
	ts.AddTag(3, "male")
	ts.AddTag(3, "active")

	// Example 2: Check if an object has a tag
	fmt.Println("\n=== Example 2: Check tags ===")
	hasTag := ts.HasTag(1, "vip")
	fmt.Printf("User 1 has VIP tag: %v\n", hasTag)

	// Example 3: Get all tags for an object
	fmt.Println("\n=== Example 3: Get object tags ===")
	tags, _ := ts.GetObjectTags(1)
	fmt.Printf("User 1 tags: %v\n", tags)

	// Example 4: Single tag query
	fmt.Println("\n=== Example 4: Single tag query ===")
	result, _ := ts.Query("vip")
	fmt.Printf("VIP users: %v\n", tagbox.GetObjectIDs(result))

	// Example 5: AND query (users with multiple tags)
	fmt.Println("\n=== Example 5: AND query ===")
	result, _ = ts.QueryAnd([]string{"vip", "male"})
	fmt.Printf("VIP AND male users: %v\n", tagbox.GetObjectIDs(result))

	// Example 6: OR query (users with any of the tags)
	fmt.Println("\n=== Example 6: OR query ===")
	result, _ = ts.QueryOr([]string{"vip", "regular"})
	fmt.Printf("VIP OR regular users: %v\n", tagbox.GetObjectIDs(result))

	// Example 7: NOT query (users without a specific tag)
	fmt.Println("\n=== Example 7: NOT query ===")
	result, _ = ts.QueryNotInSystem("vip")
	fmt.Printf("Non-VIP users: %v\n", tagbox.GetObjectIDs(result))

	// Example 8: Get tag statistics
	fmt.Println("\n=== Example 8: Statistics ===")
	stats := ts.GetStats()
	fmt.Printf("Total tags: %d\n", stats.TotalTags)
	fmt.Printf("Unique objects: %d\n", stats.UniqueObjects)
	fmt.Printf("Largest tag: %s (%d objects)\n", stats.LargestTag, stats.LargestTagSize)
	fmt.Printf("Memory usage: %d bytes\n", stats.MemoryUsage)

	// Example 9: Batch operations
	fmt.Println("\n=== Example 9: Batch operations ===")
	ts.BatchAddTags(10, []string{"new", "verified", "premium"})
	tags, _ = ts.GetObjectTags(10)
	fmt.Printf("User 10 tags after batch add: %v\n", tags)

	objectIDs := []uint32{20, 21, 22}
	ts.BatchAddObjectsToTag(objectIDs, "batch_tag")
	count, _ := ts.GetTagCount("batch_tag")
	fmt.Printf("Batch added %d objects to 'batch_tag'\n", count)

	// Example 10: Difference query
	fmt.Println("\n=== Example 10: Difference query ===")
	result, _ = ts.QueryDifference("vip", "male")
	fmt.Printf("VIP but NOT male users: %v\n", tagbox.GetObjectIDs(result))

	// Example 11: XOR query
	fmt.Println("\n=== Example 11: XOR query ===")
	result, _ = ts.QueryXor("vip", "regular")
	fmt.Printf("VIP XOR regular users: %v\n", tagbox.GetObjectIDs(result))
}
