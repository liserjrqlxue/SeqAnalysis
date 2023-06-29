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
		"current working directory",
	)
	input = flag.String(
		"i",
		"input.txt",
		"input info",
	)
	outputDir = flag.String(
		"o",
		".",
		"output directory")
)

// global
var (
	BarCode = "AGTGCT"
)

func main() {
	flag.Parse()
	if *workDir != "" {
		log.Printf("changes the current working directory to [%s]", *workDir)
		simpleUtil.CheckErr(os.Chdir(*workDir))
	}
	simpleUtil.CheckErr(os.MkdirAll(filepath.Join(*outputDir, "result"), 0755))

	var Sheets = make(map[string]string)
	var sheetMap, _ = textUtil.File2MapArray(path.Join(etcPath, "sheet.txt"), "\t", nil)
	var sheetList []string
	for _, m := range sheetMap {
		Sheets[m["Name"]] = m["SheetName"]
		sheetList = append(sheetList, m["SheetName"])
	}

	var seqList = textUtil.File2Array(*input)
	for _, s := range seqList {
		strings.TrimSuffix(s, "\r")
		var a = strings.Split(s, "\t")

		var seqInfo = &SeqInfo{
			Name:        a[0],
			IndexSeq:    strings.ToUpper(a[1]),
			Seq:         []byte(strings.ToUpper(a[2])),
			BarCode:     BarCode,
			Fastqs:      a[3:],
			Excel:       filepath.Join(*outputDir, "result", a[0]+".xlsx"),
			Sheets:      Sheets,
			SheetList:   sheetList,
			Stats:       make(map[string]int),
			HitSeqCount: make(map[string]int),
			ReadsLength: make(map[int]int),
		}
		log.Printf("[%s]:[%s]:[%s]:[%+v]\n", seqInfo.Name, seqInfo.IndexSeq, seqInfo.Seq, seqInfo.Fastqs)
		seqInfo.Init()
		seqInfo.CountError4()

		seqInfo.WriteStatsSheet()
		seqInfo.Save()
		seqInfo.PrintStats()
		seqInfo.PlotLineACGT("ACGT.html")
	}
}
