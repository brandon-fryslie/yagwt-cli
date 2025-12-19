package lock

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bmf/yagwt/internal/errors"
)

func TestLockAcquireRelease(t *testing.T) {
	// Create temp directory for lock file
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	mgr := NewManager()
	lock, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	// Acquire lock
	err = lock.Acquire(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Verify lock file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Release lock
	err = lock.Release()
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Verify we can acquire again
	err = lock.Acquire(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to re-acquire lock: %v", err)
	}

	lock.Release()
}

func TestLockTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	mgr := NewManager()

	// First lock acquires successfully
	lock1, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock1: %v", err)
	}

	err = lock1.Acquire(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock1: %v", err)
	}
	defer lock1.Release()

	// Second lock should timeout
	lock2, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock2: %v", err)
	}

	start := time.Now()
	err = lock2.Acquire(200 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected lock acquisition to fail with timeout")
	}

	// Verify it's a lock error
	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrLocked {
		t.Errorf("Expected error code %s, got %s", errors.ErrLocked, coreErr.Code)
	}

	// Verify timeout was respected (with some tolerance)
	if elapsed < 200*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("Timeout took %v, expected around 200ms", elapsed)
	}
}

func TestConcurrentLockAcquisition(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	mgr := NewManager()
	const numGoroutines = 10
	const holdDuration = 50 * time.Millisecond

	var (
		wg            sync.WaitGroup
		successCount  int
		mu            sync.Mutex
		acquisitions  []int
	)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := mgr.NewLock(lockPath)
			if err != nil {
				t.Logf("Goroutine %d: Failed to create lock: %v", id, err)
				return
			}

			err = lock.Acquire(2 * time.Second)
			if err != nil {
				t.Logf("Goroutine %d: Failed to acquire lock: %v", id, err)
				return
			}

			// Critical section
			mu.Lock()
			successCount++
			acquisitions = append(acquisitions, id)
			mu.Unlock()

			// Hold the lock briefly
			time.Sleep(holdDuration)

			lock.Release()
		}(i)
	}

	wg.Wait()

	// All goroutines should have successfully acquired the lock eventually
	if successCount != numGoroutines {
		t.Errorf("Expected %d successful acquisitions, got %d", numGoroutines, successCount)
		t.Logf("Successful acquisitions: %v", acquisitions)
	}
}

func TestLockReleaseWithoutAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	mgr := NewManager()
	lock, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	// Try to release without acquiring
	err = lock.Release()
	if err == nil {
		t.Fatal("Expected error when releasing lock that wasn't acquired")
	}

	coreErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("Expected *errors.Error, got %T", err)
	}

	if coreErr.Code != errors.ErrConfig {
		t.Errorf("Expected error code %s, got %s", errors.ErrConfig, coreErr.Code)
	}
}

func TestLockCreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "subdir", "deep", "test.lock")

	mgr := NewManager()
	lock, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	// Parent directories should be created when acquiring lock
	err = lock.Acquire(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Verify parent directories exist
	parentDir := filepath.Dir(lockPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Parent directory was not created")
	}
}

func TestMultipleAcquireReleaseCycles(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	mgr := NewManager()
	lock, err := mgr.NewLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	// Multiple acquire/release cycles
	for i := 0; i < 5; i++ {
		err = lock.Acquire(1 * time.Second)
		if err != nil {
			t.Fatalf("Failed to acquire lock on iteration %d: %v", i, err)
		}

		err = lock.Release()
		if err != nil {
			t.Fatalf("Failed to release lock on iteration %d: %v", i, err)
		}
	}
}
