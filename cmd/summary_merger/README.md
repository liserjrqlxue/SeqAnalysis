# summary_merger

它根据指定的目录 `DIR`，读取符合特定格式的 Excel 文件，提取每个批次（Batch）中最新的 "Summary" 工作表，合并到一个新的 Excel 文件中，并按版本号排序批次。程序依赖 `excelize/v2` 处理 Excel，并使用 `fvbommel/sortorder` 实现 `sort -V` 风格的自然排序。

## 使用说明

1. **安装依赖**：
   ```bash
   go get github.com/xuri/excelize/v2
   go get github.com/fvbommel/sortorder
   ```

2. **编译**：
   ```bash
   go build -o summary_merger
   ```

3. **运行**：
   ```bash
   ./summary_merger -d /path/to/DIR
   # 或指定输出文件
   ./summary_merger -d /path/to/DIR -o /path/to/output.xlsx
   ```

## 程序逻辑

- 扫描 `DIR` 下所有以 `{DIR}.` 开头、以 `.merged` 结尾的子目录。
- 从目录名中提取 `Batch` 名称。
- 在每个子目录中查找符合 `summary-{DIR}.{Batch}.merged-YYYYMMDD.xlsx` 格式的文件，选取日期（YYYYMMDD）最大的文件。
- 收集所有 Batch 及其对应文件路径。
- 使用自然排序（`sort -V`）对 Batch 排序。
- 创建输出 Excel，在 `Sheet1` 中写入 Batch 名称和文件路径。
- 依次打开每个源文件，读取其 `Summary` 工作表，将内容复制到输出文件中以 Batch 命名的新工作表（自动清理非法字符、截断长度）。

## 注意事项

- 工作表名称超过31字符会被截断，非法字符替换为下划线，若因此产生重名可能导致后续 sheet 创建失败（一般 Batch 唯一可避免）。
- 日期比较基于字符串字典序，YYYYMMDD 格式可直接比较。
- 输出文件路径默认为 `DIR/DIR.summary.xlsx`，若文件已存在会被覆盖。