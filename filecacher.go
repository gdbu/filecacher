package filecacher

import (
	"io"
	"path/filepath"
	"sync"

	"github.com/gdbu/errors"
)

const (
	// ErrFileNotFound is returned when a requested file has not been found
	ErrFileNotFound = errors.Error("file not found")
	// ErrFileExists is returned when a file is attempted to be created when it already exists
	ErrFileExists = errors.Error("file already exists")
)

// New will return a new instance of FileCacher
func New(root string) *FileCacher {
	var f FileCacher
	f.m = make(map[string]*File)
	f.root = root
	return &f
}

// FileCacher will manage an instance of file cacher
type FileCacher struct {
	mu sync.RWMutex

	m map[string]*File

	root   string
	closed bool
}

func (f *FileCacher) create(key, filename string) (file *File, err error) {
	if file, err = NewFile(filename); err != nil {
		return
	}

	f.m[key] = file
	return
}

func (f *FileCacher) get(key string) (file *File, err error) {
	var ok bool
	if file, ok = f.m[key]; !ok || file.closed.Get() {
		err = ErrFileNotFound
		return
	}

	return
}

// New will create a file
func (f *FileCacher) New(key string) (file *File, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		err = errors.ErrIsClosed
		return
	}

	if file, err = f.get(key); err == nil {
		return nil, ErrFileExists
	}

	filename := filepath.Join(f.root, key)
	return f.create(key, filename)
}

// Get will retrieve a file
func (f *FileCacher) Get(key string) (file *File, err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		err = errors.ErrIsClosed
		return
	}

	return f.get(key)
}

// GetOrCreate attempt to retrieve a file, if it does not exist - it will create it
func (f *FileCacher) GetOrCreate(key string) (file *File, err error) {
	if file, err = f.Get(key); err == nil {
		return
	}

	return f.New(key)
}

// Read will read a file
func (f *FileCacher) Read(key string, fn func(io.Reader) error) (err error) {
	var file *File
	if file, err = f.GetOrCreate(key); err != nil {
		return
	}

	return file.Read(fn)
}

// Unmount will remove a file from being opened and watched
func (f *FileCacher) Unmount(key string) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		err = errors.ErrIsClosed
		return
	}

	var file *File
	if file, err = f.get(key); err != nil {
		return
	}

	delete(f.m, key)
	file.Close()
	return
}

// Close will close the file cacher
func (f *FileCacher) Close() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return errors.ErrIsClosed
	}

	f.closed = true

	for _, ff := range f.m {
		ff.Close()
	}

	return
}
