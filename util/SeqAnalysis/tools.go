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

func ParseInput(input string) (info []map[string]string) {
	if isXlsx.MatchString(input) {
		xlsx, err := excelize.OpenFile(input)
		simpleUtil.CheckErr(err)
		rows, err := xlsx.GetRows("Sheet1")
		simpleUtil.CheckErr(err)
		info = Rows2Map(rows)

		for _, data := range info {
			data["id"] = data["样品名称"]
			data["index"] = data["靶标序列"]
			data["seq"] = data["合成序列"]
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
				data["fq"] = strings.Join(stra[3:], ",")
			} else {
				data["fq"] =
					filepath.Join("00.CleanData", stra[0], stra[0]+"_1.clean.fq.gz") +
						"," +
						filepath.Join("00.CleanData", stra[0], stra[0]+"_2.clean.fq.gz")
			}
			info = append(info, data)
		}
	}
	return
}

func Zip(resultDir string) {
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
		summaryXlsx = excelize.NewFile()
		summaryPath = fmt.Sprintf("summary-%s-%s.xlsx", filepath.Base(*outputDir), time.Now().Format("20060102"))
	)

	// Summary Sheet
	simpleUtil.CheckErr(summaryXlsx.SetSheetName("Sheet1", "Summary"))
	// write Title
	for i, s := range summaryTitle {
		SetCellStr(summaryXlsx, "Summary", 1+i, 1, s)
	}

	for i := range inputInfo {
		var (
			info = SeqInfoMap[inputInfo[i]["id"]]
			rows = info.SummaryRow()
		)
		SetRow(summaryXlsx, "Summary", 1, 2+i, rows)
	}

	// get cwd
	cwd, err := os.Getwd()
	simpleUtil.CheckErr(err)
	// change to resultDir
	simpleUtil.CheckErr(os.Chdir(resultDir))

	// 单步准确率
	simpleUtil.HandleError(summaryXlsx.NewSheet("单步准确率"))
	summaryXlsx.SetSheetRow("单步准确率", "A1", &[]string{"名字", "合成前4nt", "合成碱基", "合成位置", "单步准确率"})

	var rIdx = 2
	for i := range inputInfo {
		id := inputInfo[i]["id"]
		for _, row := range textUtil.File2Slice(id+".one.step.accuracy.rate.txt", "\t") {
			cellName, err := excelize.CoordinatesToCellName(1, rIdx)
			simpleUtil.CheckErr(err)
			summaryXlsx.SetSheetRow("单步准确率", cellName, &row)
			rIdx++
		}
	}

	// 单步准确率-横排
	simpleUtil.HandleError(summaryXlsx.NewSheet("单步准确率-横排"))
	for i := range inputInfo {
		var rIdx = 1
		id := inputInfo[i]["id"]
		cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
		simpleUtil.CheckErr(err)
		// write title
		summaryXlsx.SetSheetRow("单步准确率-横排", cellName, &[]string{"名字", "合成前4nt-" + id, "合成碱基-" + id, "合成位置-" + id, "单步准确率-" + id})
		rIdx++
		for _, row := range textUtil.File2Slice(id+".one.step.accuracy.rate.txt", "\t") {
			cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
			simpleUtil.CheckErr(err)
			summaryXlsx.SetSheetRow("单步准确率-横排", cellName, &row)
			rIdx++
		}
	}

	// save summary.xlsx
	log.Println("SaveAs ", summaryPath)
	simpleUtil.CheckErr(summaryXlsx.SaveAs(summaryPath))
	// change back
	simpleUtil.CheckErr(os.Chdir(cwd))
}

func input2summaryXlsx(input, resultDir string) {
	var summaryXlsx, err = excelize.OpenFile(input)
	simpleUtil.CheckErr(err)
	simpleUtil.CheckErr(summaryXlsx.SetSheetName("Sheet1", "Summary"))
	rows, err := summaryXlsx.GetRows("Summary")
	simpleUtil.CheckErr(err)
	title := rows[0]
	var titleIndex = make(map[string]int)
	for i := range title {
		titleIndex[title[i]] = i
	}

	for i := range rows {
		if i == 0 {
			continue
		}
		var id = rows[i][titleIndex["样品名称"]]
		var info = SeqInfoMap[id]
		var stats = info.Stats

		cellName, err := excelize.CoordinatesToCellName(titleIndex["reads"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellInt("Summary", cellName, stats["AllReadsNum"])

		cellName, err = excelize.CoordinatesToCellName(titleIndex["合成"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellInt("Summary", cellName, stats["RightReadsNum"])

		cellName, err = excelize.CoordinatesToCellName(titleIndex["收率"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellFloat("Summary", cellName, info.YieldCoefficient, 4, 64)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["平均收率"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellFloat("Summary", cellName, info.AverageYieldAccuracy, 4, 64)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["缺1个"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorDel1ReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorDel1ReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["缺2个（连续）"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorDelDupReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorDelDupReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["缺3个以上（连续）"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorDelDup3ReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorDelDup3ReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["缺2个（不连续）"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorDel2ReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorDel2ReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["缺3个以上（不连续）"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorDel3ReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorDel3ReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["第一个缺失位置"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellInt("Summary", cellName, info.DeletionDup3Index+1)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["插入"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorInsReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorInsReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["插入+缺失"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorInsDelReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorInsDelReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["突变"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorMutReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorMutReadsNum"],
			),
		)

		cellName, err = excelize.CoordinatesToCellName(titleIndex["其他错误"]+1, i+1)
		simpleUtil.CheckErr(err)
		summaryXlsx.SetCellStr(
			"Summary",
			cellName,
			fmt.Sprintf(
				"%.4f(%d)",
				math2.DivisionInt(stats["ErrorOtherReadsNum"], stats["AnalyzedReadsNum"]),
				stats["ErrorOtherReadsNum"],
			),
		)
	}

	simpleUtil.CheckErr(summaryXlsx.SaveAs("summary.xlsx"))

}
