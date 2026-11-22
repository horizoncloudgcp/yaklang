package thirdparty_bin

import (
	"archive/tar"
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// TestExtractFile_RejectsPathTraversal 确认带有 "../" 的条目不会被写到目标目录之外 (zip-slip)。
func TestExtractFile_RejectsPathTraversal(t *testing.T) {
	const payload = "escaped"

	writeZip := func(t *testing.T, archivePath string, names ...string) {
		f, err := os.Create(archivePath)
		if err != nil {
			t.Fatalf("create zip: %v", err)
		}
		defer f.Close()
		zw := zip.NewWriter(f)
		for _, name := range names {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatalf("create zip entry %q: %v", name, err)
			}
			if _, err := w.Write([]byte(payload)); err != nil {
				t.Fatalf("write zip entry %q: %v", name, err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("close zip: %v", err)
		}
	}

	writeTar := func(t *testing.T, archivePath string, names ...string) {
		f, err := os.Create(archivePath)
		if err != nil {
			t.Fatalf("create tar: %v", err)
		}
		defer f.Close()
		tw := tar.NewWriter(f)
		for _, name := range names {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(payload))}); err != nil {
				t.Fatalf("write tar header %q: %v", name, err)
			}
			if _, err := tw.Write([]byte(payload)); err != nil {
				t.Fatalf("write tar entry %q: %v", name, err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("close tar: %v", err)
		}
	}

	tests := []struct {
		name    string
		archive string
		write   func(t *testing.T, archivePath string, names ...string)
		entry   string
		pick    string
	}{
		{name: "zip all", archive: "evil.zip", write: writeZip, entry: "../escape.txt", pick: "*"},
		{name: "zip directory", archive: "evil.zip", write: writeZip, entry: "build/../../escape.txt", pick: "build/"},
		{name: "zip directory contents", archive: "evil.zip", write: writeZip, entry: "build/../../escape.txt", pick: "build/*"},
		{name: "tar all", archive: "evil.tar", write: writeTar, entry: "../escape.txt", pick: "*"},
		{name: "tar directory", archive: "evil.tar", write: writeTar, entry: "build/../../escape.txt", pick: "build/"},
		{name: "tar directory contents", archive: "evil.tar", write: writeTar, entry: "build/../../escape.txt", pick: "build/*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			archivePath := filepath.Join(root, tt.archive)
			// a harmless entry first: extraction must not stop before reaching the hostile one
			tt.write(t, archivePath, "build/ok.txt", tt.entry)

			target := filepath.Join(root, "out", "target")
			if err := ExtractFile(archivePath, target, "", tt.pick, true); err == nil {
				t.Fatalf("expected extraction of %q to be rejected", tt.entry)
			}

			// "../escape.txt" relative to target, and "build/../../escape.txt" likewise, both land here
			escaped := filepath.Join(root, "out", "escape.txt")
			if _, err := os.Stat(escaped); err == nil {
				t.Fatalf("entry %q was written outside the target directory: %s", tt.entry, escaped)
			}
		})
	}
}

func TestSafeArchiveJoin(t *testing.T) {
	target := filepath.Join("some", "target")
	accepted := map[string]string{
		"":                 target,
		"tool":             filepath.Join(target, "tool"),
		"./bin/tool":       filepath.Join(target, "bin", "tool"),
		"build/lib/x.dll":  filepath.Join(target, "build", "lib", "x.dll"),
		"build/sub/../x":   filepath.Join(target, "build", "x"),
		"name..with..dots": filepath.Join(target, "name..with..dots"),
	}
	for entry, want := range accepted {
		got, err := safeArchiveJoin(target, entry)
		if err != nil {
			t.Errorf("safeArchiveJoin(%q) unexpectedly rejected: %v", entry, err)
			continue
		}
		if got != want {
			t.Errorf("safeArchiveJoin(%q) = %q, want %q", entry, got, want)
		}
	}

	for _, entry := range []string{"..", "../x", "a/../../x", "/etc/passwd"} {
		if got, err := safeArchiveJoin(target, entry); err == nil {
			t.Errorf("safeArchiveJoin(%q) = %q, expected rejection", entry, got)
		}
	}
}

func TestArchiveEntryBaseName(t *testing.T) {
	for entry, want := range map[string]string{
		"tool":           "tool",
		"bin/tool":       "tool",
		"./bin/tool.exe": "tool.exe",
		"a/../tool":      "tool",
	} {
		got, err := archiveEntryBaseName(entry)
		if err != nil || got != want {
			t.Errorf("archiveEntryBaseName(%q) = %q, %v; want %q", entry, got, err, want)
		}
	}
	// names whose last element is not a file name must be rejected, not joined onto the target
	for _, entry := range []string{"..", "bin/..", "/"} {
		if got, err := archiveEntryBaseName(entry); err == nil {
			t.Errorf("archiveEntryBaseName(%q) = %q, expected rejection", entry, got)
		}
	}
}
