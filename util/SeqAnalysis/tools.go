package main

import (
	"embed"
	"io"
	"log"
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

func SingleRun(s, resultDir string, long bool) {
	var seqInfo = new(SeqInfo)
	defer func() {
		SeqInfoMap[s] = seqInfo
		<-chanList
	}()
	s = strings.TrimSuffix(s, "\r")
	var a = strings.Split(s, "\t")

	seqInfo = &SeqInfo{
		Name:          a[0],
		IndexSeq:      strings.ToUpper(a[1]),
		Seq:           []byte(strings.ToUpper(a[2])),
		Fastqs:        a[3:],
		Excel:         filepath.Join(*outputDir, "result", a[0]+".xlsx"),
		Sheets:        Sheets,
		SheetList:     sheetList,
		Stats:         make(map[string]int),
		HitSeqCount:   make(map[string]int),
		ReadsLength:   make(map[int]int),
		AssemblerMode: long,
	}
	if len(a) > 3 {
		seqInfo.Fastqs = a[3:]
	} else {
		seqInfo.Fastqs = []string{
			filepath.Join("00.CleanData", seqInfo.Name, seqInfo.Name+"_1.clean.fq.gz"),
			filepath.Join("00.CleanData", seqInfo.Name, seqInfo.Name+"_2.clean.fq.gz"),
		}
	}
	log.Printf("[%s]:[%s]:[%s]:[%+v]\n", seqInfo.Name, seqInfo.IndexSeq, seqInfo.Seq, seqInfo.Fastqs)
	seqInfo.Init()
	seqInfo.CountError4(resultDir, *verbose)

	seqInfo.WriteStatsSheet(resultDir)
	seqInfo.Save()
	seqInfo.PrintStats(resultDir)
	seqInfo.PlotLineACGT(filepath.Join(resultDir, seqInfo.Name+"ACGT.html"))

	// free HitSeqCount memory
	seqInfo.HitSeqCount = make(map[string]int)
	seqInfo.HitSeq = []string{}
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
