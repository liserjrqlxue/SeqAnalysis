package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// 解析命令行参数
	var dirPath string
	flag.StringVar(&dirPath, "d", ".", "指定要扫描的目录")
	flag.Parse()

	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		fmt.Printf("错误: 目录 '%s' 不存在\n", dirPath)
		os.Exit(1)
	}

	// 查找所有匹配的文件
	files, err := findPathFiles(dirPath)
	if err != nil {
		fmt.Printf("查找文件时出错: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("在目录 '%s' 中未找到匹配 *[Pp]ath.txt 的文件\n", dirPath)
		return
	}

	fmt.Printf("在目录 '%s' 中找到 %d 个匹配的文件\n", dirPath, len(files))

	// 处理每个文件
	for _, file := range files {
		if err := processFile(file); err != nil {
			fmt.Printf("处理文件 '%s' 时出错: %v\n", file, err)
		}
	}
}

// 查找所有匹配 *[Pp]ath.txt 的文件
func findPathFiles(dirPath string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件，跳过目录
		if d.IsDir() {
			return nil
		}

		// 获取文件名
		filename := filepath.Base(path)

		// 检查文件名是否匹配 *[Pp]ath.txt
		if strings.HasSuffix(strings.ToLower(filename), "ath.txt") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// 处理单个文件
func processFile(filePath string) error {
	fmt.Printf("处理文件: %s\n", filePath)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		fmt.Printf("  第 %d 行: %s\n", lineNum, line)

		// 处理该行内容
		if err := processLine(filePath, line); err != nil {
			fmt.Printf("  处理行时出错: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件时出错: %w", err)
	}

	return nil
}

// 处理单行内容
func processLine(filePath, line string) error {
	// 获取文件所在目录
	dir := filepath.Dir(filePath)

	// 根据内容类型决定执行哪个脚本
	if strings.HasPrefix(line, "oss://") {
		// 处理OSS URL
		return processOssURL(dir, line)
	} else {
		// 处理芯片号
		return processChipCode(dir, line)
	}
}

// 处理OSS URL
func processOssURL(dir, url string) error {
	// 提取最后一个'/'之后的部分
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return fmt.Errorf("无效的OSS URL: %s", url)
	}

	param := parts[len(parts)-1]
	if param == "" {
		return fmt.Errorf("OSS URL 中没有找到有效参数: %s", url)
	}

	fmt.Printf("    提取参数: %s\n", param)

	// 构建命令
	scriptPath := filepath.Join("..", "runNovo.sh")
	cmd := exec.Command("bash", scriptPath, param)
	cmd.Dir = dir

	// 执行命令
	return runCommand(cmd)
}

// 处理芯片号
func processChipCode(dir, chipCode string) error {
	fmt.Printf("    芯片号: %s\n", chipCode)

	// 构建命令
	scriptPath := filepath.Join("..", "runG99.sh")
	cmd := exec.Command("bash", scriptPath, chipCode)
	cmd.Dir = dir

	// 执行命令
	return runCommand(cmd)
}

// 执行命令并显示输出
func runCommand(cmd *exec.Cmd) error {
	fmt.Printf("    执行命令: %s\n", strings.Join(cmd.Args, " "))

	// 捕获命令输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("命令执行失败: %w", err)
	}

	fmt.Println("    命令执行成功")
	return nil
}
