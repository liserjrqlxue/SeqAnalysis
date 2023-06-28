package main

import (
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/xuri/excelize/v2"
)

func SetCellValue(xlsx *excelize.File, sheet string, col, row int, value interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetCellValue(
			sheet,
			simpleUtil.HandleError(
				excelize.CoordinatesToCellName(col, row),
			).(string),
			value,
		),
	)
}

func SetCellStr(xlsx *excelize.File, sheet string, col, row int, value string) {
	simpleUtil.CheckErr(
		xlsx.SetCellStr(
			sheet,
			simpleUtil.HandleError(
				excelize.CoordinatesToCellName(col, row),
			).(string),
			value,
		),
	)
}

func SetRow(xlsx *excelize.File, sheet string, col, row int, value []interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetSheetRow(
			sheet,
			simpleUtil.HandleError(
				excelize.CoordinatesToCellName(col, row),
			).(string),
			&value,
		),
	)
}

func SetCol(xlsx *excelize.File, sheet string, col, row int, value []interface{}) {
	simpleUtil.CheckErr(
		xlsx.SetSheetCol(
			sheet,
			simpleUtil.HandleError(
				excelize.CoordinatesToCellName(col, row),
			).(string),
			&value,
		),
	)
}
