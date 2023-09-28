package main

import (
	"bufio"
	"embed"
	"flag"
	"os"
	"path"
	"path/filepath"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/scannerUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// os
var (
	ex, _   = os.Executable()
	exPath  = filepath.Dir(ex)
	binPath = path.Join(exPath, "bin")
	etcPath = path.Join(exPath, "etc")
)

// flag
var (
	fqDir = flag.String(
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
		"",
		"output directory, default is sub directory of CWD: [BaseName]+.分析",
	)
	verbose = flag.Int(
		"v",
		0,
		"verbose level\n\t1: more log\n\t2: unmatched.txt",
	)
	thread = flag.Int(
		"t",
		0,
		"thread used, default len(input)",
	)
	zip = flag.Bool(
		"zip",
		false,
		"Compress-Archive to zip, windows only",
	)
	long = flag.Bool(
		"long",
		false,
		"if too long without polyA",
	)
	rev = flag.Bool(
		"rev",
		false,
		"reverse synthesis",
	)
)

// embed etc
//
//go:embed etc/*.txt
var etcEMFS embed.FS

func init() {
	var sheetTxt, err = Open("etc/sheet.txt", exPath, etcEMFS)
	simpleUtil.CheckErr(err)
	var sheetScan = bufio.NewScanner(sheetTxt)
	var sheetMap, _ = scannerUtil.Scanner2MapArray(sheetScan, "\t", nil)
	simpleUtil.CheckErr(sheetTxt.Close())

	for _, m := range sheetMap {
		Sheets[m["Name"]] = m["SheetName"]
		sheetList = append(sheetList, m["SheetName"])
	}
}

func main() {
	flag.Parse()

	// parse input
	var inputInfo = ParseInput(*input, *fqDir)

	// parallel options
	// runtime.GOMAXPROCS(runtime.NumCPU()) * 2)
	if *thread == 0 {
		*thread = len(inputInfo)
	}
	chanList = make(chan bool, *thread)

	if *outputDir == "" {
		*outputDir = filepath.Base(simpleUtil.HandleError(os.Getwd()).(string)) + ".分析"
	}
	// pare output directory structure
	var resultDir = filepath.Join(*outputDir, "result")
	simpleUtil.CheckErr(os.MkdirAll(filepath.Join(*outputDir, "result"), 0755))

	// write info.txt
	var info = osUtil.Create(filepath.Join(resultDir, "info.txt"))
	var infoTitle = []string{"id", "index", "seq", "fq"}
	// write title
	fmtUtil.FprintStringArray(info, infoTitle, "\t")

	// loop inputInfo for info.txt
	for i := range inputInfo {
		var data = inputInfo[i]
		fmtUtil.Fprintf(
			info,
			"%s\t%s\t%s\t%s\n",
			data["id"],
			data["index"],
			data["seq"],
			data["fq"],
		)
		var seqInfo = NewSeqInfo(data, *long, *rev)
		SeqInfoMap[seqInfo.Name] = seqInfo
	}
	simpleUtil.CheckErr(info.Close())

	for id := range SeqInfoMap {
		chanList <- true
		go func(id string) {
			defer func() {
				<-chanList
			}()
			SeqInfoMap[id].SingleRun(resultDir)
		}(id)
	}

	// wait goconcurrency thread to finish
	for i := 0; i < *thread; i++ {
		chanList <- true
	}

	// write summary.txt
	summaryTxt(resultDir, inputInfo)

	// write summary.xlsx
	if isXlsx.MatchString(*input) {
		// update from input.xlsx
		input2summaryXlsx(*input, resultDir)
	} else {
		summaryXlsx(resultDir, inputInfo)
	}

	// use Rscript to plot
	simpleUtil.CheckErr(sge.Run("Rscript", filepath.Join(binPath, "plot.R"), *outputDir))

	// Compress-Archive to zip file on windows only when *zip is true
	Zip(resultDir)
}
