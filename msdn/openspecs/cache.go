package openspecs

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// CacheFS is a simple file-system cache rooted at a directory. Files are read
// and written using paths relative to that root.
type CacheFS struct {
	Root string
}

// NewCacheFS creates a new CacheFS with the given root directory.
func NewCacheFS(root string) *CacheFS {
	return &CacheFS{Root: root}
}

// abs resolves a relative path against the cache root.
func (c *CacheFS) abs(rel string) string {
	return filepath.Join(c.Root, filepath.FromSlash(rel))
}

// Write creates (or overwrites) the file at rel, creating any parent directories
// as needed, and writes all data from r into it.
func (c *CacheFS) Write(rel string, r io.Reader) error {
	p := c.abs(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// Read opens the file at rel and returns a ReadCloser. The caller is responsible
// for closing it. Returns os.ErrNotExist if the file is not cached.
func (c *CacheFS) Read(rel string) (io.ReadCloser, error) {
	return os.Open(c.abs(rel))
}

// Has reports whether the file at rel exists in the cache.
func (c *CacheFS) Has(rel string) bool {
	_, err := os.Stat(c.abs(rel))
	return err == nil
}

// Unmarshaler is an interface that types can implement to unmarshal themselves from an io.Reader.
type Cacher interface {
	// Unmarshal reads data from the provided io.Reader and populates the implementing type with it.
	Unmarshal(context.Context, io.Reader) error
	// Marshal returns an io.Reader that produces the data to be cached for the implementing type.
	Marshal(context.Context) (io.Reader, error)
}
