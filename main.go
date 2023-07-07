package main

import (
	"flag"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
	verbose = flag.Int(
		"v",
		0,
		"verbose level\n\t1: more log\n\t2: unmatched.txt",
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

	var seqList = textUtil.File2Array(*input)
	for _, s := range seqList {
		SingelRun(s)
	}

	if runtime.GOOS == "windows" {
		var cwd = filepath.Base(simpleUtil.HandleError(os.Getwd()).(string))
		simpleUtil.CheckErr(
			sge.Run("powershell",
				"Compress-Archive",
				"-Path",
				"result",
				"-DestinationPath",
				cwd+".result.zip",
				"-Force"),
		)
		simpleUtil.CheckErr(sge.Run("powershell", "explorer", cwd+".result.zip"))
	}
}
