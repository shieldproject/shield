package s3

import (
	"time"
)

type Object struct {
	Key          string
	LastModified time.Time
	ETag         string
	Size         Bytes
	StorageClass string
	OwnerID      string
	OwnerName    string
}
