package main

import (
	"embed"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	util "SeqAnalysis/pkg/seqAnalysis"

	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// os
var (
	ex, _  = os.Executable()
	exPath = filepath.Dir(ex)
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
	short = flag.Int(
		"short",
		0,
		"filter short length",
	)
)

// embed etc
//
//go:embed etc/*.txt
var etcEMFS embed.FS

func main() {
	t0 := time.Now()
	flag.Parse()

	if !*debug {
		*cpuProfile = ""
		*memProfile = ""
	} else {
		go util.LogMemStats()
	}

	if *cpuProfile != "" {
		var LogCPUProfile = osUtil.Create(*cpuProfile)
		defer simpleUtil.DeferClose(LogCPUProfile)
		pprof.StartCPUProfile(LogCPUProfile)
		defer pprof.StopCPUProfile()
	}

	if *outputDir == "" {
		*outputDir = filepath.Base(simpleUtil.HandleError(os.Getwd())) + ".分析"
	}

	util.Short = *short

	var batch = util.Batch{
		OutputPrefix: *outputDir,
		BasePrefix:   filepath.Base(*outputDir),

		LineLimit: *lineLimit,
		Long:      *long,
		Rev:       *rev,
		UseRC:     *useRC,
		UseKmer:   *useKmer,
		LessMem:   *lessMem,
		Zip:       *zip,
		Plot:      *plot,

		Sheets:           make(map[string]string),
		SeqInfoMap:       make(map[string]*util.SeqInfo),
		ParallelStatsMap: make(map[string]*util.ParallelTest),
	}

	batch.BatchRun(*input, *fqDir, exPath, etcEMFS, *thread)

	if *memProfile != "" {
		var LogMemProfile = osUtil.Create(*memProfile)
		defer simpleUtil.DeferClose(LogMemProfile)
		pprof.WriteHeapProfile(LogMemProfile)
	}

	slog.Info("Done", "time", time.Since(t0))

}
