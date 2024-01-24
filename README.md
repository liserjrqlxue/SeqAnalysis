# SeqAnalysis

合成序列评估软件

## 运行环境

### 硬件

CPU 1 核，内存受输入数据量影响。
实际使用也受并行分析影响

### 软件

`Go 1.20` 以上版本 （编译需要，编译后的二进制软件可直接运行不需要其他依赖）

### 系统

因为 `Go` 语言具有跨平台编译支持，而软件本身没有依赖特定系统平台的库文件，所以 `Windows`、`Linux`、`MacOS` 理论上均支持。
目前已测试的系统平台有 `Windows11` 以及 `WSL` 下 `Ubuntu 20.04.6 LTS`

## 使用说明

### 安装依赖（可选）

**注意**：直接使用对应系统平台可用的预编译软件可跳过本步骤

因为编译需求，需要安装 `Go 1.20` 以上版本 安装方式如下：

1. 去 [Go](https://go.dev/) 官方网站下载对应系统的软件安装包

2. 安装 `Go` 软件

   1. `Windows` 下直接双击 msi 包安装;`Linux` 下直接解压 tar.gz 包
   2. 配置好 `GOROOT`、`GOPATH` 等环境变量，并将 `go` 添加到 `PATH` 查找环境
   3. 详情参考官网[安装文档](https://go.dev/doc/install)

### 安装软件

1. 获取软件：
   1. 使用 `git clone https://github.com/liserjrqlxue/SeqAnalysis.git` 获取源码
   2. 也可以直接通过解压提供的软件包 `SeqAnalysis.zip` 获取源码和编译软件
2. 进入解压文件夹，使用 `go build` 重新编译编（可选）
3. 使用 `SeqAnalysis/SeqAnalysis` （`linux`下）或 `SeqAnalysis\SeqAnalysis.exe` （`windows`下） # 代码逻辑分析

## `main`

1. 遍历输入 `input.txt` --> `s` = `fastq` + `target sequence` + `index sequence`
   1. 创建数据结构 `SeqInfo`
   2. `seqInfo.Init()`
   3. [`seqInfo.CountError4()`](#seqinfocounterror4)
   4. 输出统计信息 `output.txt`
      1. `Summary of s`
      2. `seqInfo.WriteStats(out)`
      3. `seqInfo.WriteDistributionFreq(out)`

## `seqInfo.CountError4`

1. [`seqInfo.WriteSeqResult("SeqResult.txt")`](#seqinfowriteseqresult)
2. `seqInfo.GetHitSeq()` 按 计数 排序
3. `seqInfo.WriteSeqResultBarCode(seqBarCode)`
4. [`seqInfo.WriteSeqResultNum()`](#seqinfowriteseqresultnum)
5. `seqInfo.UpdateDistributionStats()`
6. `seqInfo.WriteDistributionNum(disNum)`
7. `seqInfo.WriteDistributionFreq(disFrequency)`
8. `write Summary.txt`
   1. `seqInfo.WriteStats(summary)`
   2. `seqInfo.WriteDistributionFreq(summary)`

## `seqInfo.WriteSeqResult`

1. `seqHit    = regexp.MustCompile(indexSeq + tarSeq)`
2. `polyA     = regexp.MustCompile(`(._?)`+ indexSeq +`(._?)AAAAAAAA`)`
3. 统计序列 --> `seqInfo.HitSeqCount[tSeq]++`
   1. `seqHit` 匹配，`tarSeq` 计数
   2. `polyA` 匹配， `indexSeq` 与 `AAAAAAAA` 之间 序列 计数
      1. 空序列 记录 序列 为 `X`
      2. 不含 N 且 长度不比目标长度长 `10bp` 记录 序列

## `seqInfo.WriteSeqResultNum`

三种比对模式，四种序列

- `Algin1` 缺失模式
  - 遍历 参考序列
    - 同步 目标序列, 不匹配 时 插入 `-` 占位
    - 目标序列 完全 比对 时 比对成功
  - 序列仅有缺失（或完全相同）
- `Align2`
  - 遍历 参考序列
    - 同步 目标序列， 不匹配 时
      - 与 上一 参考序列 相同， 补 `+`, 继续 比较
      - 补 `-`
    - 目标序列 比对 或 作为连续重复 比对 时 比对成功
      - 排除 首位 `+`
      - 排除 连续 3 个 `+` --> 连续重复 3 次
      - 排除 `-`
  - 序列仅有少于 4 次的连续重复
- `Align3`
  - 遍历 参考序列
    - 同步 目标序列，不匹配 时 记录 `X` 替代
    - 目标序列 长度相同 且 至多 只有 1 个 替代
  - 序列仅有 1 个碱基替换

## TO-DO

- [x] 缺失 分长度
  - [x] 2 个缺失区分连续缺失
  - [x] 计算比例
- [x] 插入 把带缺失另存文件
- [x] 突变 不过滤 dup
- [x] BarCode 补充比对结果
- [x] 各表 统计总数，并写在最上面
- [ ] `summary.xlsx`
  - [x] `excel` 命名添加时间，避免同名文件无法打开
  - [x] 添加单步错误率
  - [x] 输入改为解析 `excel`，并在 `summary.xlsx` 内继承
  - [x] 命名 `summary-` + `输入名称` + `-[DATE].xlsx`
  - [x] `样品名称` 链接单样品 `excel`
  - [x] 按 平行 分组计算 平均收率 平均准确率 收率误差 准确率误差
  - [x] 比例和个数拆分
  - [x] 单步准确率 -> 单步错误率
- [x] 输入
  - [x] 默认输入 `input.xlsx`
- [x] 输出
  - [x] 输出目录 = 输入目录+".分析"
- [x] 性能
  - [x] `plot.R` 内存消耗过大
- [x] 靶标兼容N
