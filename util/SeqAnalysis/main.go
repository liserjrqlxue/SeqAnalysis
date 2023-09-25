package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/scannerUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
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
		"output directory",
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
	if *workDir != "" {
		log.Printf("changes the current working directory to [%s]", *workDir)
		simpleUtil.CheckErr(os.Chdir(*workDir))
	}

	// parse input
	var inputInfo = ParseInput(*input)

	// parallel options
	// runtime.GOMAXPROCS(runtime.NumCPU()) * 2)
	if *thread == 0 {
		*thread = len(inputInfo)
	}
	chanList = make(chan bool, *thread)

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

	var summary = osUtil.Create(filepath.Join(resultDir, "summary.txt"))

	//fmt.Println(strings.Join(textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt")), "\t"))
	var summaryTitle = textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt"))
	fmtUtil.FprintStringArray(summary, summaryTitle, "\t")

	// write summary.xlsx
	var summaryXlsx = excelize.NewFile()
	simpleUtil.CheckErr(summaryXlsx.SetSheetName("Sheet1", "Summary"))
	for i, s := range summaryTitle {
		SetCellStr(summaryXlsx, "Summary", 1+i, 1, s)
	}
	for i := range inputInfo {
		var (
			info = SeqInfoMap[inputInfo[i]["id"]]
			rows = info.SummaryRow()
		)
		SetRow(summaryXlsx, "Summary", 1, 2+i, rows)

		info.WriteStatsTxt(summary)
	}
	simpleUtil.CheckErr(summaryXlsx.SaveAs(filepath.Join(resultDir, fmt.Sprintf("summary-%s.xlsx", time.Now().Format("20060102")))))

	// close file handle before Compress-Archive
	simpleUtil.CheckErr(summary.Close())

	simpleUtil.CheckErr(sge.Run("Rscript", filepath.Join(binPath, "plot.R"), *outputDir))

	// Compress-Archive to zip file on windows only when *zip is true
	if runtime.GOOS == "windows" {
		var resultZip = *outputDir + ".result.zip"
		if *outputDir == "." {
			resultZip = "result.zip"
		}
		var args = []string{
			"Compress-Archive",
			"-Path",
			fmt.Sprintf("\"%s/*.xlsx\",\"%s/*.pdf\"", resultDir, resultDir),
			"-DestinationPath",
			resultZip,
			"-Force",
		}
		log.Println(strings.Join(args, " "))
		if *zip {
			simpleUtil.CheckErr(sge.Run("powershell", args...))
			simpleUtil.CheckErr(sge.Run("powershell", "explorer", resultDir))
		}
	}
}
