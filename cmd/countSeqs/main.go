package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Sequence struct {
	Name  string
	Seq   string
	Count int
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
		sequences = append(sequences, seq)
	}

	return sequences, scanner.Err()
}

func countSequencesInFastq(sequences []Sequence, fastqFile string) error {
	file, err := os.Open(fastqFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	readCount := 0

	// 预编译序列到map，提高查找效率
	seqMap := make(map[string]*Sequence)
	for i := range sequences {
		seqMap[sequences[i].Seq] = &sequences[i]
	}

	fmt.Printf("开始处理FASTQ文件，共有 %d 个目标序列\n", len(sequences))
	startTime := time.Now()

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
				if strings.Contains(seqLine, targetSeq) {
					seqStruct.Count++
				}
			}

			// 进度显示
			if readCount%100000 == 0 {
				elapsed := time.Since(startTime)
				fmt.Printf("已处理 %d 条reads, 用时: %v\n", readCount, elapsed)
			}
		}
	}

	totalTime := time.Since(startTime)
	fmt.Printf("处理完成! 总共处理 %d 条reads, 总用时: %v\n", readCount, totalTime)
	return scanner.Err()
}

func writeResults(sequences []Sequence, outputFile string) error {
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

	// 写入数据并按计数排序
	totalMatches := 0
	for _, seq := range sequences {
		line := fmt.Sprintf("%s\t%s\t%d\n", seq.Name, seq.Seq, seq.Count)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
		totalMatches += seq.Count
	}

	fmt.Printf("总计匹配次数: %d\n", totalMatches)
	return nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("用法: ./sequence_counter <序列文件> <fastq文件> <输出文件>")
		fmt.Println("示例: ./sequence_counter sequences.txt input.fastq results.txt")
		os.Exit(1)
	}

	sequenceFile := os.Args[1]
	fastqFile := os.Args[2]
	outputFile := os.Args[3]

	// 读取目标序列
	fmt.Printf("读取序列文件: %s\n", sequenceFile)
	sequences, err := readSequences(sequenceFile)
	if err != nil {
		fmt.Printf("错误: 无法读取序列文件: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("成功读取 %d 个目标序列\n", len(sequences))

	// 处理FASTQ文件
	err = countSequencesInFastq(sequences, fastqFile)
	if err != nil {
		fmt.Printf("错误: 无法处理FASTQ文件: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	err = writeResults(sequences, outputFile)
	if err != nil {
		fmt.Printf("错误: 无法写入结果文件: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("结果已保存到: %s\n", outputFile)
}
