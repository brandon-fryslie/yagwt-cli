package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bmf/yagwt/internal/errors"
	"golang.org/x/sys/unix"
)

// Lock provides concurrency control
type Lock interface {
	Acquire(timeout time.Duration) error
	Release() error
}

// Manager creates and manages locks
type Manager interface {
	NewLock(path string) (Lock, error)
}

// fileLock implements file-based advisory locking using flock
type fileLock struct {
	path string
	file *os.File
}

type manager struct{}

// NewManager creates a new lock manager
func NewManager() Manager {
	return &manager{}
}

// NewLock creates a new file-based lock
func (m *manager) NewLock(path string) (Lock, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.WrapError(errors.ErrConfig, "failed to create lock directory", err).
			WithDetail("path", dir)
	}

	return &fileLock{
		path: path,
	}, nil
}

// Acquire acquires the lock with a timeout
func (l *fileLock) Acquire(timeout time.Duration) error {
	// Open or create the lock file
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return errors.WrapError(errors.ErrConfig, "failed to open lock file", err).
			WithDetail("path", l.path)
	}

	// Try to acquire the lock with timeout
	deadline := time.Now().Add(timeout)
	pollInterval := 10 * time.Millisecond

	for {
		// Try to acquire exclusive lock (non-blocking)
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			l.file = file
			return nil
		}

		// Check if the error is because the lock is held by another process
		if err != unix.EWOULDBLOCK {
			file.Close()
			return errors.WrapError(errors.ErrConfig, "failed to acquire lock", err).
				WithDetail("path", l.path)
		}

		// Lock is held by another process, check timeout
		if time.Now().After(deadline) {
			file.Close()
			return errors.NewError(errors.ErrLocked, "lock acquisition timed out").
				WithDetail("path", l.path).
				WithDetail("timeout", timeout.String()).
				WithHint("Another process may be holding the lock", "")
		}

		// Wait before retrying
		time.Sleep(pollInterval)

		// Exponential backoff up to 100ms
		if pollInterval < 100*time.Millisecond {
			pollInterval *= 2
		}
	}
}

// Release releases the lock
func (l *fileLock) Release() error {
	if l.file == nil {
		return errors.NewError(errors.ErrConfig, "lock not acquired")
	}

	// Release the flock
	if err := unix.Flock(int(l.file.Fd()), unix.LOCK_UN); err != nil {
		l.file.Close()
		l.file = nil
		return errors.WrapError(errors.ErrConfig, "failed to release lock", err).
			WithDetail("path", l.path)
	}

	// Close the file
	if err := l.file.Close(); err != nil {
		l.file = nil
		return errors.WrapError(errors.ErrConfig, "failed to close lock file", err).
			WithDetail("path", l.path)
	}

	l.file = nil
	return nil
}

// String returns a string representation for debugging
func (l *fileLock) String() string {
	acquired := "not acquired"
	if l.file != nil {
		acquired = "acquired"
	}
	return fmt.Sprintf("FileLock{path=%s, status=%s}", l.path, acquired)
}
