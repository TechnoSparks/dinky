package filelist

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

func ownerName(entry os.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return "?"
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid := stat.Uid
		user, err := user.LookupId(fmt.Sprintf("%d", uid))
		if err == nil {
			return user.Username
		}
		return fmt.Sprintf("%d", uid)
	}
	return "?"
}

func groupName(entry os.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return "?"
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if group, err := user.LookupGroupId(fmt.Sprintf("%d", stat.Gid)); err == nil {
			return group.Name
		}
	}
	return "?"
}

func permissions(entry os.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return "?"
	}
	return info.Mode().String()
}

func sortCaseInsensitiveFunc(a string, b string) int {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)
	if aLower < bLower {
		return -1
	} else if aLower > bLower {
		return 1
	}
	return 0
}

func sortNameFunc(a os.DirEntry, b os.DirEntry) int {
	aLower := strings.ToLower(a.Name())
	bLower := strings.ToLower(b.Name())

	// Cheesy hack to ensure directories are sorted before files
	if a.IsDir() {
		aLower = string(rune(0)) + aLower
	}
	if b.IsDir() {
		bLower = string(rune(0)) + bLower
	}

	if aLower < bLower {
		return -1
	} else if aLower > bLower {
		return 1
	}
	return 0
}

func sortSizeFunc(a os.DirEntry, b os.DirEntry) int {
	// Directory "size" is filesystem-dependent (and not meaningful), so
	// treat directories as size 0 for sorting to keep ordering deterministic
	// across platforms.
	var aSize int64 = 0
	if !a.IsDir() {
		if aInfo, err := a.Info(); err == nil {
			aSize = aInfo.Size()
		}
	}

	var bSize int64 = 0
	if !b.IsDir() {
		if bInfo, err := b.Info(); err == nil {
			bSize = bInfo.Size()
		}
	}

	if aSize < bSize {
		return -1
	} else if aSize > bSize {
		return 1
	}
	return 0
}

func sortModifiedFunc(a os.DirEntry, b os.DirEntry) int {
	var aModTime, bModTime int64 = 0, 0
	if aInfo, err := a.Info(); err == nil {
		aModTime = aInfo.ModTime().Unix()
	}
	if bInfo, err := b.Info(); err == nil {
		bModTime = bInfo.ModTime().Unix()
	}
	if aModTime < bModTime {
		return -1
	} else if aModTime > bModTime {
		return 1
	}
	return 0
}

func sortPermissionsFunc(a os.DirEntry, b os.DirEntry) int {
	var aPerm, bPerm string
	if aInfo, err := a.Info(); err == nil {
		aPerm = aInfo.Mode().String()
	}
	if bInfo, err := b.Info(); err == nil {
		bPerm = bInfo.Mode().String()
	}
	if aPerm < bPerm {
		return -1
	} else if aPerm > bPerm {
		return 1
	}
	return 0
}

func sortOwnerFunc(a os.DirEntry, b os.DirEntry) int {
	return sortCaseInsensitiveFunc(ownerName(a), ownerName(b))
}

func sortGroupFunc(a os.DirEntry, b os.DirEntry) int {
	return sortCaseInsensitiveFunc(groupName(a), groupName(b))
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%.1fB", float64(size))
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fKiB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMiB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1fGiB", float64(size)/(1024*1024*1024))
	}
}

func emojiForFileType(entry os.DirEntry) string {
	if entry.IsDir() {
		return "📁"
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return "🔗"
	}
	// Check if the file is executable
	if entry.Type()&os.ModePerm != 0 && (entry.Type()&os.ModeType) == 0 {
		return "⚡"
	}

	ext := filepath.Ext(entry.Name())
	fileExtensionEmojis, ok := fileExtensionEmojis[ext]
	if ok {
		return fileExtensionEmojis
	}
	return "📄" // Default emoji for files
}

var fileExtensionEmojis = map[string]string{
	".txt": "\U0001F4DD", //	📝
	".md":  "\U0001F4DC", //	📜
	".log": "\U0001F4DD", //	📝
	".csv": "\U0001F4CA", //	📊
	".tsv": "\U0001F4CA", //	📊
	// ".json":   "\u2699\ufe0f",     //	⚙️
	// ".yaml":   "\u2699\ufe0f",     //	⚙️
	// ".yml":    "\u2699\ufe0f",     //	⚙️
	// ".xml":    "\u2699\ufe0f",     //	⚙️
	// ".ini":    "\u2699\ufe0f",     //	⚙️
	// ".conf":   "\u2699\ufe0f",     //	⚙️
	// ".cfg":    "\u2699\ufe0f",     //	⚙️
	".py":   "\U0001F40D", //	🐍
	".js":   "\U0001F9E0", //	🧠
	".ts":   "\U0001F9E0", //	🧠
	".java": "\u2615",     //	☕
	".c":    "\U0001F4BB", //	💻
	".cpp":  "\U0001F4BB", //	💻
	".h":    "\U0001F4BB", //	💻
	".hpp":  "\U0001F4BB", //	💻
	".rb":   "\U0001F48E", //	💎
	".sh":   "\U0001F4BB", //	💻
	".bat":  "\U0001F4BB", //	💻
	".ps1":  "\U0001F4BB", //	💻
	".html": "\U0001F310", //	🌐
	".htm":  "\U0001F310", //	🌐
	".css":  "\U0001F3A8", //	🎨
	".scss": "\U0001F3A8", //	🎨
	".less": "\U0001F3A8", //	🎨
	".go":   "\U0001F439", //	🐹
	".rs":   "\U0001F980", //	🦀
	".php":  "\U0001F418", //	🐘
	// ".swift":  "\U0001F54A\ufe0f", // 🕊️
	".pl": "\U0001F9EC", //	🧬
	".r":  "\U0001F4C8", //	📈
	// ".sql":    "\U0001F5C3\ufe0f", //	🗃️
	// ".db":     "\U0001F5C3\ufe0f", //	🗃️
	// ".sqlite": "\U0001F5C3\ufe0f", // 🗃️
	// ".zip":    "\U0001F5C3\ufe0f", //	🗃️
	// ".tar":    "\U0001F5C3\ufe0f", //	🗃️
	// ".gz":     "\U0001F5C3\ufe0f", //	🗃️
	// ".rar":    "\U0001F5C3\ufe0f", //	🗃️
	// ".7z":     "\U0001F5C3\ufe0f", //	🗃️
	".jar": "\U0001F4E6", //	📦
	".war": "\U0001F4E6", //	📦
	".dll": "\U0001F4E6", //	📦
	".so":  "\U0001F4E6", //	📦
	".exe": "\u26A1",     //	⚡
	".app": "\u26A1",     //	⚡
	".apk": "\U0001F4E6", //	📦
	// ".jpg":    "\U0001F5BC\ufe0f", //	🖼️
	// ".jpeg":   "\U0001F5BC\ufe0f", //	🖼️
	// ".png":    "\U0001F5BC\ufe0f", //	🖼️
	// ".gif":    "\U0001F5BC\ufe0f", //	🖼️
	// ".bmp":    "\U0001F5BC\ufe0f", //	🖼️
	// ".svg":    "\U0001F5BC\ufe0f", //	🖼️
	// ".webp":   "\U0001F5BC\ufe0f", //	🖼️
	".mp3":  "\U0001F3B5", //	🎵
	".wav":  "\U0001F3B5", //	🎵
	".flac": "\U0001F3B5", //	🎵
	// ".mp4":    "\U0001F39E\ufe0f", //	🎞️
	// ".mkv":    "\U0001F39E\ufe0f", //	🎞️
	// ".avi":    "\U0001F39E\ufe0f", //	🎞️
	// ".mov":    "\U0001F39E\ufe0f", //	🎞️
	".pdf":  "\U0001F4C4", //	📄
	".doc":  "\U0001F4C4", //	📄
	".docx": "\U0001F4C4", //	📄
	".ppt":  "\U0001F4CA", //	📊
	".pptx": "\U0001F4CA", //	📊
	".xls":  "\U0001F4CA", //	📊
	".xlsx": "\U0001F4CA", //	📊
	".key":  "\U0001F511", //	🔐
	".pem":  "\U0001F511", //	🔐
	".crt":  "\U0001F511", //	🔐
	".cer":  "\U0001F511", //	🔐
	".bak":  "\U0001F9F9", //	🧹
	".tmp":  "\U0001F9F9", //	🧹
	".swp":  "\U0001F9F9", //	🧹
}

type aliasDirEntryImpl struct {
	entry   os.DirEntry
	newName string
}

func aliasDirEntry(entry os.DirEntry, newName string) os.DirEntry {
	return &aliasDirEntryImpl{
		entry:   entry,
		newName: newName,
	}
}

func (a aliasDirEntryImpl) Name() string {
	return a.newName
}

func (a aliasDirEntryImpl) IsDir() bool {
	return a.entry.IsDir()
}

func (a aliasDirEntryImpl) Type() fs.FileMode {
	return a.entry.Type()
}

func (a aliasDirEntryImpl) Info() (fs.FileInfo, error) {
	return a.entry.Info()
}

func parentDirEntry(dirPath string) (os.DirEntry, error) {
	parentPath := filepath.Clean(filepath.Join(dirPath, ".."))
	baseName := filepath.Base(dirPath)
	parentEntries, err := os.ReadDir(parentPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range parentEntries {
		if entry.Name() == baseName {
			return entry, nil
		}
	}
	return nil, nil
}
