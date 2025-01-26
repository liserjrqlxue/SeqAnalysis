package seqAnalysis

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// regexp
var (
	isXlsx = regexp.MustCompile(`\.xlsx$`)
)

type Batch struct {
	OutputPrefix string
	BasePrefix   string

	LineLimit int
	Long      bool
	Rev       bool
	UseRC     bool
	UseKmer   bool
	LessMem   bool
	Zip       bool
	Plot      bool

	TitleTar     []string
	TitleStats   []string
	TitleSummary []string
	SheetList    []string
	Sheets       map[string]string

	InputInfo        []map[string]string
	StatisticalField []map[string]string

	SeqInfoMap       map[string]*SeqInfo
	ParallelStatsMap map[string]*ParallelTest
	FqSet            map[string][]*SeqInfo
}

func (batch *Batch) LoadConfig(cfgPath string, cfgFS embed.FS) {
	var sheetMap, _ = osUtil.FS2MapArray(osUtil.OpenFS("etc/sheet.txt", cfgPath, cfgFS), "\t", nil)
	for _, m := range sheetMap {
		batch.Sheets[m["Name"]] = m["SheetName"]
		batch.SheetList = append(batch.SheetList, m["SheetName"])
	}

	batch.TitleTar = osUtil.FS2Array(osUtil.OpenFS("etc/title.Tar.txt", cfgPath, cfgFS))
	batch.TitleStats = osUtil.FS2Array(osUtil.OpenFS("etc/title.Stats.txt", cfgPath, cfgFS))
	batch.TitleSummary = osUtil.FS2Array(osUtil.OpenFS("etc/title.Summary.txt", cfgPath, cfgFS))
	batch.StatisticalField, _ = osUtil.FS2MapArray(osUtil.OpenFS("etc/统计字段.txt", cfgPath, cfgFS), "\t", nil)
}

func (batch *Batch) LoadInput(input, workDir string) {
	// parse input
	var inputInfo, fqSet = ParseInput(input, workDir)
	batch.InputInfo = inputInfo
	batch.FqSet = fqSet
}

func (batch *Batch) Prepare() {
	// prepare output directory structure
	simpleUtil.CheckErr(os.MkdirAll(batch.OutputPrefix, 0755))
}

func (batch *Batch) WriteInfoTxt(path string) {
	file := osUtil.Create(path)
	defer simpleUtil.DeferClose(file)

	// write title
	fmtUtil.FprintStringArray(file, []string{"id", "index", "seq", "fq"}, "\t")

	// loop inputInfo for info.txt
	for _, data := range batch.InputInfo {
		fmtUtil.Fprintf(
			file,
			"%s\t%s\t%s\t%s\n",
			data["id"],
			data["index"],
			data["seq"],
			data["fq"],
		)
	}
}

func (batch *Batch) BuildSeqInfo() {
	for _, data := range batch.InputInfo {
		seqInfo := NewSeqInfo(data, batch.Sheets, batch.SheetList, batch.OutputPrefix, batch.LineLimit, batch.Long, batch.Rev, batch.UseRC, batch.UseKmer, batch.LessMem)
		batch.SeqInfoMap[seqInfo.Name] = seqInfo

		for _, fq := range seqInfo.Fastqs {
			batch.FqSet[fq] = append(batch.FqSet[fq], seqInfo)
		}
	}
}

func (batch *Batch) ConcurrencyRun(thread int) {
	// limit goroutine concurrency
	if thread == 0 {
		thread = min(len(batch.InputInfo), runtime.GOMAXPROCS(0))
	}

	go ReadAllFastq(batch.FqSet)

	var wg sync.WaitGroup
	for id := range batch.SeqInfoMap {
		wg.Add(1)
		go func(id string) {
			defer func() {
				wg.Done()
			}()
			slog.Info("SingleRun", "id", id)
			batch.SeqInfoMap[id].SingleRun(batch.OutputPrefix, batch.TitleTar, batch.TitleStats)
		}(id)
	}

	// wait goconcurrency thread to finish
	wg.Wait()
}

// CalculaterParallelTest calculater parallel test
func (batch *Batch) CalculaterParallelTest() {
	// 基于平行的统计
	for _, seqInfo := range batch.SeqInfoMap {
		var id = seqInfo.ParallelTestID
		var p, ok = batch.ParallelStatsMap[id]
		if !ok {
			p = &ParallelTest{}
			batch.ParallelStatsMap[id] = p
		}
		p.YieldCoefficient = append(p.YieldCoefficient, seqInfo.YieldCoefficient)
		p.AverageYieldAccuracy = append(p.AverageYieldAccuracy, seqInfo.AverageYieldAccuracy)
	}
	for _, p := range batch.ParallelStatsMap {
		p.Calculater()
	}
}

func (batch *Batch) Summary(input string) {
	// write summary.txt
	SummaryTxt(batch.OutputPrefix, batch.TitleSummary, batch.InputInfo, batch.SeqInfoMap)

	batch.CalculaterParallelTest()

	// write summary.xlsx
	if isXlsx.MatchString(input) {
		// update from input.xlsx
		Input2summaryXlsx(input, batch.OutputPrefix, batch.BasePrefix, batch.StatisticalField, batch.SeqInfoMap, batch.ParallelStatsMap)
	} else {
		SummaryXlsx(batch.OutputPrefix, batch.BasePrefix, batch.TitleSummary, batch.InputInfo, batch.SeqInfoMap)
	}
}

func (batch *Batch) Visual(exPath string) error {
	binPath := path.Join(exPath, "bin")
	if batch.Plot {
		cmd := exec.Command("Rscript", filepath.Join(binPath, "plot.R"), batch.OutputPrefix)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		slog.Info("Rscript", "cmd", cmd)
		err := cmd.Run()
		if err != nil {
			slog.Error("Rscript error:", "err", err)
			return err
		}
		// use powershell to plot
		// simpleUtil.CheckErr(sge.Run("powershell", fmt.Sprintf("powershell %s %s", filepath.Join(binPath, "plot.ps1"), batch.OutputPrefix)))
		// use Rscript to plot
		// simpleUtil.CheckErr(sge.Run("Rscript", filepath.Join(binPath, "plot.R"), batch.OutputPrefix))
	} else {
		slog.Info(fmt.Sprintf("Run Plot use Rscript: Rscript %s %s", filepath.Join(binPath, "plot.R"), batch.OutputPrefix))
	}
	return nil
}

func (batch *Batch) Compress() {
	if batch.Zip {
		Zip(batch.BasePrefix, batch.OutputPrefix)
	} else {
		slog.Info(
			"Run Zip use powershell",
			"cmd",
			fmt.Sprintf(
				"powershell Compress-Archive -Path %s/*.xlsx,%s/*.pdf -DestinationPath %s.result.zip -Force",
				batch.BasePrefix,
				batch.BasePrefix,
				batch.OutputPrefix,
			),
		)
	}
}

func (batch *Batch) BatchRun(input, workDir, exPath string, etcEMFS embed.FS, thread int) error {
	now := time.Now()

	cwd := simpleUtil.HandleError(os.Getwd())
	defer simpleUtil.CheckErr(os.Chdir(cwd))
	os.Chdir(workDir)

	batch.LoadConfig(exPath, etcEMFS)
	batch.LoadInput(input, workDir)
	batch.Prepare()
	batch.WriteInfoTxt(filepath.Join(batch.OutputPrefix, "info.txt"))
	batch.BuildSeqInfo()
	batch.ConcurrencyRun(thread)
	batch.Summary(input)
	err := batch.Visual(exPath)
	if err != nil {
		return err
	}
	// Compress-Archive to zip file on windows only when *zip is true
	batch.Compress()

	slog.Info("Done", "time", time.Since(now))
	return nil
}
