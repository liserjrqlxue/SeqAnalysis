package main

import (
	"log"
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
// from https://github.com/golang/example/blob/master/stringutil/reverse.go
func Reverse(r []byte) []byte {
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func ReverseComplement(s string) string {
	return Complement(string(Reverse([]byte(s))))
}

func SingleRun(s string, offset int) {
	var seqInfo = new(SeqInfo)
	defer func() {
		SeqInfoMap[s] = seqInfo
		<-chanList
	}()
	strings.TrimSuffix(s, "\r")
	var a = strings.Split(s, "\t")

	seqInfo = &SeqInfo{
		Name:        a[0],
		IndexSeq:    strings.ToUpper(a[1]),
		Seq:         []byte(strings.ToUpper(a[2])),
		Offset:      offset,
		Fastqs:      a[3:],
		Excel:       filepath.Join(*outputDir, "result", a[0]+".xlsx"),
		Sheets:      Sheets,
		SheetList:   sheetList,
		Stats:       make(map[string]int),
		HitSeqCount: make(map[string]int),
		ReadsLength: make(map[int]int),
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
	seqInfo.CountError4(*verbose)

	seqInfo.WriteStatsSheet()
	seqInfo.Save()
	seqInfo.PrintStats()
	seqInfo.PlotLineACGT("ACGT.html")
}
