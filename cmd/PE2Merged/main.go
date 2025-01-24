package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
		"output",
	)
	sheet = flag.String(
		"s",
		"Sheet1",
		"sheet name",
	)
	head = flag.Int(
		"h",
		0,
		"head",
	)
	run = flag.Bool(
		"run",
		false,
		"run NGmerge",
	)
	mergedDir = flag.String(
		"d",
		"",
		"merged dir",
	)
)

func main() {
	flag.Parse()
	if *output == "" {
		*output = strings.Replace(*input, ".xlsx", ".merged.xlsx", 1)
	}

	// 打开现有的xlsx文件
	f, err := excelize.OpenFile(*input)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取 `*sheet` 上的所有单元格
	rows, err := f.GetRows(*sheet)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var (
		fq1Idx = -1
		fq2Idx = -1

		cIdx1 = 'A'
		cIdx2 = 'A'

		headerIdx    = -1
		headerColIdx = 'A'

		targetSynthesisSeqIdx    = -1
		targetSynthesisSeqColIdx = 'A'

		mergedMap = make(map[string]bool)
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
				case "合成序列":
					targetSynthesisSeqIdx = cIdx
				case "靶标序列":
					headerIdx = cIdx
				}
			}
			if fq1Idx == -1 || fq2Idx == -1 {
				log.Fatalf("not found fq1Idx=%d fq2Idx=%d", fq1Idx, fq2Idx)
			}
			cIdx1 += rune(fq1Idx)
			cIdx2 += rune(fq2Idx)
			headerColIdx += rune(headerIdx)
			targetSynthesisSeqColIdx += rune(targetSynthesisSeqIdx)
			continue
		}
		//slog.Info("row", slog.Int("i", i), slog.Any("row", row))
		if len(row) < fq1Idx {
			slog.Info("skip	 row", slog.Int("i", i), slog.Any("row", row))
			continue
		}
		fq1 := row[fq1Idx]
		merged := strings.Replace(fq1, "1.fq.gz", "merged.fq.gz", -1)
		mergedMap[merged] = true
		if *mergedDir != "" {
			merged = filepath.Join(*mergedDir, filepath.Base(merged))
		}
		simpleUtil.CheckErr(f.SetCellStr(*sheet, string(cIdx1)+strconv.Itoa(i+1), merged))
		simpleUtil.CheckErr(f.SetCellStr(*sheet, string(cIdx2)+strconv.Itoa(i+1), ""))
		if *head > 0 {
			targetSynthesisSeq := row[targetSynthesisSeqIdx]
			headerSeq := row[headerIdx] + targetSynthesisSeq[:*head]
			targetSynthesisSeq = targetSynthesisSeq[*head:]
			simpleUtil.CheckErr(f.SetCellStr(*sheet, string(headerColIdx)+strconv.Itoa(i+1), headerSeq))
			simpleUtil.CheckErr(f.SetCellStr(*sheet, string(targetSynthesisSeqColIdx)+strconv.Itoa(i+1), targetSynthesisSeq))
		}
	}

	// 保存文件到新的路径
	if err := f.SaveAs(*output); err != nil {
		fmt.Println(err)
	}

	var outDir = filepath.Dir(*output)
	for merged := range mergedMap {
		merged = filepath.Join(outDir, strings.Replace(merged, "_merged.fq.gz", "", -1))
		// 检测系统环境
		if os.Getenv("OS") == "Windows_NT" {
			fmt.Printf("CMD:\n  bash -c '/mnt/d/jrqlx/Documents/中合/测序分析/NGmerge.sh %s'\n", merged)
			if *run {
				// E:\github.com\NGmerge\NGmerge.exe -a -1 merged_1.fq.gz -2 .\raw\Y24-240107_L6_2.fq.gz -o test.win.cutAdapter.fq.gz -n 14
				cmd1 := exec.Command(
					"NGmerge.exe",
					"-a",
					"-1", merged+"_1.fq.gz",
					"-2", merged+"_2.fq.gz",
					"-o", merged+"_cutAdapter",
					"-n", "14",
				)
				cmd1.Stderr = os.Stderr
				cmd1.Stdout = os.Stdout
				log.Println(cmd1)
				simpleUtil.CheckErr(cmd1.Run())
				// E:\github.com\NGmerge\NGmerge.exe -1 .\test.win.cutAdapter.fq.gz_1.fastq.gz -2 .\test.win.cutAdapter.fq.gz_1.fastq.gz -o test.win.merged.fq.gz -n 14 -m 10
				mergedFq := merged + "_merged.fq.gz"
				if *mergedDir != "" {
					mergedFq = filepath.Join(*mergedDir, filepath.Base(mergedFq))
				}
				cmd2 := exec.Command(
					"NGmerge.exe",
					"-1", merged+"_cutAdapter_1.fastq.gz",
					"-2", merged+"_cutAdapter_2.fastq.gz",
					"-o", mergedFq,
					"-n", "14",
					"-m", "10",
				)
				cmd2.Stderr = os.Stderr
				cmd2.Stdout = os.Stdout
				log.Println(cmd2)
				simpleUtil.CheckErr(cmd2.Run())
				// remove file
				simpleUtil.CheckErr(os.Remove(merged + "_cutAdapter_1.fastq.gz"))
				simpleUtil.CheckErr(os.Remove(merged + "_cutAdapter_2.fastq.gz"))
			}
		} else {
			fmt.Printf("CMD:\n\tNGmerge.sh %s\n", merged)
			if *run {
				cmd := exec.Command("NGmerge.sh", merged)
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				log.Println(cmd)
				simpleUtil.CheckErr(cmd.Run())
			}
		}
	}
}
