package seqAnalysis

import (
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

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

func GetCellValue(xlsx *excelize.File, sheet string, col, row int) string {
	return simpleUtil.HandleError(
		xlsx.GetCellValue(
			sheet,
			simpleUtil.HandleError(excelize.CoordinatesToCellName(col, row)),
		),
	)
}

func SetCellValue(xlsx *excelize.File, sheet string, col, row int, value interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetCellValue(
			sheet,
			simpleUtil.HandleError(excelize.CoordinatesToCellName(col, row)),
			value,
		),
	)
}

func SetCellStr(xlsx *excelize.File, sheet string, col, row int, value string) {
	simpleUtil.CheckErr(
		xlsx.SetCellStr(
			sheet,
			simpleUtil.HandleError(excelize.CoordinatesToCellName(col, row)),
			value,
		),
	)
}

func SetRow(xlsx *excelize.File, sheet string, col, row int, value []interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetSheetRow(
			sheet,
			simpleUtil.HandleError(excelize.CoordinatesToCellName(col, row)),
			&value,
		),
	)
}

func SetCol(xlsx *excelize.File, sheet string, col, row int, value []interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetSheetCol(
			sheet,
			simpleUtil.HandleError(excelize.CoordinatesToCellName(col, row)),
			&value,
		),
	)
}

func MergeCells(xlsx *excelize.File, sheet string, col1, row1, col2, row2 int) {
	var (
		hCel = simpleUtil.HandleError(excelize.CoordinatesToCellName(col1, row1))
		vCel = simpleUtil.HandleError(excelize.CoordinatesToCellName(col2, row2))
	)
	simpleUtil.CheckErr(xlsx.MergeCell(sheet, hCel, vCel))
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
			row = row[:5]
			cellName := simpleUtil.HandleError(excelize.CoordinatesToCellName(1+i*5, rIdx))
			excel.SetSheetRow(sheetName, cellName, &row)

			rIdx++
		}
	}
}
