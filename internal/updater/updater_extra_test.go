package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaBundlePath(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(bundle, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SSL_CERT_FILE", bundle)
	if got := caBundlePath(); got != bundle {
		t.Errorf("caBundlePath with SSL_CERT_FILE = %q, want %q", got, bundle)
	}

	t.Setenv("SSL_CERT_FILE", "")
	t.Setenv("PREFIX", dir)
	// Place cert at PREFIX/etc/tls/cert.pem
	p := filepath.Join(dir, "etc", "tls")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle2 := filepath.Join(p, "cert.pem")
	if err := os.WriteFile(bundle2, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := caBundlePath(); got != bundle2 {
		t.Errorf("caBundlePath with PREFIX = %q, want %q", got, bundle2)
	}
}

func TestLoadCertPool(t *testing.T) {
	dir := t.TempDir()
	// An invalid bundle should error.
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("not a cert"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCertPool(bad); err == nil {
		t.Error("loadCertPool accepted invalid data")
	}
}