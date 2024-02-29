package main

import (
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/xuri/excelize/v2"
)

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
