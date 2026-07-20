package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fvbommel/sortorder"
	"github.com/xuri/excelize/v2"
)

func main() {
	var dir string
	var output string
	flag.StringVar(&dir, "d", "", "目录，包含批次文件夹（必需）")
	flag.StringVar(&output, "o", "", "输出 Excel 文件路径（默认：DIR/DIR.summary.xlsx）")
	flag.Parse()

	if dir == "" {
		fmt.Fprintln(os.Stderr, "错误：缺少必要参数 -d")
		flag.Usage()
		os.Exit(1)
	}

	// 确保目录存在
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法访问目录 %s: %v\n", dir, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "%s 不是目录\n", dir)
		os.Exit(1)
	}

	baseDir := filepath.Base(dir) // 用于匹配目录前缀
	if output == "" {
		output = filepath.Join(dir, baseDir+".summary.xlsx")
	}

	// 扫描所有批次目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取目录失败: %v\n", err)
		os.Exit(1)
	}

	batchFile := make(map[string]string) // batch -> 最新文件路径
	prefix := baseDir + "."
	suffix := ".merged"

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		// 提取 batch 名称
		batch := name[len(prefix) : len(name)-len(suffix)]
		if batch == "" {
			continue
		}

		// 在该批次目录中查找所有日期文件
		batchDir := filepath.Join(dir, name)
		pattern := fmt.Sprintf("summary-%s.%s.merged-*.xlsx", baseDir, batch)
		files, err := filepath.Glob(filepath.Join(batchDir, pattern))
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：无法扫描 %s: %v\n", batchDir, err)
			continue
		}
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "警告：批次 %s 未找到任何 summary 文件\n", batch)
			continue
		}

		// 选择日期最新的文件（YYYYMMDD 越大越新）
		var latestFile, latestDate string
		dateRe := regexp.MustCompile(`-(\d{8})\.xlsx$`)
		for _, f := range files {
			base := filepath.Base(f)
			matches := dateRe.FindStringSubmatch(base)
			if len(matches) != 2 {
				continue
			}
			dateStr := matches[1]
			if latestDate == "" || dateStr > latestDate {
				latestDate = dateStr
				latestFile = f
			}
		}
		if latestFile == "" {
			fmt.Fprintf(os.Stderr, "警告：批次 %s 无有效日期文件\n", batch)
			continue
		}
		batchFile[batch] = latestFile
	}

	if len(batchFile) == 0 {
		fmt.Fprintln(os.Stderr, "未找到任何批次")
		os.Exit(1)
	}

	// 按自然排序（sort -V）对 batch 排序
	batches := make([]string, 0, len(batchFile))
	for b := range batchFile {
		batches = append(batches, b)
	}
	sort.Slice(batches, func(i, j int) bool {
		return sortorder.NaturalLess(batches[i], batches[j])
	})

	// 创建输出 Excel 文件
	outFile := excelize.NewFile()
	defer outFile.Close()

	// 写入清单 Sheet1
	sheet1 := "Sheet1"
	outFile.SetCellValue(sheet1, "A1", "BatchName")
	outFile.SetCellValue(sheet1, "B1", "FileName")
	for i, batch := range batches {
		row := i + 2
		absPath, _ := filepath.Abs(batchFile[batch]) // 转为绝对路径
		outFile.SetCellValue(sheet1, fmt.Sprintf("A%d", row), batch)
		outFile.SetCellValue(sheet1, fmt.Sprintf("B%d", row), absPath)
	}

	// 处理每个批次的 Summary 工作表
	for _, batch := range batches {
		filePath := batchFile[batch]
		// 清理 sheet 名称（Excel 限制：≤31 字符，不含非法字符）
		sheetName := sanitizeSheetName(batch)

		// 创建新 sheet
		_, err := outFile.NewSheet(sheetName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：无法为批次 %s 创建 sheet: %v\n", batch, err)
			continue
		}

		// 打开源文件
		src, err := excelize.OpenFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：无法打开文件 %s: %v\n", filePath, err)
			continue
		}

		// 读取 "Summary" 工作表的所有行
		rows, err := src.GetRows("Summary")
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告：无法读取 %s 的 Summary 工作表: %v\n", filePath, err)
			src.Close()
			continue
		}

		// 将行数据写入新 sheet
		for rIdx, row := range rows {
			cells := make([]interface{}, len(row))
			for i, v := range row {
				cells[i] = v
			}
			cell, _ := excelize.CoordinatesToCellName(1, rIdx+1) // 从 A1 开始
			if err := outFile.SetSheetRow(sheetName, cell, &cells); err != nil {
				fmt.Fprintf(os.Stderr, "警告：写入行 %d 失败（批次 %s）: %v\n", rIdx+1, batch, err)
			}
		}
		src.Close()
	}

	// 保存输出文件
	if err := outFile.SaveAs(output); err != nil {
		fmt.Fprintf(os.Stderr, "保存输出文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("成功合并到: %s\n", output)
}

// sanitizeSheetName 清理工作表名称，替换非法字符并截断至31字符
func sanitizeSheetName(name string) string {
	illegal := []rune{':', '\\', '/', '?', '*', '[', ']'}
	var sb strings.Builder
	for _, r := range name {
		if r <= 0x1F { // 控制字符直接丢弃
			continue
		}
		replace := false
		for _, il := range illegal {
			if r == il {
				replace = true
				break
			}
		}
		if replace {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	if s == "" {
		s = "Sheet"
	}
	if len(s) > 31 {
		s = s[:31]
	}
	return s
}
