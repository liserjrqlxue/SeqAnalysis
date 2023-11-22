# SeqCount

从 FASTQ 中提取序列信息，计数排序并输出 TOP N

## 特点

1. 使用 Go 编码， 适合 Windows 下使用， Linux 下某些情况下也可作为 zcat|awk 'N%4==2'|cut -c s-e|sort|uniq -c|sort -nr|head 等管道操作的替代，未进行性能评估和资源消耗优化
2. 使用 kaluspost/pgzip 加速 gzip 文件读取
3. 多文件读取尚未使用并行处理
4. 使用 sort.SliceStable 排序 seq

## 后续优化

1. 增加 pprof 排查性能瓶颈
2. scanner.Bytes() vs. scanner.Text()
3. 多文件读取并行
   1. count 加写锁 sync.Mutex（实现简单稳定）
   2. count 的 key 从 chan1 获取（控制复杂些，需保证完整性）
      1. 每个文件最后传输结束信号通过 chan2 传送给计数器，计数达到文件数后close(chan1)
      2. 每次都需要一次判断，跟 sync.Mutex 相比不一定会有性能上的提升
4. 排序先排 count 的 value，过滤 top N 后再排 key
   1. 数据量还是时，先对 value 局部排序过滤 top N, 其它部分过滤后 再排序更新 top N, 最后对 key 过滤和排序

```go
go func(){
    for key := range chan1 {
        count[key]++
    }
    countDone <- true
}

go func(){
    for _ := range chan2 {
        done++
        if done == len(files){
            close(chan1)
        }
    }
}

<-countDone
// sort and print top N

```
