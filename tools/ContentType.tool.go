package tools

import (
	"path/filepath"
	"strings"
)

func ContentType(key string) string {
	ext := strings.ToLower(filepath.Ext(key))
	var contentType string

	// Define MIME types for common file extensions
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".bmp":
		contentType = "image/bmp"
	case ".svg":
		contentType = "image/svg+xml"
	case ".webp":
		contentType = "image/webp"
	case ".ico":
		contentType = "image/x-icon"
	case ".tiff", ".tif":
		contentType = "image/tiff"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".wav":
		contentType = "audio/wav"
	case ".ogg":
		contentType = "audio/ogg"
	case ".m4a":
		contentType = "audio/mp4"
	case ".mp4":
		contentType = "video/mp4"
	case ".webm":
		contentType = "video/webm"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	case ".pdf":
		contentType = "application/pdf"
	case ".json":
		contentType = "application/json"
	case ".xml":
		contentType = "application/xml"
	case ".txt":
		contentType = "text/plain"
	case ".html", ".htm":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".csv":
		contentType = "text/csv"
	case ".zip":
		contentType = "application/zip"
	case ".tar":
		contentType = "application/x-tar"
	case ".gzip", ".gz":
		contentType = "application/gzip"
	case ".rar":
		contentType = "application/x-rar-compressed"
	case ".7z":
		contentType = "application/x-7z-compressed"
	default:
		contentType = "application/octet-stream" // Default for other types
	}

	return contentType
}
