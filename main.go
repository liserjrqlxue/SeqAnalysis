package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
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
	var out = osUtil.Create("output.txt")
	defer simpleUtil.DeferClose(out)

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

		log.Print("write output.txt")
		fmtUtil.Fprintf(out, "#######################################  Summary of %s\n", s)
		seqInfo.WriteStats(out)
		seqInfo.WriteDistributionFreq(out)
		fmtUtil.Fprint(out, "\n\n\n")

		seqInfo.WriteExcel()
		seqInfo.Save()

		log.Print("eqInfo.PlotLineACGT")
		seqInfo.PlotLineACGT("ACGT.html")

		log.Print("Done")
	}
}
