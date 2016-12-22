// Package `rotate` provides an io.Writer interface for files that will detect when the open
// file has been rotated away (due to log rotation, or manual move/deletion) and re-open it.
// This allows the standard log.Logger to continue working as expected after log rotation
// happens (without needing to specify the `copytruncate` or equivalient options).
//
// This package is safe to use concurrently from multiple goroutines
package rotate

import (
	"os"
	"sync"
)

// RotatableFileWriter implementation that knows when the file has been rotated and re-opens it
type RotatableFileWriter struct {
	sync.Mutex
	file     *os.File
	fileInfo *os.FileInfo
	mode     os.FileMode
	name     string
}

// Close closes the underlying file
func (f *RotatableFileWriter) Close() error {
	f.Lock()
	err := f.file.Close()
	f.Unlock()

	return err
}

// reopen provides the (not exported, not concurrency safe) implementation of re-opening the file and updates the struct's fileInfo
func (f *RotatableFileWriter) reopen() error {
	if f.file != nil {
		f.file.Close()
		f.file = nil
		f.fileInfo = nil
	}

	reopened, err := os.OpenFile(f.name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, f.mode)
	if err != nil {
		return err
	}

	f.file = reopened

	fileInfo, err := os.Stat(f.name)
	if err != nil {
		return err
	}

	f.fileInfo = &fileInfo

	return nil
}

// Reopen provides the concurrency safe implementation of re-opening the file, and updating the struct's fileInfo
func (f *RotatableFileWriter) Reopen() error {
	f.Lock()
	err := f.reopen()
	f.Unlock()

	return err
}

// Write implements the standard io.Writer interface, but checks whether or not the file
// has changed prior to writing. If it has, it will reopen the file first, then write
func (f *RotatableFileWriter) Write(p []byte) (int, error) {
	f.Lock()
	defer f.Unlock() // Defer unlock due to the possibility of early return

	currentFileInfo, err := os.Stat(f.name)
	if err != nil {
		// os.Stat will throw an error if the file doesn't exist (ie: it was moved/rotated/deleted)
		// this specific error is not fatal, and passing along the invalid os.FileInfo pointer causes
		// the os.SameFile check to fail. This is the desired behavior
		if !os.IsNotExist(err) {
			return 0, err
		}
	}

	if !os.SameFile(*f.fileInfo, currentFileInfo) {
		err := f.reopen()
		if err != nil {
			return 0, err
		}

		f.fileInfo = &currentFileInfo
	}

	bytesWritten, err := f.file.Write(p)

	return bytesWritten, err
}

// NewRotatableFileWriter opens a file for appending and writing that can be safely rotated
func NewRotatableFileWriter(name string, mode os.FileMode) (*RotatableFileWriter, error) {
	rotatableFileWriter := RotatableFileWriter{
		file:     nil,
		name:     name,
		mode:     mode,
		fileInfo: nil,
	}

	err := rotatableFileWriter.reopen()
	if err != nil {
		return nil, err
	}

	return &rotatableFileWriter, nil
}
