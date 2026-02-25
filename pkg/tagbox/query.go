package tagbox

import (
	"fmt"

	"github.com/RoaringBitmap/roaring"
)

// QueryOp represents a query operation type.
type QueryOp struct {
	Type string // "AND", "OR", "NOT"
	Tags []string
}

// Query returns objects that have a specific tag.
func (ts *TagSystem) Query(tag string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		return roaring.NewBitmap(), nil
	}

	return bitmap.Clone(), nil
}

// QueryAnd returns objects that have ALL the specified tags (intersection).
func (ts *TagSystem) QueryAnd(tags []string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if len(tags) == 0 {
		return roaring.NewBitmap(), nil
	}

	// Get first tag's bitmap
	firstBitmap, exists := ts.tags[tags[0]]
	if !exists {
		return roaring.NewBitmap(), nil
	}

	result := firstBitmap.Clone()

	// Intersect with other tags
	for _, tag := range tags[1:] {
		bitmap, exists := ts.tags[tag]
		if !exists {
			return roaring.NewBitmap(), nil // Tag doesn't exist, empty result
		}
		result.And(bitmap)
	}

	return result, nil
}

// QueryOr returns objects that have ANY of the specified tags (union).
func (ts *TagSystem) QueryOr(tags []string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := roaring.NewBitmap()

	for _, tag := range tags {
		bitmap, exists := ts.tags[tag]
		if exists {
			result.Or(bitmap)
		}
	}

	return result, nil
}

// QueryNot returns objects that do NOT have the specified tag.
// The allObjects parameter represents the universe of all objects.
func (ts *TagSystem) QueryNot(tag string, allObjects *roaring.Bitmap) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap, exists := ts.tags[tag]
	if !exists {
		return allObjects.Clone(), nil
	}

	result := allObjects.Clone()
	result.AndNot(bitmap)

	return result, nil
}

// QueryNotInSystem returns objects that do NOT have the specified tag,
// using the system's allObjects as the universe.
func (ts *TagSystem) QueryNotInSystem(tag string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	allObjectsClone := ts.allObjects.Clone()
	ts.mu.RUnlock()

	return ts.QueryNot(tag, allObjectsClone)
}

// ComplexQuery executes a complex query with multiple operations.
// Example:
//   [
//     {Type: "AND", Tags: ["male", "vip"]},
//     {Type: "OR", Tags: ["new_user", "referred"]}
//   ]
// This returns objects that are (male AND vip) OR (new_user OR referred).
func (ts *TagSystem) ComplexQuery(ops []QueryOp) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if len(ops) == 0 {
		return roaring.NewBitmap(), nil
	}

	var result *roaring.Bitmap

	for i, op := range ops {
		var partial *roaring.Bitmap
		var err error

		switch op.Type {
		case "AND":
			partial, err = ts.queryAndLocked(op.Tags)
		case "OR":
			partial, err = ts.queryOrLocked(op.Tags)
		case "NOT":
			if len(op.Tags) != 1 {
				return nil, fmt.Errorf("NOT operation requires exactly one tag")
			}
			partial = ts.queryNotLocked(op.Tags[0])
		default:
			return nil, fmt.Errorf("unknown operation: %s", op.Type)
		}

		if err != nil {
			return nil, err
		}

		if i == 0 {
			result = partial
		} else {
			// Combine results with AND by default
			// (operations are implicitly ANDed together)
			result.And(partial)
		}
	}

	return result, nil
}

// queryAndLocked performs AND query while holding read lock.
// Caller must hold ts.mu.RLock().
func (ts *TagSystem) queryAndLocked(tags []string) (*roaring.Bitmap, error) {
	if len(tags) == 0 {
		return roaring.NewBitmap(), nil
	}

	firstBitmap, exists := ts.tags[tags[0]]
	if !exists {
		return roaring.NewBitmap(), nil
	}

	result := firstBitmap.Clone()

	for _, tag := range tags[1:] {
		bitmap, exists := ts.tags[tag]
		if !exists {
			return roaring.NewBitmap(), nil
		}
		result.And(bitmap)
	}

	return result, nil
}

// queryOrLocked performs OR query while holding read lock.
// Caller must hold ts.mu.RLock().
func (ts *TagSystem) queryOrLocked(tags []string) (*roaring.Bitmap, error) {
	result := roaring.NewBitmap()

	for _, tag := range tags {
		bitmap, exists := ts.tags[tag]
		if exists {
			result.Or(bitmap)
		}
	}

	return result, nil
}

// queryNotLocked performs NOT query while holding read lock.
// Caller must hold ts.mu.RLock().
func (ts *TagSystem) queryNotLocked(tag string) *roaring.Bitmap {
	bitmap, exists := ts.tags[tag]
	if !exists {
		return ts.allObjects.Clone()
	}

	result := ts.allObjects.Clone()
	result.AndNot(bitmap)

	return result
}

// QueryDifference returns objects that are in tag1 but not in tag2.
func (ts *TagSystem) QueryDifference(tag1, tag2 string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap1, exists1 := ts.tags[tag1]
	if !exists1 {
		return roaring.NewBitmap(), nil
	}

	bitmap2, exists2 := ts.tags[tag2]
	if !exists2 {
		return bitmap1.Clone(), nil
	}

	result := bitmap1.Clone()
	result.AndNot(bitmap2)

	return result, nil
}

// QueryXor returns objects that are in exactly one of the tags (exclusive or).
func (ts *TagSystem) QueryXor(tag1, tag2 string) (*roaring.Bitmap, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bitmap1, exists1 := ts.tags[tag1]
	bitmap2, exists2 := ts.tags[tag2]

	if !exists1 && !exists2 {
		return roaring.NewBitmap(), nil
	}
	if !exists1 {
		return bitmap2.Clone(), nil
	}
	if !exists2 {
		return bitmap1.Clone(), nil
	}

	result := bitmap1.Clone()
	result.Xor(bitmap2)

	return result, nil
}

// GetObjectIDs returns the object IDs from a bitmap as a slice.
func GetObjectIDs(bitmap *roaring.Bitmap) []uint32 {
	return bitmap.ToArray()
}

// Count returns the number of objects in the bitmap.
func Count(bitmap *roaring.Bitmap) uint64 {
	return bitmap.GetCardinality()
}

// Contains checks if an object ID is in the bitmap.
func Contains(bitmap *roaring.Bitmap, objectID uint32) bool {
	return bitmap.Contains(objectID)
}
