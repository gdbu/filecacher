package filecacher

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"

	"github.com/gdbu/atoms"

	"github.com/gdbu/errors"
	"github.com/gdbu/poller"
)

// NewFile will return a new file
func NewFile(filename string) (fp *File, err error) {
	var f File
	f.b = bytes.NewBuffer(nil)
	f.filename = filename

	if err = f.refreshBuffer(); err != nil {
		return
	}

	if f.p, err = poller.New(filename, f.onEvent); err != nil {
		return
	}

	go f.p.Run(0)
	fp = &f
	return
}

// File represents a file
type File struct {
	mu sync.RWMutex

	p *poller.Poller
	b *bytes.Buffer

	filename string

	closed atoms.Bool
}

func (f *File) onEvent(e poller.Event) {
	switch e {
	case poller.EventWrite:
		if err := f.refreshBuffer(); err != nil {
			log.Printf("filecacher: error refreshing buffer: %v\n", err)
			return
		}
	case poller.EventRemove:
		f.Close()
	}
}

func (f *File) refreshBuffer() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var tgt *os.File
	if tgt, err = os.Open(f.filename); err != nil {
		if os.IsNotExist(err) {
			err = ErrFileNotFound
		}

		return
	}
	defer tgt.Close()

	f.b.Reset()
	if _, err = io.Copy(f.b, tgt); err != nil {
		return
	}

	return
}

// Read will read a file
func (f *File) Read(fn func(io.Reader) error) (err error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed.Get() {
		return errors.ErrIsClosed
	}

	r := bytes.NewReader(f.b.Bytes())

	return fn(r)
}

// Close will close a file
func (f *File) Close() (err error) {
	if !f.closed.Set(true) {
		return errors.ErrIsClosed
	}

	f.p.Close()
	return
}
