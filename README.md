# SeqAnalysis

简单序列分析统计

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
2. `polyA     = regexp.MustCompile(`(.*?)` + indexSeq + `(.*?)AAAAAAAA`)`
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
      - 排除 连续3个 `+` --> 连续重复3次
      - 排除 `-`
  - 序列仅有少于4次的连续重复
- `Align3`
  - 遍历 参考序列
    - 同步 目标序列，不匹配 时 记录 `X` 替代
    - 目标序列 长度相同 且 至多 只有 1 个 替代
  - 序列仅有1个碱基替换

## TO-DO

- [ ] 1. 缺失 分长度
- [ ] 2. 插入 把带缺失另存文件
- [ ] 3. 突变 不过滤dup
- [ ] 4. 新增 总表，按个数排序
- [ ] 5. 各表 统计总数，并写在最上面
