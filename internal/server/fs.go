package server

import (
	"io/fs"
	"sync"
	"time"

	"github.com/sgq995/nova/internal/logger"
)

type memFS struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMemFS() *memFS {
	return &memFS{
		files: map[string][]byte{},
	}
}

func (fsys *memFS) update(filename string, contents []byte) {
	fsys.mu.Lock()
	defer fsys.mu.Unlock()
	logger.Debugf("[server] update %s\n", filename)
	fsys.files[filename] = contents
}

func (fsys *memFS) remove(filename string) {
	fsys.mu.Lock()
	defer fsys.mu.Unlock()
	logger.Debugf("[server] remove %s\n", filename)
	delete(fsys.files, filename)
}

func (fsys *memFS) Open(name string) (fs.File, error) {
	if f, exists := fsys.files[name]; exists {
		return &memFile{
			name:     name,
			size:     int64(len(f)),
			contents: f,
		}, nil
	}
	return nil, fs.ErrNotExist
}

type memFile struct {
	name     string
	size     int64
	contents []byte
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return &memFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *memFile) Read(out []byte) (int, error) {
	n := copy(out, f.contents)
	f.contents = f.contents[n:]
	return n, nil
}

func (f *memFile) Close() error {
	return nil
}

type memFileInfo struct {
	name string
	size int64
}

func (fi *memFileInfo) Name() string {
	return fi.name
}

func (fi *memFileInfo) Size() int64 {
	return fi.size
}

func (fi *memFileInfo) Mode() fs.FileMode {
	return 0755
}

func (fi *memFileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *memFileInfo) IsDir() bool {
	return false
}

func (fi *memFileInfo) Sys() any {
	return nil
}
