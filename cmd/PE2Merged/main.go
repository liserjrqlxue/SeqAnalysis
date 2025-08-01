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
	"sync"

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
	rawDir = flag.String(
		"raw",
		"",
		"raw fq dir",
	)
	thread = flag.Int(
		"t",
		8,
		"max thread",
	)
)
var (
	MaxThread = 128
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
		if len(row) <= fq1Idx {
			slog.Info("skip	 row", slog.Int("i", i), slog.Any("row", row))
			continue
		}
		fq1 := row[fq1Idx]
		merged := strings.ReplaceAll(fq1, "1.fq.gz", "merged.fq.gz")
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

	if *run {
		simpleUtil.CheckErr(RunNGmerge(mergedMap, *thread))
	}
}

func RunNGmerge(mergedMap map[string]bool, maxConcurrent int) error {
	// 创建带缓冲的通道用于控制并发数
	var (
		wg sync.WaitGroup
		mu sync.Mutex

		sem    = make(chan struct{}, maxConcurrent)
		errCh  = make(chan error, maxConcurrent)
		doneCh = make(chan struct{})

		errs []error

		NGmerge = "NGmerge"
	)
	// 检测系统环境
	if os.Getenv("OS") == "Windows_NT" {
		NGmerge = "NGmerge.exe"
	}

	// 错误收集协程
	go func() {
		for err := range errCh {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
		close(doneCh)
	}()

	for merged := range mergedMap {
		prefix := strings.ReplaceAll(merged, "_merged.fq.gz", "")
		if *rawDir != "" {
			prefix = filepath.Join(*rawDir, prefix)
		} else {
			prefix = filepath.Join(filepath.Dir(*output), prefix)
		}
		wg.Add(1)
		go func(prefix string) { // 使用闭包捕获当前merged值
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			// 错误处理函数
			handleError := func(err error, operation string) {
				if err != nil {
					slog.Error("NGmerge", "operation", operation, "err", err, "prefix", prefix)
					errCh <- fmt.Errorf("%s failed on %s: %w", operation, filepath.Base(prefix), err)
				}
			}

			var (
				fq1 = prefix + "_1.fq.gz"
				fq2 = prefix + "_2.fq.gz"
			)

			// 第一步命令：切除接头
			cmd1 := exec.Command(
				NGmerge,
				"-a",
				"-1", fq1, "-2", fq2,
				"-o", prefix+"_cutAdapter",
				"-n", "14",
			)
			cmd1.Stderr = os.Stderr
			cmd1.Stdout = os.Stdout
			log.Println(cmd1)
			if err := cmd1.Run(); err != nil {
				handleError(err, "Adapter trimming")
				return
			}

			// 第二步命令：合并序列
			mergedFq := prefix + "_merged.fq.gz"
			if *mergedDir != "" {
				mergedFq = filepath.Join(*mergedDir, filepath.Base(mergedFq))
			}
			// 创建输出目录
			if err := os.MkdirAll(filepath.Dir(mergedFq), 0755); err != nil {
				handleError(err, "Create output directory")
				return
			}
			cmd2 := exec.Command(
				NGmerge,
				"-1", prefix+"_cutAdapter_1.fastq.gz",
				"-2", prefix+"_cutAdapter_2.fastq.gz",
				"-o", mergedFq,
				"-n", "14",
				"-m", "10",
			)
			cmd2.Stderr = os.Stderr
			cmd2.Stdout = os.Stdout
			log.Println(cmd2)
			if err := cmd2.Run(); err != nil {
				handleError(err, "Read merging")
				return
			}

			// 清理临时文件
			if err := os.Remove(prefix + "_cutAdapter_1.fastq.gz"); err != nil {
				handleError(err, "Remove temporary file")
			}
			if err := os.Remove(prefix + "_cutAdapter_2.fastq.gz"); err != nil {
				handleError(err, "Remove temporary file")
			}
		}(prefix) // 传递当前prefix值到闭包
	}

	// 等待所有任务完成
	wg.Wait()
	close(errCh) // 关闭错误通道，触发收集协程退出
	<-doneCh     // 等待错误收集协程完成

	// 返回遇到的第一个错误
	if len(errs) > 0 {
		return fmt.Errorf("processing completed with %d errors. First error: %w", len(errs), errs[0])
	}
	return nil
}
