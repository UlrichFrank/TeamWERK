package videos

import (
	"errors"
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
