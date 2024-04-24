package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/xuri/excelize/v2"
)

// flag
var (
	input = flag.String(
		"i",
		"input.xlsx",
		"input",
	)
	output = flag.String(
		"o",
		"merged.xlsx",
		"output",
	)
)

func main() {
	// 打开现有的xlsx文件
	f, err := excelize.OpenFile(*input)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取Sheet1上的所有单元格
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var (
		fq1Idx = -1
		fq2Idx = -1
		cIdx1  = 'A'
		cIdx2  = 'A'
	)

	// 遍历所有行，修改指定列的值
	for i, row := range rows {
		if i == 0 {
			for cIdx, title := range row {
				switch title {
				case "路径-R1":
					fq1Idx = cIdx
				case "路径-R2":
					fq2Idx = cIdx
				}
			}
			if fq1Idx == -1 || fq2Idx == -1 {
				log.Fatalf("not found fq1Idx=%d fq2Idx=%d", fq1Idx, fq2Idx)
			}
			cIdx1 += rune(fq1Idx)
			cIdx2 += rune(fq2Idx)
			continue
		}
		fq1 := row[fq1Idx]
		merged := strings.Replace(fq1, "1.fq.gz", "merged.fq.gz", -1)
		simpleUtil.CheckErr(f.SetCellStr("Sheet1", string(cIdx1)+strconv.Itoa(i+1), merged))
		simpleUtil.CheckErr(f.SetCellStr("Sheet1", string(cIdx2)+strconv.Itoa(i+1), ""))
	}

	// 保存文件到新的路径
	if err := f.SaveAs(*output); err != nil {
		fmt.Println(err)
	}
}
