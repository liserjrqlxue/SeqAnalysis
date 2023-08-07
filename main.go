package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
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
		"output directory",
	)
	offset = flag.Int(
		"c",
		0,
		"offset between index and target seq",
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
	rc = flag.Bool(
		"rc",
		false,
		"if analysis reverse complement",
	)
)

func init() {
	var sheetMap, _ = textUtil.File2MapArray(path.Join(etcPath, "sheet.txt"), "\t", nil)
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
	simpleUtil.CheckErr(os.MkdirAll(filepath.Join(*outputDir, "result"), 0755))

	// runtime.GOMAXPROCS(runtime.NumCPU()) * 2)

	var seqList = textUtil.File2Array(*input)
	if *thread == 0 {
		*thread = len(seqList)
	}
	chanList = make(chan bool, *thread)
	for _, s := range seqList {
		chanList <- true
		go SingleRun(s, *offset)
	}
	for i := 0; i < *thread; i++ {
		chanList <- true
	}

	var summary = osUtil.Create(filepath.Join("result", "summary.txt"))

	//fmt.Println(strings.Join(textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt")), "\t"))
	var summaryTitle = textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt"))
	fmtUtil.FprintStringArray(summary, summaryTitle, "\t")

	var summaryXlsx = excelize.NewFile()
	simpleUtil.CheckErr(summaryXlsx.SetSheetName("Sheet1", "Summary"))
	for i, s := range summaryTitle {
		SetCellStr(summaryXlsx, "Summary", 1+i, 1, s)
	}

	for i, s := range seqList {
		var (
			info  = SeqInfoMap[s]
			stats = info.Stats
		)
		var rows = []interface{}{
			info.Name, info.IndexSeq, info.Seq, len(info.Seq),
			stats["AllReadsNum"], stats["IndexReadsNum"], stats["AnalyzedReadsNum"], stats["RightReadsNum"],
			info.YieldCoefficient, info.AverageYieldAccuracy,
			math.DivisionInt(stats["ErrorReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDelReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel1ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel2ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDelDupReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel3ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorInsReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorInsDelReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorMutReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorOtherReadsNum"], stats["AnalyzedReadsNum"]),
		}
		SetRow(summaryXlsx, "Summary", 1, 2+i, rows)

		fmtUtil.Fprintf(
			summary,
			"%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n",
			info.Name, info.IndexSeq, info.Seq, len(info.Seq),
			stats["AllReadsNum"], stats["IndexReadsNum"], stats["AnalyzedReadsNum"], stats["RightReadsNum"],
			info.YieldCoefficient, info.AverageYieldAccuracy,
			math.DivisionInt(stats["ErrorReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDelReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel1ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel2ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDelDupReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorDel3ReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorInsReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorInsDelReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorMutReadsNum"], stats["AnalyzedReadsNum"]),
			math.DivisionInt(stats["ErrorOtherReadsNum"], stats["AnalyzedReadsNum"]),
		)
	}
	simpleUtil.CheckErr(summaryXlsx.SaveAs(filepath.Join("result", "summary.xlsx")))

	// close file handle before Compress-Archive
	simpleUtil.DeferClose(summary)

	if runtime.GOOS == "windows" {
		var cwd = filepath.Base(simpleUtil.HandleError(os.Getwd()).(string))
		var args = []string{
			"Compress-Archive",
			"-Path",
			"result",
			"-DestinationPath",
			cwd + ".result.zip",
			"-Force",
		}
		log.Println(strings.Join(args, " "))
		if *zip {
			simpleUtil.CheckErr(sge.Run("powershell", args...))
			simpleUtil.CheckErr(sge.Run("powershell", "explorer", cwd+".result.zip"))
		}
	}
}
