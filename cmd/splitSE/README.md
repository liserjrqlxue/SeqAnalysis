# splitSE

SE 模式对测序数据按照序列分析的逻辑进行数据抽提，
基于靶标、尾靶标将对应样品序列分析需要的数据提取出来。

使用场景：

- 拆分数据，避免序列分析主程序读取数据过大
- 拆分数据，进行特殊处理（如过滤特定错误序列）
- 拆分数据，直接进行测序结果分析（如合成基座分析）

## 用法

```bash
splitSE -i in.fq.gz -o out.fq.gz -b AAGC -e GGTTAA [-cut] [-skip TT] [-rc] [-minLen 20]
```

输入必须是 gzip 压缩的 FASTQ（`.fq.gz`），`-i` 支持逗号分隔多个文件，依次写入同一个 `-o`。

## 参数

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `-i` | 必填 | 输入 fq.gz，多个文件用 `,` 分隔 |
| `-o` | 必填 | 输出 fq.gz |
| `-b` | 空 | barcode。**不要求出现在 read 开头**，可出现在任意位置，若出现多次则取最靠 3' 端（最后一次）的那次匹配 |
| `-e` | 空 | 尾靶标（trailer），在 barcode 之后的序列里取第一次出现 |
| `-cut` | `false` | 在尾靶标**起点**截断，即丢弃尾靶标本身及其后的序列；不加则保留到尾靶标末尾 |
| `-skip` | 空 | 紧跟 barcode 之后的序列。命中 `barcode + skip` 前缀的 read 会被丢弃，用于过滤特定错误序列 |
| `-rc` | `false` | 正向未命中时，把 read 反向互补后按同一套规则再匹配一次 |
| `-minLen` | `0` | 长度阈值（read 起点到尾靶标起点的距离），小于该值的 read 不输出 |

`-b` 与 `-e` 至少要给一个：

- 只给 `-b`：不截断，整条 read 原样输出（只做 barcode 过滤）
- 只给 `-e`：tail-only 模式，不校验 barcode，只按尾靶标截断

序列比对大小写不敏感（内部统一转大写匹配），输出保留 read 原始大小写。

## 输出规则

输出始终是输入 read 的**前缀**，所以 barcode 之前的残留碱基会一并保留。

| 参数组合 | 输出 |
| --- | --- |
| `-b ... -e ...` | read 起点 → 第一个尾靶标末尾 |
| `-b ... -e ... -cut` | read 起点 → 第一个尾靶标起点（不含尾靶标） |
| `-b ...` | 整条 read |
| `-e ...`（同 `-cut` 与否） | 同上，但不要求 barcode |

命中后质量值会与输出序列**等长**：截断时同步截断，`-rc` 时先整体反转再截断。

## 处理流程

1. 按 4 行为一条 read 读入 FASTQ。
2. 正向匹配：`-b` 在 read 内取最后一次出现；`-e` 在 barcode 之后的序列里取第一次出现。
3. 正向未命中且给了 `-rc`：对 read 做反向互补（同时反转质量值），再按第 2 步同一套规则匹配。
4. 命中后若给了 `-skip`，且输出序列以 `barcode + skip` 开头，则丢弃该 read。
5. 到尾靶标的距离小于 `-minLen` 则丢弃。
6. 写出：序列与质量值（`-rc` 时质量值已反转）。

未命中的 read（找不到 barcode，或给了 `-e` 但找不到尾靶标）直接丢弃，不做补偿。

## 示例

```bash
# 保留 barcode 到尾靶标末尾
splitSE -i A.fq.gz,B.fq.gz -o out.fq.gz -b AAGC -e GGTTAA

# 切掉尾靶标及其后（-cut 后接 -e 最常用组合）
splitSE -i A.fq.gz -o out.fq.gz -b AAGC -e GGTTAA -cut

# 同时兼容反向测序的 read
splitSE -i A.fq.gz -o out.fq.gz -b AAGC -e GGTTAA -cut -rc

# 过滤 barcode 后紧跟 TT 的错误序列
splitSE -i A.fq.gz -o out.fq.gz -b AAGC -e GGTTAA -skip TT

# 只用尾靶标切分，不校验 barcode
splitSE -i A.fq.gz -o out.fq.gz -e GGTTAA -cut
```

## TO-DO

1. [ ] 批量拆分数据，以及是否参考主程序的方式只读取输入一次
2. [ ] 通过1，把主程序拆分步骤，重分析可跳过拆分数据操作
