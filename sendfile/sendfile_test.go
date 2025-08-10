package sendfile

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justin-molloy/tfagent/config"
)

// --- helpers ---

func mustWriteFile(t *testing.T, dir, name, contents string) string {
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

// Generates a valid RSA private key in PEM so ssh.ParsePrivateKey succeeds.
func mustWriteRSAPrivateKey(t *testing.T, dir, name string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBlk := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	fp := filepath.Join(dir, name)
	f, err := os.Create(fp)
	if err != nil {
		t.Fatalf("create pem: %v", err)
	}
	defer f.Close()
	if err := pem.Encode(f, pemBlk); err != nil {
		t.Fatalf("encode pem: %v", err)
	}
	return fp
}

func minimalTransfer(pkPath string) config.ConfigEntry {
	return config.ConfigEntry{
		Name:       "t",
		Username:   "user",
		Server:     "127.0.0.1", // used only in the dial test
		Port:       "1",         // intentionally unreachable
		RemotePath: "/tmp",
		PrivateKey: pkPath,
	}
}

// --- tests ---

func TestUploadSFTP_ReadKeyError(t *testing.T) {
	tmp := t.TempDir()
	// Do not create the key file → should error on read
	missingKey := filepath.Join(tmp, "nope.pem")
	tf := minimalTransfer(missingKey)
	local := mustWriteFile(t, tmp, "local.txt", "data")

	_, err := UploadSFTP(local, tf)
	if err == nil {
		t.Fatalf("expected error reading missing private key")
	}
	if !strings.Contains(err.Error(), "unable to read private key") {
		t.Fatalf("expected 'unable to read private key', got: %v", err)
	}
}

func TestUploadSFTP_ParseKeyError(t *testing.T) {
	tmp := t.TempDir()
	badKey := mustWriteFile(t, tmp, "bad.pem", "not a real key")
	tf := minimalTransfer(badKey)
	local := mustWriteFile(t, tmp, "local.txt", "data")

	_, err := UploadSFTP(local, tf)
	if err == nil {
		t.Fatalf("expected parse error for invalid key")
	}
	if !strings.Contains(err.Error(), "unable to parse private key") {
		t.Fatalf("expected 'unable to parse private key', got: %v", err)
	}
}

func TestUploadSFTP_DialFailure_RetriesAndReturnsError(t *testing.T) {
	// This exercises the real retry loop (3 attempts, 2s delay between).
	// Expect total wall time ≈ 4s due to two sleeps.
	tmp := t.TempDir()
	keyPath := mustWriteRSAPrivateKey(t, tmp, "id_rsa")
	tf := minimalTransfer(keyPath)
	local := mustWriteFile(t, tmp, "local.txt", "data")

	start := time.Now()
	_, err := UploadSFTP(local, tf)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected dial failure error (unreachable addr)")
	}

	// We can't inspect internal attempt count without stubs,
	// but elapsed time should reflect two retries (~4s).
	if elapsed < 3900*time.Millisecond {
		t.Fatalf("expected ~4s elapsed due to retries; got %v", elapsed)
	}
	// Upper bound sanity (to catch accidental hangs): allow some headroom.
	if elapsed > 8*time.Second {
		t.Fatalf("upload took unexpectedly long: %v", elapsed)
	}
}
