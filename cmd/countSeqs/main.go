package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	// "compress/gzip"
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

func countSequencesInFastq(sequences []Sequence, fastqFile string) (int, error) {
	// 打开文件
	file, err := os.Open(fastqFile)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var scanner *bufio.Scanner

	// 检查是否为gzip压缩文件
	if strings.HasSuffix(fastqFile, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return 0, fmt.Errorf("无法解压gzip文件: %v", err)
		}
		defer gzReader.Close()
		scanner = bufio.NewScanner(gzReader)
	} else {
		scanner = bufio.NewScanner(file)
	}

	lineCount := 0
	readCount := 0

	// 预编译序列到map，提高查找效率
	seqMap := make(map[string]*Sequence)
	for i := range sequences {
		seqMap[sequences[i].Seq] = &sequences[i]
	}

	fmt.Printf("开始处理FASTQ文件: %s\n", fastqFile)
	startTime := time.Now()

	// 设置缓冲区大小以提高大文件读取性能
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// FASTQ格式：第2行是序列行
		if lineCount%4 == 2 {
			readCount++
			// 转换为大写确保匹配
			seqLine := strings.ToUpper(line)

			// 检查每个目标序列
			for targetSeq, seqStruct := range seqMap {
				if strings.Contains(seqLine, targetSeq) || strings.Contains(seqLine, seqStruct.RcSeq) {
					seqStruct.Count++
					break
				}
			}

			// 进度显示
			if readCount%100000 == 0 {
				elapsed := time.Since(startTime)
				fmt.Printf("  %s: 已处理 %d 条reads, 用时: %v\n", fastqFile, readCount, elapsed)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return readCount, fmt.Errorf("读取文件时出错: %v", err)
	}

	totalTime := time.Since(startTime)
	fmt.Printf("处理完成! %s: 总共处理 %d 条reads, 用时: %v\n", fastqFile, readCount, totalTime)
	return readCount, nil
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

	// 处理所有FASTQ文件
	var processedFiles []FastqFile
	totalStartTime := time.Now()

	for _, fastqFile := range fastqFiles {
		readCount, err := countSequencesInFastq(sequences, fastqFile)
		if err != nil {
			fmt.Printf("警告: 处理文件 %s 时出错: %v\n", fastqFile, err)
			continue
		}
		processedFiles = append(processedFiles, FastqFile{
			Filename:  fastqFile,
			ReadCount: readCount,
		})
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
