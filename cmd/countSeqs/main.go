package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	// "compress/gzip"
	"github.com/cloudflare/ahocorasick"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/DNA/pkg/util"
)

type Sequence struct {
	Name  string
	Seq   string
	RcSeq string
	Count int
}

type FastqFile struct {
	Filename  string
	ReadCount int
	CountMap  []int
}

func readSequences(filename string) ([]Sequence, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sequences []Sequence
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			fmt.Printf("警告: 第%d行格式错误，跳过\n", lineNum)
			continue
		}

		// 转换为大写确保一致性
		seq := Sequence{
			Name:  fields[0],
			Seq:   strings.ToUpper(fields[1]),
			Count: 0,
		}
		seq.RcSeq = util.ReverseComplement(seq.Seq)
		sequences = append(sequences, seq)
	}

	return sequences, scanner.Err()
}

func processFastqFile(sequences []Sequence, fastqFile string, wg *sync.WaitGroup, results chan<- FastqFile) {
	defer wg.Done()

	// 打开文件
	file, err := os.Open(fastqFile)
	if err != nil {
		fmt.Printf("错误: 无法打开文件 %s: %v\n", fastqFile, err)
		return
	}
	defer file.Close()

	var scanner *bufio.Scanner

	// 检查是否为gzip压缩文件
	if strings.HasSuffix(fastqFile, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			fmt.Printf("错误: 无法解压gzip文件 %s: %v\n", fastqFile, err)
			return
		}
		defer gzReader.Close()
		scanner = bufio.NewScanner(gzReader)
	} else {
		scanner = bufio.NewScanner(file)
	}

	// 准备所有模式：原始序列和它们的反向互补序列
	patterns := make([]string, 0, len(sequences)*2)
	patternToSeqIdx := make(map[string]int) // 映射模式到序列索引

	for i, seq := range sequences {
		// 添加原始序列
		patterns = append(patterns, seq.Seq)
		patternToSeqIdx[seq.Seq] = i

		// 添加反向互补序列
		patterns = append(patterns, seq.RcSeq)
		patternToSeqIdx[seq.RcSeq] = i
	}

	// 构建Aho-Corasick自动机
	matcher := ahocorasick.NewStringMatcher(patterns)

	// 创建计数映射
	countMap := make([]int, len(sequences))

	fmt.Printf("开始处理FASTQ文件: %s\n", fastqFile)
	startTime := time.Now()

	const maxCapacity = 4 * 1024 * 1024 // 4MB缓冲区
	const batchSize = 5000              // 更大的批次
	const numWorkers = 16               // 更多worker

	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineCount := 0
	readCount := 0

	// 使用通道和worker池并行处理reads
	batches := make(chan []string, 20)
	batchResults := make(chan []int, 20)

	// 启动worker
	var workerWg sync.WaitGroup
	for i := range numWorkers {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			processReadsBatch(i, matcher, patterns, patternToSeqIdx, batches, batchResults)
		}()
	}

	// 启动结果收集器
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for batchResult := range batchResults {
			for i, count := range batchResult {
				countMap[i] += count
			}
		}
	}()

	// 读取文件并分批次发送
	var currentBatch []string
	batchCounter := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// FASTQ格式：第2行是序列行
		if lineCount%4 == 2 {
			readCount++

			// currentBatch = append(currentBatch, strings.ToUpper(line))
			currentBatch = append(currentBatch, line)

			// 当批次达到大小时发送处理
			if len(currentBatch) >= batchSize {
				batches <- currentBatch
				currentBatch = nil

				batchCounter++

				// 减少进度输出频率，减少I/O
				if batchCounter%200 == 0 {
					elapsed := time.Since(startTime)
					fmt.Printf("  %s: 已处理 %d 条reads, 用时: %v\n", fastqFile, readCount, elapsed)
				}
			}

		}
	}

	// 处理剩余的批次
	if len(currentBatch) > 0 {
		batches <- currentBatch
	}

	// 关闭通道并等待worker完成
	close(batches)
	workerWg.Wait()
	close(batchResults)
	collectorWg.Wait()

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取文件 %s 时出错: %v\n", fastqFile, err)
	}

	totalTime := time.Since(startTime)
	fmt.Printf("处理完成! %s: 总共处理 %d 条reads, 用时: %v\n", fastqFile, readCount, totalTime)

	// 发送结果，包含文件信息和计数
	results <- FastqFile{
		Filename:  fastqFile,
		ReadCount: readCount,
		CountMap:  countMap,
	}
}

// 处理reads批次的worker函数
func processReadsBatch(i int, matcher *ahocorasick.Matcher, patterns []string, patternToSeqIdx map[string]int,
	batches <-chan []string, results chan<- []int) {
	total := 0
	hit := 0

	for batch := range batches {
		batchCounts := make([]int, len(patternToSeqIdx)/2) // 除以2因为每个序列有两个模式

		// 预编译字节切片，避免重复转换
		batchBytes := make([][]byte, len(batch))
		for i, read := range batch {
			batchBytes[i] = []byte(read)
		}

		for _, readBytes := range batchBytes {
			total++
			matches := matcher.Match(readBytes)
			if len(matches) > 0 {
				hit++
				// 取第一个匹配的模式索引
				patternIdx := matches[0]
				// 通过模式索引找到对应的序列
				pattern := patterns[patternIdx]
				// 通过模式找到序列索引
				seqIdx := patternToSeqIdx[pattern]
				batchCounts[seqIdx]++
			}
		}

		results <- batchCounts
	}
	// 调试输出
	fmt.Printf("ProcessReadsBatch %2d, Total %d, Hit %d\n", i+1, total, hit)
}

// 并行处理多个FASTQ文件
func countSequencesInFastqFiles(sequences []Sequence, fastqFiles []string) ([]FastqFile, error) {
	var wg sync.WaitGroup
	// 修改通道类型为FileResult
	results := make(chan FastqFile, len(fastqFiles))
	processedFiles := make([]FastqFile, 0, len(fastqFiles))

	// 为每个文件启动一个处理goroutine
	for _, fastqFile := range fastqFiles {
		wg.Add(1)
		go processFastqFile(sequences, fastqFile, &wg, results)
	}

	// 等待所有文件处理完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	fileCount := 0
	for result := range results {
		// 将每个文件的结果累加到sequences中
		for i, count := range result.CountMap {
			sequences[i].Count += count
		}
		fileCount++

		// 这里我们简化处理，实际应该记录每个文件的reads数
		processedFiles = append(processedFiles, result)
	}

	return processedFiles, nil
}

func writeResults(sequences []Sequence, outputFile string, fastqFiles []FastqFile) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	if _, err := writer.WriteString("Name\tSequence\tReadCount\n"); err != nil {
		return err
	}

	// 写入数据
	totalMatches := 0
	for _, seq := range sequences {
		line := fmt.Sprintf("%s\t%s\t%d\n", seq.Name, seq.Seq, seq.Count)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
		totalMatches += seq.Count
	}

	// 写入处理摘要
	if _, err := writer.WriteString("\n# Processing Summary\n"); err != nil {
		return err
	}
	if _, err := writer.WriteString("# =================\n"); err != nil {
		return err
	}

	totalReads := 0
	for _, fq := range fastqFiles {
		summary := fmt.Sprintf("# %s: %d reads\n", fq.Filename, fq.ReadCount)
		if _, err := writer.WriteString(summary); err != nil {
			return err
		}
		totalReads += fq.ReadCount
	}

	totalSummary := fmt.Sprintf("# Total FASTQ files processed: %d\n", len(fastqFiles))
	if _, err := writer.WriteString(totalSummary); err != nil {
		return err
	}

	totalSummary = fmt.Sprintf("# Total reads processed: %d\n", totalReads)
	if _, err := writer.WriteString(totalSummary); err != nil {
		return err
	}

	totalSummary = fmt.Sprintf("# Total matches found: %d\n", totalMatches)
	if _, err := writer.WriteString(totalSummary); err != nil {
		return err
	}

	fmt.Printf("总计处理 %d 个FASTQ文件, %d 条reads, %d 次匹配\n",
		len(fastqFiles), totalReads, totalMatches)
	return nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("用法: ./sequence_counter <序列文件> <输出文件> <fastq文件1> [fastq文件2 ...]")
		fmt.Println("示例: ./sequence_counter sequences.txt results.txt sample1.fastq sample2.fastq")
		fmt.Println("示例: ./sequence_counter sequences.txt results.txt *.fastq")
		os.Exit(1)
	}

	sequenceFile := os.Args[1]
	outputFile := os.Args[2]
	fastqFiles := os.Args[3:]

	// 读取目标序列
	fmt.Printf("读取序列文件: %s\n", sequenceFile)
	sequences, err := readSequences(sequenceFile)
	if err != nil {
		fmt.Printf("错误: 无法读取序列文件: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("成功读取 %d 个目标序列\n", len(sequences))

	// 并行处理所有FASTQ文件
	fmt.Printf("开始并行处理 %d 个FASTQ文件\n", len(fastqFiles))
	totalStartTime := time.Now()

	processedFiles, err := countSequencesInFastqFiles(sequences, fastqFiles)
	if err != nil {
		fmt.Printf("错误: 处理FASTQ文件时出错: %v\n", err)
		os.Exit(1)
	}

	totalTime := time.Since(totalStartTime)
	fmt.Printf("\n所有FASTQ文件处理完成! 总用时: %v\n", totalTime)

	// 输出结果
	err = writeResults(sequences, outputFile, processedFiles)
	if err != nil {
		fmt.Printf("错误: 无法写入结果文件: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("结果已保存到: %s\n", outputFile)

	// 显示汇总统计
	fmt.Println("\n处理汇总:")
	fmt.Printf("  处理的FASTQ文件数: %d\n", len(processedFiles))
	fmt.Printf("  目标序列数: %d\n", len(sequences))

	// 显示匹配最多的前10个序列
	fmt.Println("\n匹配最多的前10个序列:")
	fmt.Println("Name\tSequence\tReadCount")
	count := 0
	for i := range sequences {
		if count >= 10 {
			break
		}
		// 简单排序：找到计数最大的
		maxIndex := i
		for j := i + 1; j < len(sequences); j++ {
			if sequences[j].Count > sequences[maxIndex].Count {
				maxIndex = j
			}
		}
		if maxIndex != i {
			sequences[i], sequences[maxIndex] = sequences[maxIndex], sequences[i]
		}
		fmt.Printf("%s\t%s\t%d\n", sequences[i].Name, sequences[i].Seq, sequences[i].Count)
		count++
	}
}
