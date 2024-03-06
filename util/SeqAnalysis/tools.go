package main

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	util "SeqAnalysis/pkg/seqAnalysis"

	gzip "github.com/klauspost/pgzip"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	math2 "github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

// Open is a function that opens a file from the given path using the embed.FS file system.
//
// It takes three parameters:
//   - path: a string that represents the path of the file to be opened.
//   - exPath: a string that represents the extra path to be joined with the file path in case the file is not found in the embed.FS file system.
//   - embedFS: an embed.FS file system that provides access to embedded files.
//
// It returns two values:
//   - file: an io.ReadCloser that represents the opened file.
//   - err: an error indicating any error that occurred during the file opening process.
func Open(path, exPath string, embedFS embed.FS) (file io.ReadCloser, err error) {
	file, err = embedFS.Open(path)
	if err != nil {
		return os.Open(filepath.Join(exPath, path))
	}
	return
}

func Rows2Map(rows [][]string) (result []map[string]string) {
	var title = rows[0]
	for i, row := range rows {
		if i == 0 {
			continue
		}
		var data = make(map[string]string)
		for i, v := range row {
			data[title[i]] = v
		}
		result = append(result, data)
	}
	return
}

type SeqInfo util.SeqInfo

func ParseInput(input, fqDir string) (info []map[string]string, fqSet map[string][]*SeqInfo) {
	fqSet = make(map[string][]*SeqInfo)
	if isXlsx.MatchString(input) {
		xlsx, err := excelize.OpenFile(input)
		simpleUtil.CheckErr(err)
		rows, err := xlsx.GetRows("Summary")
		if err != nil {
			rows, err = xlsx.GetRows("Sheet1")
		}
		simpleUtil.CheckErr(err)
		info = Rows2Map(rows)

		for _, data := range info {
			data["id"] = data["样品名称"]
			data["index"] = data["靶标序列"]
			data["seq"] = data["合成序列"]
			if fqDir != "" {
				if data["路径-R1"] != "" {
					data["路径-R1"] = filepath.Join(fqDir, data["路径-R1"])

					fqSet[data["路径-R1"]] = []*SeqInfo{}
				}
				if data["路径-R2"] != "" {
					data["路径-R2"] = filepath.Join(fqDir, data["路径-R2"])

					fqSet[data["路径-R2"]] = []*SeqInfo{}
				}
			}
			data["fq"] = data["路径-R1"] + "," + data["路径-R2"]
		}
	} else {
		var seqList = textUtil.File2Array(input)
		for _, s := range seqList {
			var data = make(map[string]string)
			var stra = strings.Split(strings.TrimSuffix(s, "\r"), "\t")
			data["id"] = stra[0]
			data["index"] = stra[1]
			data["seq"] = stra[2]
			if len(stra) > 3 {
				var fqList = stra[3:]
				if fqDir != "" {
					for i := range fqList {
						fqList[i] = filepath.Join(fqDir, fqList[i])
					}
				}
				data["fq"] = strings.Join(fqList, ",")

				for _, v := range fqList {
					fqSet[v] = []*SeqInfo{}
				}
			} else {
				fq1 := filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_1.clean.fq.gz")
				fq2 := filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_2.clean.fq.gz")
				data["fq"] = fq1 + "," + fq2

				fqSet[fq1] = []*SeqInfo{}
				fqSet[fq2] = []*SeqInfo{}
			}
			info = append(info, data)
		}
	}
	return
}

func Zip(dir string) {
	if runtime.GOOS == "windows" {
		var args = []string{
			"Compress-Archive",
			"-Path",
			fmt.Sprintf("\"%s/*.xlsx\",\"%s/*.pdf\"", dir, dir),
			"-DestinationPath",
			dir + ".result.zip",
			"-Force",
		}
		log.Println(strings.Join(args, " "))
		if *zip {
			simpleUtil.CheckErr(sge.Run("powershell", args...))
			absDir, err := filepath.Abs(dir)
			if err != nil {
				slog.Error("get abs dir error", "dir", dir, "err", err)
				return
			}
			simpleUtil.CheckErr(sge.Run("powershell", "explorer", absDir))
		}
	}
}

func summaryTxt(resultDir string, inputInfo []map[string]string) {
	var summary = osUtil.Create(filepath.Join(resultDir, "summary.txt"))

	fmtUtil.FprintStringArray(summary, TitleSummary, "\t")

	for i := range inputInfo {
		SeqInfoMap[inputInfo[i]["id"]].WriteStatsTxt(summary)
	}
	// close file handle before Compress-Archive
	simpleUtil.CheckErr(summary.Close())
}

func summaryXlsx(resultDir string, inputInfo []map[string]string) {

	// write summary.xlsx
	var (
		excel       = excelize.NewFile()
		summaryPath = fmt.Sprintf("summary-%s-%s.xlsx", filepath.Base(*outputDir), time.Now().Format("20060102"))
	)

	// Summary Sheet
	simpleUtil.CheckErr(excel.SetSheetName("Sheet1", "Summary"))
	// write Title
	for i, s := range TitleSummary {
		util.SetCellStr(excel, "Summary", 1+i, 1, s)
	}

	var sampleList []string
	for i := range inputInfo {
		var (
			id   = inputInfo[i]["id"]
			info = SeqInfoMap[id]
			rows = info.SummaryRow()
		)
		sampleList = append(sampleList, id)
		util.SetRow(excel, "Summary", 1, 2+i, rows)
	}

	// get cwd
	cwd, err := os.Getwd()
	simpleUtil.CheckErr(err)
	// change to resultDir
	simpleUtil.CheckErr(os.Chdir(resultDir))

	AddSteps2Sheet(excel, sampleList)

	// save summary.xlsx
	log.Println("SaveAs ", summaryPath)
	simpleUtil.CheckErr(excel.SaveAs(summaryPath))
	// change back
	simpleUtil.CheckErr(os.Chdir(cwd))
}

func input2summaryXlsx(input, resultDir string) {
	var excel, err = excelize.OpenFile(input)
	simpleUtil.CheckErr(err)
	rows, err := excel.GetRows("Summary")
	if err != nil {
		rows, err = excel.GetRows("Sheet1")
		simpleUtil.CheckErr(err)
		simpleUtil.CheckErr(excel.SetSheetName("Sheet1", "Summary"))
	}

	var titleIndex = make(map[string]int)
	var sampleList []string
	for i := range rows {
		if i == 0 {
			for j, v := range rows[i] {
				titleIndex[v] = j + 1
			}
			for _, v := range StatisticalField {
				var title = v["summary_title"]
				var _, ok = titleIndex[title]
				if !ok {
					var cellName = GetCellName(1, title, titleIndex)
					excel.SetCellStr("Summary", cellName, title)
				}
				title += "/个数"
				_, ok = titleIndex[title]
				if !ok {
					var cellName = GetCellName(1, title, titleIndex)
					excel.SetCellStr("Summary", cellName, title)
				}
			}
			continue
		}
		var (
			nrow     = i + 1
			cellName string

			id           = rows[i][titleIndex["样品名称"]-1]
			info         = SeqInfoMap[id]
			stats        = info.Stats
			pId          = info.ParallelTestID
			parallelTest = ParallelStatsMap[pId]
		)
		sampleList = append(sampleList, id)

		// 写入内部链接
		cellName = GetCellName(nrow, "样品名称", titleIndex)
		excel.SetCellHyperLink("Summary", cellName, id+".xlsx", "External")

		cellName = GetCellName(nrow, "分析reads", titleIndex)
		excel.SetCellInt("Summary", cellName, stats["AnalyzedReadsNum"])

		cellName = GetCellName(nrow, "正确reads", titleIndex)
		excel.SetCellInt("Summary", cellName, info.RightReadsNum)

		cellName = GetCellName(nrow, "收率", titleIndex)
		excel.SetCellFloat("Summary", cellName, info.YieldCoefficient, 4, 64)
		cellName = GetCellName(nrow, "平均收率", titleIndex)
		excel.SetCellFloat("Summary", cellName, parallelTest.YieldCoefficientMean, 4, 64)
		cellName = GetCellName(nrow, "收率误差", titleIndex)
		excel.SetCellFloat("Summary", cellName, parallelTest.YieldCoefficientSD, 4, 64)

		cellName = GetCellName(nrow, "单步准确率", titleIndex)
		excel.SetCellFloat("Summary", cellName, info.AverageYieldAccuracy, 4, 64)
		cellName = GetCellName(nrow, "平均准确率", titleIndex)
		excel.SetCellFloat("Summary", cellName, parallelTest.AverageYieldAccuracyMean, 4, 64)
		cellName = GetCellName(nrow, "准确率误差", titleIndex)
		excel.SetCellFloat("Summary", cellName, parallelTest.AverageYieldAccuracySD, 4, 64)

		// 写入统计
		for _, v := range StatisticalField {
			var (
				key   = v["key"]
				title = v["summary_title"]
			)
			cellName = GetCellName(nrow, title, titleIndex)
			excel.SetCellFloat(
				"Summary", cellName,
				math2.DivisionInt(stats[key], stats["AnalyzedReadsNum"]),
				4, 64,
			)
			cellName = GetCellName(nrow, title+"/个数", titleIndex)
			excel.SetCellInt("Summary", cellName, stats[key])
		}
	}

	// get cwd
	cwd, err := os.Getwd()
	simpleUtil.CheckErr(err)
	// change to resultDir
	simpleUtil.CheckErr(os.Chdir(resultDir))

	AddSteps2Sheet(excel, sampleList)

	var summaryPath = fmt.Sprintf("summary-%s-%s.xlsx", filepath.Base(*outputDir), time.Now().Format("20060102"))
	// save summary.xlsx
	log.Println("SaveAs ", summaryPath)
	simpleUtil.CheckErr(excel.SaveAs(summaryPath))
	// change back
	simpleUtil.CheckErr(os.Chdir(cwd))
}

func GetCellName(row int, colName string, name2col map[string]int) string {
	var col, ok = name2col[colName]
	if !ok {
		col = len(name2col) + 1
		name2col[colName] = col
	}
	var cellName, err = excelize.CoordinatesToCellName(col, row)
	simpleUtil.CheckErr(err)
	return cellName
}

// AddSteps2Sheet Add one.step.error.rate.txt to 单步错误率 sheet
func AddSteps2Sheet(excel *excelize.File, list []string) {
	var sheetName = "单步错误率-横排"
	simpleUtil.HandleError(excel.NewSheet(sheetName))
	for i := range list {
		var rIdx = 1
		id := list[i]
		cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
		simpleUtil.CheckErr(err)
		// write title
		excel.SetSheetRow(sheetName, cellName, &[]string{"名字", "合成前4nt-" + id, "合成碱基-" + id, "合成位置-" + id, "单步错误率-" + id})
		rIdx++
		for _, row := range textUtil.File2Slice(id+".one.step.error.rate.txt", "\t") {
			cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
			simpleUtil.CheckErr(err)
			excel.SetSheetRow(sheetName, cellName, &row)
			rIdx++
		}
	}
}

func LogMemStats() {
	var m runtime.MemStats
	var logFile = osUtil.Create("log.MemStats.txt")
	defer simpleUtil.DeferClose(logFile)
	logger := slog.New(slog.NewTextHandler(logFile, nil))
	for {
		runtime.ReadMemStats(&m)
		logger.Info(
			"memStats2",
			"Alloc", m.Alloc,
			"TotalAlloc", m.TotalAlloc,
			"Sys", m.Sys,
			"HeapAlloc", m.HeapAlloc,
			"HeapSys", m.HeapSys,
			"HeapIdle", m.HeapIdle,
			"HeapInuse", m.HeapInuse,
			"HeapReleased", m.HeapReleased,
			"HeapObjects", m.HeapObjects,
			"StackInuse", m.StackInuse,
			"StackSys", m.StackSys,
			"MSpanInuse", m.MSpanInuse,
			"MSpanSys", m.MSpanSys,
			"MCacheInuse", m.MCacheInuse,
			"MCacheSys", m.MCacheSys,
			"BuckHashSys", m.BuckHashSys,
			"GCSys", m.GCSys,
			"OtherSys", m.OtherSys,
			"NextGC", m.NextGC,
			"LastGC", m.LastGC,
			"PauseTotalNs", m.PauseTotalNs,
			"NumGC", m.NumGC,
			"NumForcedGC", m.NumForcedGC,
			"GCCPUFraction", m.GCCPUFraction,
		)
		time.Sleep(1 * time.Second)
	}
}

func ReadFastq(fastq string, chanList []chan string) {
	var (
		file    = osUtil.Open(fastq)
		scanner *bufio.Scanner
		i       = -1
	)
	if gz.MatchString(fastq) {
		scanner = bufio.NewScanner(simpleUtil.HandleError(gzip.NewReader(file)))
	} else {
		scanner = bufio.NewScanner(file)
	}

	for scanner.Scan() {
		var s = scanner.Text()
		i++
		if i%4 != 1 {
			continue
		}
		for _, ch := range chanList {
			ch <- s
		}
	}

	simpleUtil.CheckErr(file.Close())
	slog.Info("ReadFastq Done", "fq", fastq)
}

func ReadAllFastq(fqSet map[string][]*SeqInfo) {
	var wg sync.WaitGroup

	// read fastqs 多对多 到各个 SeqChan
	wg.Add(len(fqSet))
	for fastq, seqInfos := range fqSet {
		if fastq == "" {
			for _, seqInfo := range seqInfos {
				seqInfo.SeqChanWG.Done()
			}
			wg.Done()
			continue
		}
		slog.Info("ReadFastq", "fq", fastq)
		// read fastq 一对多 到各个 SeqChan
		go func(fastq string, seqInfos []*SeqInfo) {
			var chanList []chan string
			for _, seqInfo := range seqInfos {
				chanList = append(chanList, seqInfo.SeqChan)
			}
			ReadFastq(fastq, chanList)
			for _, seqInfo := range seqInfos {
				seqInfo.SeqChanWG.Done()
			}
			wg.Done()
		}(fastq, seqInfos)
	}
	// wait readDone
	wg.Wait()
	slog.Info("ReadAllFastq Done")
}
