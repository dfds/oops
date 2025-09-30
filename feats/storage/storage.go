package storage

import "context"

type Storage interface {
	Put(ctx context.Context, path string, content []byte) error
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) bool
}
