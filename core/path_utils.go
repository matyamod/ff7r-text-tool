package core

import (
	"bytes"
	"fmt"
	"io"
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

func OpenFile(path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		Throw(err)
	}
	return file
}

func CreateFile(path string) *os.File {
	file, err := os.Create(path)
	if err != nil {
		Throw(err)
	}
	return file
}

func FilesAreEqual(file1Path, file2Path string) (bool, error) {
	fmt.Printf("Comparing %s and %s...\n", file1Path, file2Path)
	// Open the first file
	file1, err := os.Open(file1Path)
	if err != nil {
		return false, err
	}
	defer file1.Close()

	// Open the second file
	file2, err := os.Open(file2Path)
	if err != nil {
		return false, err
	}
	defer file2.Close()

	// Get file sizes
	file1Info, err := file1.Stat()
	if err != nil {
		return false, err
	}
	file2Info, err := file2.Stat()
	if err != nil {
		return false, err
	}

	// Compare file sizes
	if file1Info.Size() != file2Info.Size() {
		return false, nil
	}

	// Compare file contents
	buf1 := make([]byte, 4096)
	buf2 := make([]byte, 4096)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}

		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("error reading files: %v, %v", err1, err2)
		}
	}

	return true, nil
}
