package objects

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/aidenappl/openbucket-go/db"
	"github.com/aidenappl/openbucket-go/types"
)

func GetObjectTags(bucketID int, objectID int) ([]types.Tag, error) {
	rows, err := db.Psql.Select(
		"id",
		"object_id",
		"tag_key",
		"tag_value",
	).
		From("object_tags").
		Where(sq.Eq{"bucket_id": bucketID, "object_id": objectID}).
		OrderBy("id ASC").
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get object tags: %w", err)
	}
	defer rows.Close()

	var tags []types.Tag
	for rows.Next() {
		var tag types.Tag
		if err := rows.Scan(&tag.ID, &tag.ObjectID, &tag.Key, &tag.Value); err != nil {
			return nil, fmt.Errorf("failed to scan object tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// BatchGetObjectTags loads tags for all objects in a bucket in a single query,
// returning a map from object ID to its tags.
func BatchGetObjectTags(bucketID int) (map[int][]types.Tag, error) {
	rows, err := db.Psql.Select(
		"id",
		"object_id",
		"tag_key",
		"tag_value",
	).
		From("object_tags").
		Where(sq.Eq{"bucket_id": bucketID}).
		OrderBy("object_id ASC", "id ASC").
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to batch get object tags: %w", err)
	}
	defer rows.Close()

	result := make(map[int][]types.Tag)
	for rows.Next() {
		var tag types.Tag
		if err := rows.Scan(&tag.ID, &tag.ObjectID, &tag.Key, &tag.Value); err != nil {
			return nil, fmt.Errorf("failed to scan object tag: %w", err)
		}
		result[tag.ObjectID] = append(result[tag.ObjectID], tag)
	}

	return result, nil
}
