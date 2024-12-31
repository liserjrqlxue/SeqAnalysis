package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
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
		"",
		"output, default: input.cutTail.xlsx",
	)
	tag = flag.String(
		"t",
		"cutTail",
		"tag add to output",
	)
	cut = flag.Int(
		"c",
		8,
		"cut tail length, unshifted to trailer",
	)
	length = flag.Int(
		"l",
		8,
		"length of trailer",
	)
	polyA = flag.Bool(
		"p",
		false,
		"append polyA trailer",
	)
)

func main() {
	flag.Parse()

	if *output == "" {
		*output = *input + "." + *tag + ".xlsx"
	}

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
		targetSynthesisSeqColIdx = -1
		trailerSeqColIdx         = len(rows[0])
		cIdx1                    = 'A'
		cIdx2                    = 'A'
	)

	// 遍历所有行，修改指定列的值
	for i, row := range rows {
		if i == 0 {
			for cIdx, title := range row {
				switch title {
				case "合成序列":
					targetSynthesisSeqColIdx = cIdx
				case "后靶标":
					trailerSeqColIdx = cIdx
				}
			}
			if targetSynthesisSeqColIdx == -1 {
				log.Fatal("not found 合成序列")
			}
			cIdx1 += rune(targetSynthesisSeqColIdx)
			cIdx2 += rune(trailerSeqColIdx)
			if trailerSeqColIdx == len(row) {
				simpleUtil.CheckErr(f.SetCellStr("Sheet1", string(cIdx2)+strconv.Itoa(i+1), "后靶标"))
			}
			continue
		}
		slog.Debug("cutTail", "row", i+1)

		var targetSynthesisSeq = row[targetSynthesisSeqColIdx]
		if len(targetSynthesisSeq) < *cut {
			slog.Info("cutTail", "row", i+1, "targetSynthesisSeq", targetSynthesisSeq, "len(targetSynthesisSeq)", len(targetSynthesisSeq), "cut", *cut, "row", row)
			continue
		}
		var trailerSeq = targetSynthesisSeq[len(targetSynthesisSeq)-(*cut):]
		targetSynthesisSeq = targetSynthesisSeq[:len(targetSynthesisSeq)-(*cut)]
		if trailerSeqColIdx < len(row) {
			trailerSeq += row[trailerSeqColIdx]
		}
		if *polyA && len(trailerSeq) < *length {
			trailerSeq += strings.Repeat("A", *length-len(trailerSeq))
		}

		simpleUtil.CheckErr(f.SetCellStr("Sheet1", string(cIdx1)+strconv.Itoa(i+1), targetSynthesisSeq))
		simpleUtil.CheckErr(f.SetCellStr("Sheet1", string(cIdx2)+strconv.Itoa(i+1), trailerSeq))
	}

	// 保存文件到新的路径
	if err := f.SaveAs(*output); err != nil {
		fmt.Println(err)
	}
}
