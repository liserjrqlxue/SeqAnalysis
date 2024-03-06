package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"time"

	util "SeqAnalysis/pkg/seqAnalysis"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// os
var (
	ex, _   = os.Executable()
	exPath  = filepath.Dir(ex)
	binPath = path.Join(exPath, "bin")
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
		"input.xlsx",
		"input info",
	)
	outputDir = flag.String(
		"o",
		"",
		"output directory, default is sub directory of CWD: [BaseName]+.分析",
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
	useRC = flag.Bool(
		"rc",
		false,
		"use reverse complement",
	)
	useKmer = flag.Bool(
		"kmer",
		false,
		"use kmer",
	)
	plot = flag.Bool(
		"plot",
		false,
		"plot",
	)
	lessMem = flag.Bool(
		"lessMem",
		false,
		"less memory: no BarCode Sheet",
	)
	lineLimit = flag.Int(
		"lineLimit",
		100000,
		"line limit",
	)
	debug = flag.Bool(
		"debug",
		false,
		"debug",
	)
	cpuProfile = flag.String(
		"cpu",
		"log.cpuProfile",
		"cpu profile",
	)
	memProfile = flag.String(
		"mem",
		"log.memProfile",
		"mem profile",
	)
)

// embed etc
//
//go:embed etc/*.txt
var etcEMFS embed.FS

func init() {
	var sheetMap, _ = osUtil.FS2MapArray(osUtil.OpenFS("etc/sheet.txt", exPath, etcEMFS), "\t", nil)
	for _, m := range sheetMap {
		Sheets[m["Name"]] = m["SheetName"]
		sheetList = append(sheetList, m["SheetName"])
	}

	TitleTar = osUtil.FS2Array(osUtil.OpenFS("etc/title.Tar.txt", exPath, etcEMFS))
	TitleStats = osUtil.FS2Array(osUtil.OpenFS("etc/title.Stats.txt", exPath, etcEMFS))
	TitleSummary = osUtil.FS2Array(osUtil.OpenFS("etc/title.Summary.txt", exPath, etcEMFS))
	StatisticalField, _ = osUtil.FS2MapArray(osUtil.OpenFS("etc/统计字段.txt", exPath, etcEMFS), "\t", nil)
}

func main() {
	flag.Parse()
	now := time.Now()

	if !*debug {
		*cpuProfile = ""
		*memProfile = ""
	} else {
		go LogMemStats()
	}

	if *cpuProfile != "" {
		var LogCPUProfile = osUtil.Create(*cpuProfile)
		defer simpleUtil.DeferClose(LogCPUProfile)
		pprof.StartCPUProfile(LogCPUProfile)
		defer pprof.StopCPUProfile()
	}

	// parse input
	var inputInfo, fqSet = ParseInput(*input, *fqDir)

	if *outputDir == "" {
		*outputDir = filepath.Base(simpleUtil.HandleError(os.Getwd())) + ".分析"
	}
	// pare output directory structure
	simpleUtil.CheckErr(os.MkdirAll(*outputDir, 0755))

	// parallel options
	// runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	if *thread == 0 {
		*thread = len(inputInfo)
	}
	chanList = make(chan bool, *thread)

	// write info.txt
	var info = osUtil.Create(filepath.Join(*outputDir, "info.txt"))
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
		var seqInfo = util.NewSeqInfo(data, Sheets, sheetList, *outputDir, *lineLimit, *long, *rev, *useRC, *useKmer, *lessMem)
		SeqInfoMap[seqInfo.Name] = seqInfo

		for _, fq := range seqInfo.Fastqs {
			fqSet[fq] = append(fqSet[fq], seqInfo)
		}
	}
	simpleUtil.CheckErr(info.Close())

	go util.ReadAllFastq(fqSet)

	var wg sync.WaitGroup
	for id := range SeqInfoMap {
		chanList <- true
		wg.Add(1)
		go func(id string) {
			defer func() {
				wg.Done()
				<-chanList
			}()
			slog.Info("SingleRun", "id", id)
			SeqInfoMap[id].SingleRun(*outputDir, TitleTar, TitleStats)
		}(id)
	}

	// wait goconcurrency thread to finish
	wg.Wait()

	// write summary.txt
	util.SummaryTxt(*outputDir, TitleSummary, inputInfo, SeqInfoMap)

	// 基于平行的统计
	for _, seqInfo := range SeqInfoMap {
		var id = seqInfo.ParallelTestID
		var p, ok = ParallelStatsMap[id]
		if !ok {
			p = &util.ParallelTest{}
			ParallelStatsMap[id] = p
		}
		p.YieldCoefficient = append(p.YieldCoefficient, seqInfo.YieldCoefficient)
		p.AverageYieldAccuracy = append(p.AverageYieldAccuracy, seqInfo.AverageYieldAccuracy)
	}
	for _, p := range ParallelStatsMap {
		p.Calculater()
	}

	// write summary.xlsx
	if isXlsx.MatchString(*input) {
		// update from input.xlsx
		util.Input2summaryXlsx(*input, *outputDir, filepath.Base(*outputDir), StatisticalField, SeqInfoMap, ParallelStatsMap)
	} else {
		util.SummaryXlsx(*outputDir, filepath.Base(*outputDir), TitleSummary, inputInfo, SeqInfoMap)
	}

	if *plot {
		// use Rscript to plot
		simpleUtil.CheckErr(sge.Run("Rscript", filepath.Join(binPath, "plot.R"), *outputDir))
	} else {
		slog.Info(fmt.Sprintf("Run Plot use Rscript: Rscript %s %s", filepath.Join(binPath, "plot.R"), *outputDir))
	}

	// Compress-Archive to zip file on windows only when *zip is true
	if *zip {
		util.Zip(*outputDir)
	} else {
		slog.Info(fmt.Sprintf("Run Zip use powershell: powershell Compress-Archive -Path %s/*.xlsx,%s/*.pdf -DestinationPath %s.result.zip -Force", *outputDir, *outputDir, *outputDir))
	}

	if *memProfile != "" {
		var LogMemProfile = osUtil.Create(*memProfile)
		defer simpleUtil.DeferClose(LogMemProfile)
		pprof.WriteHeapProfile(LogMemProfile)
	}

	slog.Info("Done", "time", time.Since(now))
}
