package main

import (
	"fmt"

	"github.com/gongvirgil/roaring-tags/roaring-tags/pkg/tagbox"
)

func main() {
	// Create a new tag system
	config := tagbox.DefaultConfig()
	config.RedisAddr = "localhost:6379"
	config.AutoSave = true // Enable auto-save to Redis

	ts, err := tagbox.New(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create tag system: %v", err))
	}
	defer ts.Close()

	// Setup: Simulate a user profiling system
	fmt.Println("=== Setting up user profiles ===")

	// User segments
	users := []struct {
		id       uint32
		tags     []string
		name     string
	}{
		{1001, []string{"premium", "active", "mobile", "30days"}, "Alice"},
		{1002, []string{"free", "inactive", "desktop", "90days"}, "Bob"},
		{1003, []string{"premium", "active", "mobile", "7days"}, "Charlie"},
		{1004, []string{"premium", "inactive", "mobile", "60days"}, "Diana"},
		{1005, []string{"free", "active", "desktop", "30days"}, "Eve"},
		{1006, []string{"premium", "active", "desktop", "30days"}, "Frank"},
	}

	for _, user := range users {
		ts.BatchAddTags(user.id, user.tags)
		fmt.Printf("User %d (%s): %v\n", user.id, user.name, user.tags)
	}

	// Use Case 1: Find active premium users for marketing campaign
	fmt.Println("\n=== Use Case 1: Active Premium Users ===")
	result, _ := ts.QueryAnd([]string{"premium", "active"})
	userIDs := tagbox.GetObjectIDs(result)
	fmt.Printf("Target audience (Premium AND Active): %v\n", userIDs)
	fmt.Printf("Count: %d users\n", tagbox.Count(result))

	// Use Case 2: Find at-risk users (premium but inactive)
	fmt.Println("\n=== Use Case 2: At-Risk Users (Premium but Inactive) ===")
	result, _ = ts.QueryAnd([]string{"premium"})
	inactive, _ := ts.Query("inactive")
	atRisk := result.Clone()
	atRisk.And(inactive)
	fmt.Printf("At-risk users: %v\n", tagbox.GetObjectIDs(atRisk))

	// Use Case 3: Mobile user engagement analysis
	fmt.Println("\n=== Use Case 3: Mobile Active Users ===")
	result, _ = ts.QueryAnd([]string{"mobile", "active"})
	fmt.Printf("Active mobile users: %v\n", tagbox.GetObjectIDs(result))

	// Use Case 4: Recent user acquisition (last 30 days)
	fmt.Println("\n=== Use Case 4: Recent Users (30 days) ===")
	result, _ = ts.Query("30days")
	fmt.Printf("Users acquired in last 30 days: %v\n", tagbox.GetObjectIDs(result))

	// Use Case 5: Churn risk analysis (inactive OR 90days)
	fmt.Println("\n=== Use Case 5: Churn Risk Analysis ===")
	result, _ = ts.QueryOr([]string{"inactive", "90days"})
	fmt.Printf("Users at churn risk: %v\n", tagbox.GetObjectIDs(result))

	// Use Case 6: Cross-platform users (both mobile and desktop)
	fmt.Println("\n=== Use Case 6: Cross-Platform Users ===")
	mobile, _ := ts.Query("mobile")
	desktop, _ := ts.Query("desktop")
	crossPlatform := mobile.Clone()
	crossPlatform.And(desktop)
	fmt.Printf("Cross-platform users (empty expected): %v\n", tagbox.GetObjectIDs(crossPlatform))

	// Use Case 7: Premium tier upgrade candidates (free AND active)
	fmt.Println("\n=== Use Case 7: Upgrade Candidates ===")
	result, _ = ts.QueryAnd([]string{"free", "active"})
	fmt.Printf("Free active users (upgrade candidates): %v\n", tagbox.GetObjectIDs(result))

	// Use Case 8: Reactivation campaign target (inactive AND premium)
	fmt.Println("\n=== Use Case 8: Reactivation Campaign ===")
	result, _ = ts.QueryAnd([]string{"inactive", "premium"})
	fmt.Printf("Premium inactive users: %v\n", tagbox.GetObjectIDs(result))

	// Use Case 9: Feature adoption analysis
	fmt.Println("\n=== Use Case 9: Mobile Adoption Rate ===")
	mobileUsers, _ := ts.Query("mobile")
	totalUsers := uint32(1000) // Assume 1000 total users
	adoptionRate := float64(tagbox.Count(mobileUsers)) / float64(totalUsers) * 100
	fmt.Printf("Mobile adoption rate: %.2f%%\n", adoptionRate)

	// Use Case 10: Complex audience segmentation
	fmt.Println("\n=== Use Case 10: Complex Segmentation ===")
	// Find premium users who are either active on mobile OR have been inactive for 60+ days
	premium, _ := ts.Query("premium")
	activeMobile, _ := ts.QueryAnd([]string{"active", "mobile"})
	longInactive, _ := ts.Query("60days")
	target := activeMobile.Clone()
	target.Or(longInactive)
	target.And(premium)
	fmt.Printf("Complex segment (Premium AND (Active+Mobile OR 60days)): %v\n", tagbox.GetObjectIDs(target))

	// Statistics
	fmt.Println("\n=== System Statistics ===")
	stats := ts.GetStats()
	fmt.Printf("Total tags: %d\n", stats.TotalTags)
	fmt.Printf("Total tagged objects: %d\n", stats.UniqueObjects)
	fmt.Printf("Memory usage: %.2f KB\n", float64(stats.MemoryUsage)/1024)
	fmt.Printf("Largest segment: %s (%d users)\n", stats.LargestTag, stats.LargestTagSize)

	// Per-tag statistics
	fmt.Println("\n=== Per-Tag Statistics ===")
	allTags := ts.GetAllTags()
	for _, tag := range allTags {
		count, _ := ts.GetTagCount(tag)
		fmt.Printf("  %s: %d users\n", tag, count)
	}
}
