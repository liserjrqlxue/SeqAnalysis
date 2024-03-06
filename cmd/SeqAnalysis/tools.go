package main

import (
	"embed"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	util "SeqAnalysis/pkg/seqAnalysis"

	"github.com/liserjrqlxue/goUtil/osUtil"
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

func ParseInput(input, fqDir string) (info []map[string]string, fqSet map[string][]*util.SeqInfo) {
	fqSet = make(map[string][]*util.SeqInfo)
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

					fqSet[data["路径-R1"]] = []*util.SeqInfo{}
				}
				if data["路径-R2"] != "" {
					data["路径-R2"] = filepath.Join(fqDir, data["路径-R2"])

					fqSet[data["路径-R2"]] = []*util.SeqInfo{}
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
					fqSet[v] = []*util.SeqInfo{}
				}
			} else {
				fq1 := filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_1.clean.fq.gz")
				fq2 := filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_2.clean.fq.gz")
				data["fq"] = fq1 + "," + fq2

				fqSet[fq1] = []*util.SeqInfo{}
				fqSet[fq2] = []*util.SeqInfo{}
			}
			info = append(info, data)
		}
	}
	return
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
