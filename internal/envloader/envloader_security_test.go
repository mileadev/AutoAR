package envloader

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateEnvPermissionsAndReplacement(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("Unix file permissions") }
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := writePrivateEnvFile(path, []byte("TOKEN=first\n")); err != nil { t.Fatal(err) }
	stat, err := os.Stat(path)
	if err != nil { t.Fatal(err) }
	if stat.Mode().Perm() != 0600 { t.Fatalf("expected 0600, got %o", stat.Mode().Perm()) }
	if err := os.Chmod(path, 0644); err != nil { t.Fatal(err) }
	if err := writePrivateEnvFile(path, []byte("TOKEN=second\n")); err != nil { t.Fatal(err) }
	stat, err = os.Stat(path)
	if err != nil { t.Fatal(err) }
	if stat.Mode().Perm() != 0600 { t.Fatalf("replacement must restore 0600, got %o", stat.Mode().Perm()) }
}

func TestPrivateEnvRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("unchanged"), 0600); err != nil { t.Fatal(err) }
	link := filepath.Join(dir, ".env")
	if err := os.Symlink(target,link); err != nil { t.Skipf("symlink unavailable: %v", err) }
	if err := writePrivateEnvFile(link, []byte("TOKEN=secret\n")); err == nil { t.Fatal("symlink destination accepted") }
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "unchanged" { t.Fatalf("symlink target modified: %v",err) }
}

func TestEnvInputRejectsControlCharacters(t *testing.T) {
	// Rejected before UpdateEnv can write a file or set an environment variable.
	for _, test := range []struct{key, value string}{
		{"A-B","value"}, {"VALID","first\nOTHER=evil"}, {"VALID","hello\rworld"},
	} {
		if err := UpdateEnv(test.key, test.value); err == nil {
			t.Fatalf("invalid env input accepted: %q", test.key)
		}
	}
	if strings.Contains(os.Getenv("VALID"), "OTHER=evil") { t.Fatal("injected environment variable") }
}
