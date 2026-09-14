package videos

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFreeBytes_RealTempDir(t *testing.T) {
	dir := t.TempDir()
	free, err := FreeBytes(dir)
	if err != nil {
		t.Fatalf("FreeBytes: %v", err)
	}
	if free == 0 {
		t.Fatalf("FreeBytes returned 0 for a real temp dir; expected > 0")
	}
}

func TestFreeBytes_NonexistentDir(t *testing.T) {
	if _, err := FreeBytes("/this/path/does/not/exist/teamwerk-test"); err == nil {
		t.Fatal("FreeBytes on a nonexistent path must return an error")
	}
}

func TestRequireFreeBytes(t *testing.T) {
	dir := t.TempDir()
	free, err := FreeBytes(dir)
	if err != nil {
		t.Fatalf("FreeBytes: %v", err)
	}

	// needed + reserved well within the free space → ok.
	if err := RequireFreeBytes(dir, 1024, 1024); err != nil {
		t.Errorf("RequireFreeBytes with tiny demand should pass, got %v", err)
	}

	// needed + reserved exceeds free space → ErrInsufficientDiskSpace.
	err = RequireFreeBytes(dir, free, 1)
	if !errors.Is(err, ErrInsufficientDiskSpace) {
		t.Errorf("RequireFreeBytes over capacity = %v, want ErrInsufficientDiskSpace", err)
	}

	// reserved alone exceeding free space → ErrInsufficientDiskSpace.
	err = RequireFreeBytes(dir, 0, free+1)
	if !errors.Is(err, ErrInsufficientDiskSpace) {
		t.Errorf("RequireFreeBytes reserved over capacity = %v, want ErrInsufficientDiskSpace", err)
	}
}

func TestRequireFreeBytes_NonexistentDirReturnsStatError(t *testing.T) {
	err := RequireFreeBytes("/this/path/does/not/exist/teamwerk-test", 1, 1)
	if err == nil {
		t.Fatal("expected an error for a nonexistent dir")
	}
	if errors.Is(err, ErrInsufficientDiskSpace) {
		t.Errorf("a stat error must not be reported as ErrInsufficientDiskSpace, got %v", err)
	}
}

func TestStorageStats_RealTempDir(t *testing.T) {
	dir := t.TempDir()
	free, total, err := StorageStats(dir)
	if err != nil {
		t.Fatalf("StorageStats: %v", err)
	}
	if free == 0 || total == 0 {
		t.Fatalf("StorageStats returned free=%d total=%d for a real temp dir; expected both > 0", free, total)
	}
	if free > total {
		t.Errorf("free (%d) must not exceed total (%d)", free, total)
	}
}

func TestStorageStats_NonexistentDir(t *testing.T) {
	if _, _, err := StorageStats("/this/path/does/not/exist/teamwerk-test"); err == nil {
		t.Fatal("StorageStats on a nonexistent path must return an error")
	}
}

func TestDirSize_SummiertRekursiv(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.ts"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "b.ts"), []byte("12345678"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize: %v", err)
	}
	if want := int64(4 + 8); got != want {
		t.Errorf("DirSize = %d, want %d", got, want)
	}
}

func TestDirSize_NonexistentDirReturnsZero(t *testing.T) {
	got, err := DirSize(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("DirSize on a missing dir should not error, got %v", err)
	}
	if got != 0 {
		t.Errorf("DirSize on a missing dir = %d, want 0", got)
	}
}
