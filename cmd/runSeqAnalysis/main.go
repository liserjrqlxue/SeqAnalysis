package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SeqAnalysis/pkg/wechatwork" // 替换为你的模块名

	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

const (
	maxConcurrent = 8 // 最大并发数
)

type FileResult struct {
	FileName string
	Success  bool
	Error    string
}

func main() {
	// 解析命令行参数
	var dirPath string
	var batch string
	var webhookKey string
	flag.StringVar(&dirPath, "d", ".", "指定要处理的目录")
	flag.StringVar(&batch, "batch", "", "批次名称（如果不提供，将从Path.txt文件中解析）")
	flag.StringVar(&webhookKey, "webhook", "", "企业微信Webhook Key（可选）")
	flag.Parse()

	// 初始化企业微信通知
	notifier := wechatwork.NewNotificationSender(webhookKey)

	// 记录开始时间
	startTime := time.Now()

	// 发送开始通知
	if err := sendStartNotification(notifier, simpleUtil.HandleError(filepath.Abs(dirPath))); err != nil {
		fmt.Printf("发送开始通知失败: %v\n", err)
	}

	// 切换到指定目录
	if dirPath != "." {
		if err := os.Chdir(dirPath); err != nil {
			msg := fmt.Sprintf("切换到目录 '%s' 失败: %v", dirPath, err)
			fmt.Println(msg)
			sendErrorNotification(notifier, msg)
			os.Exit(1)
		}
		fmt.Printf("已切换到目录: %s\n", dirPath)
	}

	// 获取当前工作目录（切换后）
	currentDir, err := os.Getwd()
	if err != nil {
		msg := fmt.Sprintf("获取当前目录失败: %v", err)
		fmt.Println(msg)
		sendErrorNotification(notifier, msg)
		os.Exit(1)
	}

	// 如果batch未指定，尝试从Path.txt文件中解析
	if batch == "" {
		fmt.Println("batch参数未指定，尝试从Path.txt文件中解析...")
		var err error
		batch, err = extractBatchFromPathFiles(currentDir)
		if err != nil {
			msg := fmt.Sprintf("从Path.txt文件中解析batch失败: %v", err)
			fmt.Println(msg)
			sendErrorNotification(notifier, msg)
			fmt.Println("请使用 -batch 参数指定批次名称")
			os.Exit(1)
		}
		fmt.Printf("从Path.txt文件中解析到的batch: %s\n", batch)
	}

	// 获取目录名
	dirName := filepath.Base(currentDir)

	// 构建路径
	rawDataPath := fmt.Sprintf("/data2/wangyaoshen/novo-medical-customer-tj/CYB24030020/%s/Rawdata", batch)
	seqAnalysisPath := "/data2/wangyaoshen/src/SeqAnalysis/cmd/SeqAnalysis/SeqAnalysis"

	// 查找所有非merged的.xlsx文件
	xlsxFiles, err := findXLSXFiles(currentDir)
	if err != nil {
		msg := fmt.Sprintf("查找.xlsx文件失败: %v", err)
		fmt.Println(msg)
		sendErrorNotification(notifier, msg)
		os.Exit(1)
	}

	if len(xlsxFiles) == 0 {
		msg := "未找到.xlsx文件"
		fmt.Println(msg)
		sendWarningNotification(notifier, msg)
		return
	}

	fmt.Printf("找到 %d 个.xlsx文件\n", len(xlsxFiles))

	// 使用工作池处理文件
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrent)
	results := make(chan FileResult, len(xlsxFiles))

	for _, xlsxFile := range xlsxFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			// 获取信号量，控制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := processXLSXFile(file, rawDataPath, seqAnalysisPath, dirName)
			if err != nil {
				results <- FileResult{
					FileName: file,
					Success:  false,
					Error:    err.Error(),
				}
			} else {
				results <- FileResult{
					FileName: file,
					Success:  true,
				}
			}
		}(xlsxFile)
	}

	// 等待所有任务完成
	wg.Wait()
	close(results)

	// 收集结果
	var fileResults []FileResult
	var successCount, failCount int
	var failedFiles []string

	for result := range results {
		fileResults = append(fileResults, result)
		if result.Success {
			successCount++
		} else {
			failCount++
			failedFiles = append(failedFiles, fmt.Sprintf("%s: %s", result.FileName, result.Error))
		}
	}

	// 计算处理时间
	duration := time.Since(startTime)

	// 发送完成通知
	sendCompletionNotification(notifier, currentDir, batch, len(xlsxFiles), successCount, failCount, failedFiles, duration)

	// 输出总结
	fmt.Printf("\n=== 处理完成 ===\n")
	fmt.Printf("总文件数: %d\n", len(xlsxFiles))
	fmt.Printf("成功: %d\n", successCount)
	fmt.Printf("失败: %d\n", failCount)
	fmt.Printf("耗时: %v\n", duration)

	if failCount > 0 {
		fmt.Println("\n失败的文件:")
		for _, failedFile := range failedFiles {
			fmt.Printf("  - %s\n", failedFile)
		}
		fmt.Println("\n部分文件处理失败，请检查日志")
		os.Exit(1)
	}

	fmt.Println("\n所有文件处理完成")
}

// 发送开始通知
func sendStartNotification(notifier *wechatwork.NotificationSender, dirPath string) error {
	content := fmt.Sprintf("### 🔄 开始处理数据\n"+
		"**目录**: %s\n"+
		"**开始时间**: %s\n"+
		"---\n"+
		"正在开始处理...",
		dirPath,
		time.Now().Format("2006-01-02 15:04:05"))

	return notifier.SendMarkdown(content)
}

// 发送错误通知
func sendErrorNotification(notifier *wechatwork.NotificationSender, errorMsg string) {
	content := fmt.Sprintf("### ❌ 处理失败\n"+
		"**错误信息**: %s\n"+
		"**时间**: %s\n"+
		"---\n"+
		"请立即检查！",
		errorMsg,
		time.Now().Format("2006-01-02 15:04:05"))

	// @所有人
	notifier.SendText(content, []string{"@all"}, nil)
}

// 发送警告通知
func sendWarningNotification(notifier *wechatwork.NotificationSender, warningMsg string) {
	content := fmt.Sprintf("### ⚠️ 处理警告\n"+
		"**警告信息**: %s\n"+
		"**时间**: %s",
		warningMsg,
		time.Now().Format("2006-01-02 15:04:05"))

	notifier.SendMarkdown(content)
}

// 发送完成通知
func sendCompletionNotification(notifier *wechatwork.NotificationSender, currentDir, batch string, total, success, fail int, failedFiles []string, duration time.Duration) {
	var statusIcon string
	var statusText string

	if fail == 0 {
		statusIcon = "✅"
		statusText = "全部成功"
	} else if success == 0 {
		statusIcon = "❌"
		statusText = "全部失败"
	} else {
		statusIcon = "⚠️"
		statusText = "部分失败"
	}

	// 构建Markdown内容
	content := fmt.Sprintf("### %s 数据处理完成\n"+
		"**批次**: %s\n"+
		"**目录**: %s\n"+
		"**状态**: %s\n"+
		"**总文件数**: %d\n"+
		"**成功**: %d\n"+
		"**失败**: %d\n"+
		"**耗时**: %v\n"+
		"**完成时间**: %s\n",
		statusIcon,
		batch,
		currentDir,
		statusText,
		total,
		success,
		fail,
		duration,
		time.Now().Format("2006-01-02 15:04:05"))

	// 如果有失败的文件，添加到消息中
	if fail > 0 {
		content += "\n**失败文件**:\n"
		// 最多显示5个失败文件，避免消息过长
		maxShow := 5
		if len(failedFiles) < maxShow {
			maxShow = len(failedFiles)
		}
		for i := 0; i < maxShow; i++ {
			content += fmt.Sprintf("- %s\n", failedFiles[i])
		}
		if len(failedFiles) > maxShow {
			content += fmt.Sprintf("- ... 还有%d个失败文件\n", len(failedFiles)-maxShow)
		}
	}

	content += "\n---\n"

	// 根据结果决定是否@所有人
	if fail > 0 {
		// 如果有失败，@所有人提醒
		notifier.SendText(fmt.Sprintf("数据处理完成，有%d个文件失败，请检查！", fail), []string{"@all"}, nil)
	} else {
		// 全部成功，只发Markdown消息
		notifier.SendMarkdown(content)
	}
}

// 从Path.txt文件中提取batch
func extractBatchFromPathFiles(dir string) (string, error) {
	// 查找所有匹配的文件
	var pathFiles []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 检查文件名是否匹配 *[Pp]ath.txt
		lowerName := strings.ToLower(name)
		if strings.HasSuffix(lowerName, "ath.txt") {
			pathFiles = append(pathFiles, name)
		}
	}

	if len(pathFiles) == 0 {
		return "", fmt.Errorf("未找到*[Pp]ath.txt文件")
	}

	if len(pathFiles) > 1 {
		return "", fmt.Errorf("找到多个*[Pp]ath.txt文件: %v", pathFiles)
	}

	// 读取文件
	filePath := pathFiles[0]
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件 %s: %w", filePath, err)
	}
	defer file.Close()

	// 读取文件内容
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" { // 只处理非空行
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("文件 %s 为空或只包含空行", filePath)
	}

	if len(lines) > 1 {
		return "", fmt.Errorf("文件 %s 包含多行有效内容，无法确定batch", filePath)
	}

	// 解析第一行
	firstLine := lines[0]

	// 如果是oss://开头的URL，提取最后一部分
	if strings.HasPrefix(firstLine, "oss://") {
		parts := strings.Split(firstLine, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// 否则直接返回第一行
	return firstLine, nil
}

// 查找所有非merged的.xlsx文件
func findXLSXFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 检查是否为.xlsx文件且不包含merged
		if strings.HasSuffix(strings.ToLower(name), ".xlsx") &&
			!strings.Contains(strings.ToLower(name), "merged") {
			files = append(files, name)
		}
	}

	return files, nil
}

// 处理单个.xlsx文件
func processXLSXFile(xlsxFile, rawDataPath, seqAnalysisPath, dirName string) error {
	fileName := strings.TrimSuffix(xlsxFile, ".xlsx")

	fmt.Printf("开始处理文件: %s\n", xlsxFile)

	// 第一步: 运行PE2Merged类似处理
	fmt.Printf("  第一步: 处理 %s\n", xlsxFile)
	if err := runPE2Merged(xlsxFile, rawDataPath); err != nil {
		return fmt.Errorf("PE2Merged处理失败: %v", err)
	}

	// 检查是否生成了.merged.xlsx文件
	mergedFile := fileName + ".merged.xlsx"
	if _, err := os.Stat(mergedFile); os.IsNotExist(err) {
		// 如果没有生成，尝试其他可能的命名
		mergedFile = strings.Replace(xlsxFile, ".xlsx", ".merged.xlsx", 1)
		if _, err := os.Stat(mergedFile); os.IsNotExist(err) {
			return fmt.Errorf("未找到合并后的文件: %s", mergedFile)
		}
	}

	// 第二步: 运行SeqAnalysis
	fmt.Printf("  第二步: 分析 %s\n", mergedFile)
	if err := runSeqAnalysis(mergedFile, seqAnalysisPath, dirName); err != nil {
		return fmt.Errorf("SeqAnalysis处理失败: %v", err)
	}

	fmt.Printf("文件 %s 处理完成\n", xlsxFile)
	return nil
}

// 运行PE2Merged处理
func runPE2Merged(xlsxFile, rawDataPath string) error {
	// 构建命令
	cmd := exec.Command("PE2Merged",
		"-fastp",
		"-raw", rawDataPath,
		"-d", ".",
		"-run",
		"-i", xlsxFile,
		"-m", "5",
	)
	cmd.Stderr = os.Stderr

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("命令执行失败: %v\n输出: %s", err, string(output))
	}

	fmt.Printf("    PE2Merged输出: %s\n", string(output))
	return nil
}

// 运行SeqAnalysis处理
func runSeqAnalysis(mergedFile, seqAnalysisPath, dirName string) error {
	// 获取基本文件名（不带扩展名）
	baseName := strings.TrimSuffix(mergedFile, ".xlsx")

	// 构建命令
	cmd := exec.Command(seqAnalysisPath,
		"--lessMem",
		"-plot",
		"-rc",
		"-zip",
		"-i", mergedFile,
		"-o", fmt.Sprintf("%s.%s", dirName, baseName),
	)
	cmd.Stderr = os.Stderr

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("命令执行失败: %v\n输出: %s", err, string(output))
	}

	fmt.Printf("    SeqAnalysis输出: %s\n", string(output))
	return nil
}
