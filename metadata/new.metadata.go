package metadata

import (
	"time"

	"github.com/aidenappl/openbucket-go/types"
)

func New(
	bucket, key string, etag string, public bool, owner string, size int64,
) *types.ObjectMetadata {
	return &types.ObjectMetadata{
		ETag:         etag,
		Key:          key,
		LastModified: types.IsoTime(time.Now()),
	}
}
