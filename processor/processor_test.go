package processor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/selector"
)

func mustWriteTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fp := filepath.Join(dir, name)
	if err := os.WriteFile(fp, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return fp
}

func TestActionOnSuccess_Delete(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "a.txt", "hello")

	entry := config.ConfigEntry{
		Name:            "t1",
		SourceDirectory: tmp,
		ActionOnSuccess: "delete",
	}

	if err := ActionOnSuccess(entry, src); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got err=%v", err)
	}
}

func TestActionOnSuccess_NoneAndUnknown_NoOp(t *testing.T) {
	tmp := t.TempDir()
	src1 := mustWriteTempFile(t, tmp, "b.txt", "x")
	src2 := mustWriteTempFile(t, tmp, "c.txt", "y")

	noOp := config.ConfigEntry{SourceDirectory: tmp, ActionOnSuccess: ""}
	unk := config.ConfigEntry{SourceDirectory: tmp, ActionOnSuccess: "whoknows"}

	if err := ActionOnSuccess(noOp, src1); err != nil {
		t.Fatalf("none failed: %v", err)
	}
	if _, err := os.Stat(src1); err != nil {
		t.Fatalf("expected file to remain: %v", err)
	}

	if err := ActionOnSuccess(unk, src2); err != nil {
		t.Fatalf("unknown action should be no-op: %v", err)
	}
	if _, err := os.Stat(src2); err != nil {
		t.Fatalf("expected file to remain: %v", err)
	}
}

func TestActionOnSuccess_Archive_DefaultDest(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "d.txt", "data")

	entry := config.ConfigEntry{
		Name:            "archive-default",
		SourceDirectory: tmp,
		ActionOnSuccess: "archive",
		ArchiveDest:     "", // -> <SourceDirectory>/archive
	}

	if err := ActionOnSuccess(entry, src); err != nil {
		t.Fatalf("archive default failed: %v", err)
	}
	dest := filepath.Join(tmp, "archive", "d.txt")

	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected archived file at %s: %v", dest, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed, got err=%v", err)
	}
}

func TestActionOnSuccess_Archive_ConfiguredDest(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "custom-arc")
	src := mustWriteTempFile(t, tmp, "e.txt", "data")

	entry := config.ConfigEntry{
		Name:            "archive-configured",
		SourceDirectory: tmp,
		ActionOnSuccess: "archive",
		ArchiveDest:     dst,
	}

	if err := ActionOnSuccess(entry, src); err != nil {
		t.Fatalf("archive configured failed: %v", err)
	}
	dest := filepath.Join(dst, "e.txt")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected archived file at %s: %v", dest, err)
	}
}

func TestActionOnSuccess_Archive_FallbackWhenConfiguredMkdirFails(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "f.txt", "data")

	// Make ArchiveDest a FILE path to force MkdirAll error.
	badDir := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(badDir, []byte("file, not dir"), 0o644); err != nil {
		t.Fatalf("write badDir: %v", err)
	}

	entry := config.ConfigEntry{
		Name:            "archive-fallback",
		SourceDirectory: tmp,
		ActionOnSuccess: "archive",
		ArchiveDest:     badDir,
	}

	if err := ActionOnSuccess(entry, src); err != nil {
		t.Fatalf("archive fallback failed: %v", err)
	}
	dest := filepath.Join(tmp, "archive", "f.txt")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected archived file at fallback %s: %v", dest, err)
	}
}

func TestActionOnSuccess_Archive_RenameFailure(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "g.txt", "data")

	archiveDest := filepath.Join(tmp, "arc")
	if err := os.MkdirAll(archiveDest, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Create a directory named "g.txt" at the destination so rename fails.
	destDir := filepath.Join(archiveDest, "g.txt")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("mkdir destDir: %v", err)
	}

	entry := config.ConfigEntry{
		Name:            "archive-rename-fail",
		SourceDirectory: tmp,
		ActionOnSuccess: "archive",
		ArchiveDest:     archiveDest,
	}

	err := ActionOnSuccess(entry, src)
	if err == nil {
		t.Fatalf("expected rename failure, got nil")
	}
	// Source should remain after failure.
	if _, statErr := os.Stat(src); statErr != nil && runtime.GOOS != "windows" {
		t.Fatalf("expected source to remain after failure: %v", statErr)
	}
}

func newProcessingSet(t *testing.T) *selector.FileSelector {
	t.Helper()
	// Replace with your real constructor if named differently:
	ps := selector.NewFileSelector()
	return ps
}

func TestStartProcessor_Success_Local_Delete(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "in.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "t",
				SourceDirectory: tmp,
				TransferType:    "local",  // will not call SFTP
				ActionOnSuccess: "delete", // exercises success action
			},
		},
	}

	q := make(chan string, 1)
	q <- src
	close(q)

	ps := newProcessingSet(t)

	// Run synchronously; StartProcessor returns when channel closes.
	StartProcessor(cfg, q, ps)

	// File should have been deleted by ActionOnSuccess.
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected deleted file, got err=%v", err)
	}
}

func TestStartProcessor_UnmatchedFile_Returns(t *testing.T) {
	tmp := t.TempDir()
	other := filepath.Join(tmp, "outside.txt")
	if err := os.WriteFile(other, []byte("z"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := &config.ConfigData{
		Transfers: []config.ConfigEntry{
			{
				Name:            "t3",
				SourceDirectory: filepath.Join(tmp, "not-matching"),
				TransferType:    "local",
			},
		},
	}

	q := make(chan string, 1)
	q <- other
	close(q)

	ps := newProcessingSet(t)

	done := make(chan struct{})
	go func() {
		StartProcessor(cfg, q, ps)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("processor did not return")
	}

	// File should remain untouched.
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("expected file to remain: %v", err)
	}
}

func TestActionOnFail_Delete(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "a.txt", "hello")

	entry := config.ConfigEntry{
		Name:            "t1",
		SourceDirectory: tmp,
		ActionOnFail:    "delete",
	}

	if err := ActionOnFail(entry, src); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got err=%v", err)
	}
}

func TestActionOnFail_NoneAndUnknown_NoOp(t *testing.T) {
	tmp := t.TempDir()
	src1 := mustWriteTempFile(t, tmp, "b.txt", "x")
	src2 := mustWriteTempFile(t, tmp, "c.txt", "y")

	noOp := config.ConfigEntry{SourceDirectory: tmp, ActionOnFail: ""}
	unk := config.ConfigEntry{SourceDirectory: tmp, ActionOnFail: "whoknows"}

	if err := ActionOnFail(noOp, src1); err != nil {
		t.Fatalf("none failed: %v", err)
	}
	if _, err := os.Stat(src1); err != nil {
		t.Fatalf("expected file to remain: %v", err)
	}

	if err := ActionOnFail(unk, src2); err != nil {
		t.Fatalf("unknown action should be no-op: %v", err)
	}
	if _, err := os.Stat(src2); err != nil {
		t.Fatalf("expected file to remain: %v", err)
	}
}

func TestActionOnFail_Fail_DefaultDest(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "d.txt", "data")

	entry := config.ConfigEntry{
		Name:            "fail-default",
		SourceDirectory: tmp,
		ActionOnFail:    "archive",
		FailDest:        "", // -> <SourceDirectory>/fail
	}

	if err := ActionOnFail(entry, src); err != nil {
		t.Fatalf("fail default failed: %v", err)
	}
	dest := filepath.Join(tmp, "fail", "d.txt")

	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected fail file at %s: %v", dest, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source removed, got err=%v", err)
	}
}

func TestActionOnFail_Fail_ConfiguredDest(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "custom-fail")
	src := mustWriteTempFile(t, tmp, "e.txt", "data")

	entry := config.ConfigEntry{
		Name:            "fail-configured",
		SourceDirectory: tmp,
		ActionOnFail:    "archive",
		FailDest:        dst,
	}

	if err := ActionOnFail(entry, src); err != nil {
		t.Fatalf("fail dir configured failed: %v", err)
	}
	dest := filepath.Join(dst, "e.txt")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected failed file at %s: %v", dest, err)
	}
}

func TestActionOnFail_Fail_FallbackWhenConfiguredMkdirFails(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "f.txt", "data")

	// Make ArchiveDest a FILE path to force MkdirAll error.
	badDir := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(badDir, []byte("file, not dir"), 0o644); err != nil {
		t.Fatalf("write badDir: %v", err)
	}

	entry := config.ConfigEntry{
		Name:            "fail-fallback",
		SourceDirectory: tmp,
		ActionOnFail:    "archive",
		ArchiveDest:     badDir,
	}

	if err := ActionOnFail(entry, src); err != nil {
		t.Fatalf("fail fallback failed: %v", err)
	}
	dest := filepath.Join(tmp, "fail", "f.txt")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("expected failed file at fallback %s: %v", dest, err)
	}
}

func TestActionOnFail_Archive_RenameFailure(t *testing.T) {
	tmp := t.TempDir()
	src := mustWriteTempFile(t, tmp, "g.txt", "data")

	failDest := filepath.Join(tmp, "fail")
	if err := os.MkdirAll(failDest, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Create a directory named "g.txt" at the destination so rename fails.
	destDir := filepath.Join(failDest, "g.txt")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("mkdir destDir: %v", err)
	}

	entry := config.ConfigEntry{
		Name:            "fail-rename-fail",
		SourceDirectory: tmp,
		ActionOnFail:    "archive",
		FailDest:        failDest,
	}

	err := ActionOnFail(entry, src)
	if err == nil {
		t.Fatalf("expected rename failure, got nil")
	}
	// Source should remain after failure.
	if _, statErr := os.Stat(src); statErr != nil && runtime.GOOS != "windows" {
		t.Fatalf("expected source to remain after failure: %v", statErr)
	}
}
