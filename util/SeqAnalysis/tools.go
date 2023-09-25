package main

import (
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// from https://forum.golangbridge.org/t/easy-way-for-letter-substitution-reverse-complementary-dna-sequence/20101
// from https://go.dev/play/p/IXI6PY7XUXN
var dnaComplement = strings.NewReplacer(
	"A", "T",
	"T", "A",
	"G", "C",
	"C", "G",
	"a", "t",
	"t", "a",
	"g", "c",
	"c", "g",
)

func Complement(s string) string {
	return dnaComplement.Replace(s)
}

// Reverse returns its argument string reversed rune-wise left to right.
// from https://github.com/golang/example/blob/master/stringmain/reverse.go
func Reverse(r []byte) []byte {
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func ReverseComplement(s string) string {
	return Complement(string(Reverse([]byte(s))))
}

// Open is a function that opens a file from the given path using the embed.FS file system.
//
// It takes three parameters:
//   - path: a string that represents the path of the file to be opened.
//   - exPath: a string that represents the extra path to be joined with the file path in case the file is not found in the embed.FS file system.
//   - embedFS: an embed.FS file system that provides access to embedded files.
//
// It returns two values:
//   - file: an io.ReadCloser that represents the opened file.
//   - err: an error indicating any error that occurred during the file opening process.
func Open(path, exPath string, embedFS embed.FS) (file io.ReadCloser, err error) {
	file, err = embedFS.Open(path)
	if err != nil {
		return os.Open(filepath.Join(exPath, path))
	}
	return
}
