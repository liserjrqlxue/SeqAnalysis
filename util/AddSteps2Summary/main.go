package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

// flag
var (
	excel = flag.String(
		"e",
		"",
		"input excel file",
	)
	dir = flag.String(
		"d",
		"",
		"input dir",
	)
)

func main() {
	flag.Parse()
	if *excel == "" {
		flag.Usage()
		log.Fatal("input excel is nil")
	}
	excelDir := filepath.Dir(*excel)
	excelBase := filepath.Base(*excel)

	if *dir != "" {
		*dir = simpleUtil.HandleError(filepath.Abs(*dir)).(string)
	}

	// change to excelDir
	simpleUtil.CheckErr(os.Chdir(excelDir))

	excelizeFile, err := excelize.OpenFile(excelBase)
	simpleUtil.CheckErr(err)
	rows, err := excelizeFile.GetRows("Summary")
	simpleUtil.CheckErr(err)
	data := rows2mapArray(rows)

	simpleUtil.HandleError(excelizeFile.NewSheet("单步准确率"))
	// write title
	excelizeFile.SetSheetRow("单步准确率", "A1", &[]string{"名字", "合成前4nt", "合成碱基", "合成位置", "单步准确率"})

	var rIdx = 2
	for i := range data {
		id := data[i]["名字"]
		for _, row := range textUtil.File2Slice(filepath.Join(*dir, id+".one.step.accuracy.rate.txt"), "\t") {
			cellName, err := excelize.CoordinatesToCellName(1, rIdx)
			simpleUtil.CheckErr(err)
			excelizeFile.SetSheetRow("单步准确率", cellName, &row)
			rIdx++
		}
	}

	_, err = excelizeFile.NewSheet("单步准确率-横排")
	simpleUtil.CheckErr(err)

	for i := range data {
		var rIdx = 1
		id := data[i]["名字"]
		cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
		simpleUtil.CheckErr(err)
		// write title
		excelizeFile.SetSheetRow("单步准确率-横排", cellName, &[]string{"名字", "合成前4nt-" + id, "合成碱基-" + id, "合成位置-" + id, "单步准确率-" + id})
		rIdx++
		for _, row := range textUtil.File2Slice(filepath.Join(*dir, id+".one.step.accuracy.rate.txt"), "\t") {
			cellName, err := excelize.CoordinatesToCellName(1+i*5, rIdx)
			simpleUtil.CheckErr(err)
			excelizeFile.SetSheetRow("单步准确率-横排", cellName, &row)
			rIdx++
		}
	}

	simpleUtil.CheckErr(excelizeFile.Save())
	simpleUtil.CheckErr(excelizeFile.Close())

}

func rows2mapArray(rows [][]string) (result []map[string]string) {
	var title []string
	for i := range rows {
		if i == 0 {
			title = rows[i]
		} else {
			data := make(map[string]string)
			for j := range rows[i] {
				data[title[j]] = rows[i][j]
			}
			result = append(result, data)
		}
	}
	return
}
