package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	math2 "github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/sge"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

// from https://forum.golangbridge.org/t/easy-way-for-letter-substitution-reverse-complementary-dna-sequence/20101
// from https://go.dev/play/p/IXI6PY7XUXN
var dnaComplement = strings.NewReplacer(
	"A", "T",
	"T", "A",
	"G", "C",
	"C", "G",
	"a", "t",
	"t", "a",
	"g", "c",
	"c", "g",
)

func Complement(s string) string {
	return dnaComplement.Replace(s)
}

// Reverse returns its argument string reversed rune-wise left to right.
// from https://github.com/golang/example/blob/master/stringmain/reverse.go
func Reverse(r []byte) []byte {
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func ReverseComplement(s string) string {
	return Complement(string(Reverse([]byte(s))))
}

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

func ParseInput(input, fqDir string) (info []map[string]string) {
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
				data["路径-R1"] = filepath.Join(fqDir, data["路径-R1"])
				data["路径-R2"] = filepath.Join(fqDir, data["路径-R2"])
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
			} else {
				data["fq"] =
					filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_1.clean.fq.gz") +
						"," +
						filepath.Join(fqDir, "00.CleanData", stra[0], stra[0]+"_2.clean.fq.gz")
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
			simpleUtil.CheckErr(sge.Run("powershell", "explorer", dir))
		}
	}
}

func summaryTxt(resultDir string, inputInfo []map[string]string) {
	var summary = osUtil.Create(filepath.Join(resultDir, "summary.txt"))

	//fmt.Println(strings.Join(textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt")), "\t"))
	var summaryTitle = textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt"))
	fmtUtil.FprintStringArray(summary, summaryTitle, "\t")

	for i := range inputInfo {
		SeqInfoMap[inputInfo[i]["id"]].WriteStatsTxt(summary)
	}
	// close file handle before Compress-Archive
	simpleUtil.CheckErr(summary.Close())
}

func summaryXlsx(resultDir string, inputInfo []map[string]string) {

	//fmt.Println(strings.Join(textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt")), "\t"))
	var summaryTitle = textUtil.File2Array(filepath.Join(etcPath, "title.Summary.txt"))

	// write summary.xlsx
	var (
		excel       = excelize.NewFile()
		summaryPath = fmt.Sprintf("summary-%s-%s.xlsx", filepath.Base(*outputDir), time.Now().Format("20060102"))
	)

	// Summary Sheet
	simpleUtil.CheckErr(excel.SetSheetName("Sheet1", "Summary"))
	// write Title
	for i, s := range summaryTitle {
		SetCellStr(excel, "Summary", 1+i, 1, s)
	}

	var sampleList []string
	for i := range inputInfo {
		var (
			id   = inputInfo[i]["id"]
			info = SeqInfoMap[id]
			rows = info.SummaryRow()
		)
		sampleList = append(sampleList, id)
		SetRow(excel, "Summary", 1, 2+i, rows)
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

	var keysInfo, _ = textUtil.File2MapArray(
		filepath.Join(etcPath, "统计字段.txt"), "\t", nil,
	)

	var titleIndex = make(map[string]int)
	var sampleList []string
	for i := range rows {
		if i == 0 {
			for j, v := range rows[i] {
				titleIndex[v] = j + 1
			}
			for _, v := range keysInfo {
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

			id    = rows[i][titleIndex["样品名称"]-1]
			info  = SeqInfoMap[id]
			stats = info.Stats
		)
		sampleList = append(sampleList, id)

		// 写入内部链接
		cellName = GetCellName(nrow, "样品名称", titleIndex)
		excel.SetCellHyperLink("Summary", cellName, id+".xlsx", "External")

		cellName = GetCellName(nrow, "分析reads", titleIndex)
		excel.SetCellInt("Summary", cellName, stats["AnalyzedReadsNum"])

		cellName = GetCellName(nrow, "正确reads", titleIndex)
		excel.SetCellInt("Summary", cellName, stats["RightReadsNum"])

		cellName = GetCellName(nrow, "收率", titleIndex)
		excel.SetCellFloat("Summary", cellName, info.YieldCoefficient, 4, 64)

		cellName = GetCellName(nrow, "单步准确率", titleIndex)
		excel.SetCellFloat("Summary", cellName, info.AverageYieldAccuracy, 4, 64)

		for _, v := range keysInfo {
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
