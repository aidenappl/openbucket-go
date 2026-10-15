package objects

// objectUpsert makes writing a key replace its metadata row, as an S3 PUT
// replaces the object.
//
// ⚠️ Uploads used to INSERT a new row on every overwrite, with nothing
// enforcing one row per key: every listing then repeated the key (atlas
// hotels/250/ listed 15 files as 40 rows) and the console showed duplicates.
// db.RunMigrations removes existing duplicates and adds the unique index this
// ON CONFLICT relies on (uniq_objects_bucket_key).
//
// The row keeps its id, so object tags survive an overwrite.
const objectUpsert = `ON CONFLICT (bucket_id, key) DO UPDATE SET
	etag = EXCLUDED.etag,
	owner_id = EXCLUDED.owner_id,
	public = EXCLUDED.public,
	last_modified = EXCLUDED.last_modified,
	uploaded_at = EXCLUDED.uploaded_at,
	version_id = EXCLUDED.version_id,
	size = EXCLUDED.size`
