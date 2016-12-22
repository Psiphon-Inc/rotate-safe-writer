package rotate

import (
	"os"
	"sync"
	"syscall"
)

// RotatableFileWriter implementation that knows when the file has been rotated and can handle it
type RotatableFileWriter struct {
	sync.Mutex
	file  *os.File
	mode  os.FileMode
	name  string
	inode uint64
}

// Get the current inode of the open file (not thread safe)
func (f *RotatableFileWriter) getCurrentInode() (uint64, error) {
	fileInfo, err := os.Stat(f.name)
	if err != nil {
		return 0, err
	}

	return fileInfo.Sys().(*syscall.Stat_t).Ino, nil
}

// Check if the current inode is different from the last known inode (not thread safe)
func (f *RotatableFileWriter) hasInodeChanged() bool {
	inode, _ := f.getCurrentInode()
	if f.inode != inode {
		return true
	}

	return false
}

// Closes the file
func (f *RotatableFileWriter) Close() error {
	f.Lock()
	err := f.file.Close()
	f.Unlock()

	return err
}

// Re-open the file (not thread safe)
func (f *RotatableFileWriter) reopen() error {

	if f.file != nil {
		f.file.Close()
		f.file = nil
		f.inode = 0
	}

	reopened, err := os.OpenFile(f.name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, f.mode)
	if err != nil {
		return err
	}

	f.file = reopened

	inode, err := f.getCurrentInode()
	if err != nil {
		return err
	}

	f.inode = inode

	return nil
}

// Re-open the file
func (f *RotatableFileWriter) Reopen() error {
	f.Lock()
	err := f.reopen()
	f.Unlock()

	return err
}

// Implements the standar io.Writer interface, but checks whether or not the inode
// has changed prior to writing. If it has, reopen the file first, then write
func (f *RotatableFileWriter) Write(p []byte) (int, error) {
	f.Lock()
	defer f.Unlock() // Defer unlock due to the possibility of early return

	if f.hasInodeChanged() {
		err := f.reopen()
		if err != nil {
			return 0, err
		}

		newInode, err := f.getCurrentInode()
		if err != nil {
			return 0, err
		}

		f.inode = newInode
	}

	bytesWritten, err := f.file.Write(p)

	return bytesWritten, err
}

// NewRotatableFileWriter opens a file for appending and writing and can be safely rotated
func NewRotatableFileWriter(name string, mode os.FileMode) (*RotatableFileWriter, error) {
	rotatableFileWriter := RotatableFileWriter{
		file:  nil,
		name:  name,
		mode:  mode,
		inode: 0,
	}

	err := rotatableFileWriter.reopen()
	if err != nil {
		return nil, err
	}

	return &rotatableFileWriter, nil
}
