package core

import (
	"fmt"
	"os"
	"path/filepath"
)

func PathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			Throw(err)
		}
	}
	return true
}

func GetFullPath(path string) string {
	if !PathExists(path) {
		Throw(fmt.Errorf("path does not exist: %s", path))
	}
	newPath, err := filepath.Abs(path)
	if err != nil {
		Throw(err)
	}
	return newPath
}

func PathIsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		Throw(err)
	}
	return info.IsDir()
}

// Split a path into parent dir and base name
func SplitPath(path string) (string, string) {
	return filepath.Dir(path), filepath.Base(path)
}

// Split a path into parent dir, base name, and extension
func SplitFilePath(path string) (string, string, string) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(path)
	return dir, base[:len(base)-len(ext)], ext
}

// Remove .* from a path
func RemoveExtension(fileName string) string {
	extension := filepath.Ext(fileName)
	return fileName[:len(fileName)-len(extension)]
}

func MakeDir(path string) string {
	fileInfo, err := os.Lstat("./")
	if err != nil {
		Throw(err)
	}

	fileMode := fileInfo.Mode()
	unixPerms := fileMode & os.ModePerm

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, unixPerms)
		if err != nil {
			Throw(err)
		}
	}
	return GetFullPath(path)
}
