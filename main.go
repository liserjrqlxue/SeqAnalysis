package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
)

// os
var (
	ex, _   = os.Executable()
	exPath  = filepath.Dir(ex)
	etcPath = path.Join(exPath, "etc")
)

// flag
var (
	workDir = flag.String(
		"w",
		"",
		"workdir",
	)
	input = flag.String(
		"i",
		"input.txt",
		"input info",
	)
)

// global
var (
	BarCode = "AGTGCT"
)

func main() {
	flag.Parse()
	if *workDir != "" {
		simpleUtil.CheckErr(os.Chdir(*workDir))
	}

	var seqList = textUtil.File2Array(*input)
	for _, s := range seqList {
		strings.TrimSuffix(s, "\r")
		var a = strings.Split(s, "\t")

		var seqInfo = &SeqInfo{
			Name:        a[0],
			IndexSeq:    a[1],
			Seq:         []byte(a[2]),
			BarCode:     BarCode,
			Fastqs:      a[3:],
			Stats:       make(map[string]int),
			HitSeqCount: make(map[string]int),
			ReadsLength: make(map[int]int),
		}
		log.Print("seqInfo.Init")
		seqInfo.Init()
		log.Print("seqInfo.CountError4")
		seqInfo.CountError4()

		seqInfo.WriteStatsSheet()
		seqInfo.Save()

		log.Print("eqInfo.PlotLineACGT")
		seqInfo.PlotLineACGT("ACGT.html")

		log.Print("Done")
	}
}
