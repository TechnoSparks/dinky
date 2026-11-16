package filelist

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeFileInfo is a minimal fs.FileInfo implementation used to control the
// values returned to the sorting and formatting helpers.
type fakeFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     any
}

func (fi fakeFileInfo) Name() string       { return fi.name }
func (fi fakeFileInfo) Size() int64        { return fi.size }
func (fi fakeFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fakeFileInfo) ModTime() time.Time { return fi.modTime }
func (fi fakeFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fakeFileInfo) Sys() any           { return fi.sys }

// fakeDirEntry is a minimal os.DirEntry implementation.
type fakeDirEntry struct {
	name    string
	isDir   bool
	typ     fs.FileMode
	info    fs.FileInfo
	infoErr error
}

func (e fakeDirEntry) Name() string      { return e.name }
func (e fakeDirEntry) IsDir() bool       { return e.isDir }
func (e fakeDirEntry) Type() fs.FileMode { return e.typ }
func (e fakeDirEntry) Info() (fs.FileInfo, error) {
	if e.infoErr != nil {
		return nil, e.infoErr
	}
	return e.info, nil
}

func newFakeFile(name string, size int64, mode fs.FileMode, modTime time.Time, uid, gid uint32) fakeDirEntry {
	return fakeDirEntry{
		name:  name,
		isDir: mode.IsDir(),
		typ:   mode.Type(),
		info: fakeFileInfo{
			name:    name,
			size:    size,
			mode:    mode,
			modTime: modTime,
			sys:     &syscall.Stat_t{Uid: uid, Gid: gid},
		},
	}
}

func readEntries(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	return entries
}

func entryByName(entries []os.DirEntry, name string) os.DirEntry {
	for _, entry := range entries {
		if entry.Name() == name {
			return entry
		}
	}
	return nil
}

func namesOf(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{"zero", 0, "0.0B"},
		{"one byte", 1, "1.0B"},
		{"just under one KiB", 1023, "1023.0B"},
		{"one KiB", 1024, "1.0KiB"},
		{"one and a half KiB", 1536, "1.5KiB"},
		{"just under one MiB", 1024*1024 - 1, "1024.0KiB"},
		{"one MiB", 1024 * 1024, "1.0MiB"},
		{"two and a quarter MiB", 2*1024*1024 + 256*1024, "2.2MiB"},
		{"one GiB", 1024 * 1024 * 1024, "1.0GiB"},
		{"two GiB", 2 * 1024 * 1024 * 1024, "2.0GiB"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatSize(tc.size); got != tc.want {
				t.Errorf("formatSize(%d) = %q, want %q", tc.size, got, tc.want)
			}
		})
	}
}

func TestSortCaseInsensitiveFunc(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"abc", "abd", -1},
		{"abd", "abc", 1},
		{"ABC", "abc", 0},
		{"apple", "Banana", -1},
		{"Banana", "apple", 1},
		{"", "a", -1},
		{"a", "", 1},
	}
	for _, tc := range tests {
		got := sortCaseInsensitiveFunc(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("sortCaseInsensitiveFunc(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSortNameFunc(t *testing.T) {
	dir := fakeDirEntry{name: "zdir", isDir: true, typ: fs.ModeDir}
	file := fakeDirEntry{name: "afile"}
	otherDir := fakeDirEntry{name: "adir", isDir: true, typ: fs.ModeDir}

	tests := []struct {
		name string
		a, b os.DirEntry
		want int
	}{
		{"directories sort first", dir, file, -1},
		{"files sort after directories", file, dir, 1},
		{"directories compare by name", dir, otherDir, 1},
		{"case insensitive comparison", fakeDirEntry{name: "Alpha"}, fakeDirEntry{name: "bravo"}, -1},
		{"equal ignoring case", fakeDirEntry{name: "Alpha"}, fakeDirEntry{name: "alpha"}, 0},
		{"identical", file, fakeDirEntry{name: "afile"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sortNameFunc(tc.a, tc.b); got != tc.want {
				t.Errorf("sortNameFunc(%q, %q) = %d, want %d", tc.a.Name(), tc.b.Name(), got, tc.want)
			}
		})
	}
}

func TestSortSizeFunc(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		a, b fakeDirEntry
		want int
	}{
		{"smaller first", newFakeFile("a", 10, 0, now, 0, 0), newFakeFile("b", 20, 0, now, 0, 0), -1},
		{"larger second", newFakeFile("a", 20, 0, now, 0, 0), newFakeFile("b", 10, 0, now, 0, 0), 1},
		{"equal", newFakeFile("a", 10, 0, now, 0, 0), newFakeFile("b", 10, 0, now, 0, 0), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sortSizeFunc(tc.a, tc.b); got != tc.want {
				t.Errorf("sortSizeFunc() = %d, want %d", got, tc.want)
			}
		})
	}

	errEntry := fakeDirEntry{name: "bad", infoErr: os.ErrPermission}
	if got := sortSizeFunc(errEntry, newFakeFile("b", 0, 0, now, 0, 0)); got != 0 {
		t.Errorf("sortSizeFunc with info error = %d, want 0", got)
	}
}

func TestSortModifiedFunc(t *testing.T) {
	base := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		a, b fakeDirEntry
		want int
	}{
		{"older first", newFakeFile("a", 0, 0, base, 0, 0), newFakeFile("b", 0, 0, base.Add(time.Hour), 0, 0), -1},
		{"newer last", newFakeFile("a", 0, 0, base.Add(time.Hour), 0, 0), newFakeFile("b", 0, 0, base, 0, 0), 1},
		{"equal", newFakeFile("a", 0, 0, base, 0, 0), newFakeFile("b", 0, 0, base, 0, 0), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sortModifiedFunc(tc.a, tc.b); got != tc.want {
				t.Errorf("sortModifiedFunc() = %d, want %d", got, tc.want)
			}
		})
	}

	errEntry := fakeDirEntry{name: "bad", infoErr: os.ErrPermission}
	if got := sortModifiedFunc(errEntry, newFakeFile("b", 0, 0, base, 0, 0)); got != -1 {
		t.Errorf("sortModifiedFunc with info error = %d, want -1", got)
	}
}

func TestSortPermissionsFunc(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		a, b fakeDirEntry
		want int
	}{
		{"read only first", newFakeFile("a", 0, 0o400, now, 0, 0), newFakeFile("b", 0, 0o644, now, 0, 0), -1},
		{"writable last", newFakeFile("a", 0, 0o755, now, 0, 0), newFakeFile("b", 0, 0o644, now, 0, 0), 1},
		{"equal", newFakeFile("a", 0, 0o644, now, 0, 0), newFakeFile("b", 0, 0o644, now, 0, 0), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sortPermissionsFunc(tc.a, tc.b); got != tc.want {
				t.Errorf("sortPermissionsFunc() = %d, want %d", got, tc.want)
			}
		})
	}

	errEntry := fakeDirEntry{name: "bad", infoErr: os.ErrPermission}
	if got := sortPermissionsFunc(errEntry, newFakeFile("b", 0, 0o644, now, 0, 0)); got != -1 {
		t.Errorf("sortPermissionsFunc with info error = %d, want -1", got)
	}
}

func TestSortOwnerAndGroupFuncs(t *testing.T) {
	now := time.Now()
	a := newFakeFile("a", 0, 0o644, now, 4000000000, 4000000000)
	b := newFakeFile("b", 0, 0o644, now, 4000000001, 4000000001)

	if got := sortOwnerFunc(a, b); got != sortCaseInsensitiveFunc(ownerName(a), ownerName(b)) {
		t.Errorf("sortOwnerFunc() = %d, want comparison of owner names", got)
	}
	if got := sortGroupFunc(a, b); got != sortCaseInsensitiveFunc(groupName(a), groupName(b)) {
		t.Errorf("sortGroupFunc() = %d, want comparison of group names", got)
	}
	if got := sortOwnerFunc(a, a); got != 0 {
		t.Errorf("sortOwnerFunc(a, a) = %d, want 0", got)
	}
	if got := sortGroupFunc(a, a); got != 0 {
		t.Errorf("sortGroupFunc(a, a) = %d, want 0", got)
	}
}

func TestOwnerAndGroupName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entry := entryByName(readEntries(t, dir), "file.txt")
	if entry == nil {
		t.Fatal("file entry not found")
	}

	owner := ownerName(entry)
	if owner == "" || owner == "?" {
		t.Errorf("ownerName() = %q, want a resolved owner", owner)
	}
	group := groupName(entry)
	if group == "" {
		t.Errorf("groupName() = %q, want a non-empty group", group)
	}

	unknownUID := newFakeFile("unknown", 0, 0o644, time.Now(), 4000000000, 4000000000)
	if got := ownerName(unknownUID); got != "4000000000" {
		t.Errorf("ownerName() for unknown uid = %q, want numeric uid", got)
	}
	if got := groupName(unknownUID); got != "?" {
		t.Errorf("groupName() for unknown gid = %q, want %q", got, "?")
	}

	errEntry := fakeDirEntry{name: "bad", infoErr: os.ErrPermission}
	if got := ownerName(errEntry); got != "?" {
		t.Errorf("ownerName() with info error = %q, want %q", got, "?")
	}
	if got := groupName(errEntry); got != "?" {
		t.Errorf("groupName() with info error = %q, want %q", got, "?")
	}
}

func TestPermissions(t *testing.T) {
	dir := t.TempDir()

	file600 := filepath.Join(dir, "file600")
	if err := os.WriteFile(file600, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(file600, 0o600); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	file644 := filepath.Join(dir, "file644")
	if err := os.WriteFile(file644, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(file644, 0o644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o750); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.Chmod(subdir, 0o750); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file600, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	entries := readEntries(t, dir)
	tests := []struct {
		name string
		want string
	}{
		{"file600", "-rw-------"},
		{"file644", "-rw-r--r--"},
		{"subdir", "drwxr-x---"},
		{"link", "Lrwxrwxrwx"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := entryByName(entries, tc.name)
			if entry == nil {
				t.Fatalf("entry %q not found", tc.name)
			}
			if got := permissions(entry); got != tc.want {
				t.Errorf("permissions() = %q, want %q", got, tc.want)
			}
		})
	}

	errEntry := fakeDirEntry{name: "bad", infoErr: os.ErrPermission}
	if got := permissions(errEntry); got != "?" {
		t.Errorf("permissions() with info error = %q, want %q", got, "?")
	}
}

func TestEmojiForFileType(t *testing.T) {
	tests := []struct {
		name  string
		entry os.DirEntry
		want  string
	}{
		{"directory", fakeDirEntry{name: "dir", isDir: true, typ: fs.ModeDir}, "📁"},
		{"symlink", fakeDirEntry{name: "link", typ: fs.ModeSymlink}, "🔗"},
		{"executable", fakeDirEntry{name: "runme", typ: 0o755}, "⚡"},
		{"go source", fakeDirEntry{name: "main.go"}, "🐹"},
		{"markdown", fakeDirEntry{name: "notes.md"}, "📜"},
		{"text", fakeDirEntry{name: "notes.txt"}, "📝"},
		{"windows executable", fakeDirEntry{name: "tool.exe"}, "⚡"},
		{"unknown extension", fakeDirEntry{name: "data.unknown"}, "📄"},
		{"no extension", fakeDirEntry{name: "LICENSE"}, "📄"},
		{"extension is case sensitive", fakeDirEntry{name: "main.GO"}, "📄"},
		{"directory wins over extension", fakeDirEntry{name: "folder.go", isDir: true, typ: fs.ModeDir}, "📁"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := emojiForFileType(tc.entry); got != tc.want {
				t.Errorf("emojiForFileType(%q) = %q, want %q", tc.entry.Name(), got, tc.want)
			}
		})
	}
}

func TestAliasDirEntry(t *testing.T) {
	base := newFakeFile("original", 123, 0o644, time.Now(), 0, 0)
	aliased := aliasDirEntry(base, "..")

	if got := aliased.Name(); got != ".." {
		t.Errorf("Name() = %q, want %q", got, "..")
	}
	if got := aliased.IsDir(); got != base.IsDir() {
		t.Errorf("IsDir() = %v, want %v", got, base.IsDir())
	}
	if got := aliased.Type(); got != base.Type() {
		t.Errorf("Type() = %v, want %v", got, base.Type())
	}
	info, err := aliased.Info()
	if err != nil {
		t.Fatalf("Info(): %v", err)
	}
	baseInfo, err := base.Info()
	if err != nil {
		t.Fatalf("base Info(): %v", err)
	}
	if info.Size() != baseInfo.Size() {
		t.Errorf("Info().Size() = %d, want %d", info.Size(), baseInfo.Size())
	}
}

func TestParentDirEntry(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	entry, err := parentDirEntry(child)
	if err != nil {
		t.Fatalf("parentDirEntry(%q): %v", child, err)
	}
	if entry == nil {
		t.Fatal("parentDirEntry returned nil entry")
	}
	if entry.Name() != "child" || !entry.IsDir() {
		t.Errorf("parentDirEntry = %q (dir=%v), want child directory", entry.Name(), entry.IsDir())
	}

	entry, err = parentDirEntry(root)
	if err != nil {
		t.Fatalf("parentDirEntry(%q): %v", root, err)
	}
	if entry == nil || entry.Name() != filepath.Base(root) {
		t.Errorf("parentDirEntry(root) = %v, want name %q", entry, filepath.Base(root))
	}

	entry, err = parentDirEntry(filepath.Join(root, "missing"))
	if err != nil {
		t.Fatalf("parentDirEntry(missing): %v", err)
	}
	if entry != nil {
		t.Errorf("parentDirEntry(missing) = %v, want nil", entry)
	}

	if _, err = parentDirEntry(filepath.Join(root, "no-such-dir", "child")); err == nil {
		t.Error("parentDirEntry with nonexistent parent returned nil error")
	}
}

// TestSortNameOrdersDirectoriesFirst is an integration check using real entries
// read from a temporary directory.
func TestSortNameOrdersDirectoriesFirst(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"afile", "zdir", "Bfile"} {
		if strings.HasSuffix(name, "dir") {
			if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
				t.Fatalf("Mkdir: %v", err)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	entries := readEntries(t, dir)
	sortEntries := append([]os.DirEntry(nil), entries...)
	// Use a local sort to avoid depending on filelist internals here.
	for i := 0; i < len(sortEntries); i++ {
		for j := i + 1; j < len(sortEntries); j++ {
			if sortNameFunc(sortEntries[j], sortEntries[i]) < 0 {
				sortEntries[i], sortEntries[j] = sortEntries[j], sortEntries[i]
			}
		}
	}
	got := namesOf(sortEntries)
	want := []string{"zdir", "afile", "Bfile"}
	if !equalStrings(got, want) {
		t.Errorf("sorted names = %v, want %v", got, want)
	}
}
