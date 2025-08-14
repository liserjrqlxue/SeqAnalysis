package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"

	"github.com/liserjrqlxue/DNA/pkg/util"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	id = flag.String(
		"id",
		"",
		"id",
	)
	input = flag.String(
		"i",
		"",
		"input",
	)
	barcode = flag.String(
		"b",
		"",
		"barcode",
	)
	target = flag.String(
		"s",
		"",
		"target seq",
	)
	tail = flag.String(
		"t",
		"",
		"tail barcode",
	)
	line = flag.Bool(
		"line",
		false,
		"plot line",
	)
	logScale = flag.Bool(
		"log",
		false,
		"LogScale of Y",
	)
	offset = flag.Float64(
		"offset",
		0.1,
		"offset in case log(0)",
	)
	distanceThreshold = flag.Int(
		"dt",
		30,
		"distance threshold",
	)

/*
	 	length = flag.Int(
			"l",
			0,
			"length of mutation",
		)
*/
)

func main() {
	flag.Parse()
	if *input == "" || *barcode == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2/-p required!")
	}
	var (
		header = strings.ToUpper(*barcode)
		filter = regexp.MustCompile(`^` + header + `(.*)`)
		inList = strings.Split(*input, ",")

		targetCh = make(chan string, 1024)

		done = make(chan bool)
	)
	if *tail != "" {
		trailer := strings.ToUpper(*tail)
		filter = regexp.MustCompile(`^` + header + `(.*?)` + trailer)
	}

	go func() {
		var (
			count     = make(map[string]int, 1024*1024)
			distCount = make(map[int]int) // 距离 -> 加权个数
			seqs      []string

			topSeq  string
			total   = 0
			top     = 0
			maxDist = 0
		)

		// 收集统计
		for seq := range targetCh {
			count[seq]++
		}

		// 统计总数 & top
		for k, v := range count {
			seqs = append(seqs, k)
			total += v
			if v > top {
				top = v
				topSeq = k
			}
		}
		rate := float64(top) / float64(total)
		meanRate := math.Pow(rate, 1/float64(len(*target)))
		fmt.Printf("%s\t%s+%s\t%d\t%s\t%d\t%.4f%%\t%.4f%%\n", *id, *barcode, *tail, total, topSeq, top, rate*100, meanRate*100)

		// 排序（高频优先）
		sort.Slice(seqs, func(i, j int) bool { return count[seqs[i]] > count[seqs[j]] })

		// 输出明细并统计距离分布
		f := osUtil.Create(*id + "_levenshtein.txt")
		fmtUtil.Fprintln(f, "ID\tCount\tDistance\tSeq")
		for _, k := range seqs {
			hit := count[k]
			dist := levenshtein(*target, k)
			if maxDist < dist {
				maxDist = dist
			}
			fmtUtil.Fprintf(f, "%s\t%d\t%d\t%s\n", *id, hit, dist, k)
			distCount[dist] += hit
		}
		f.Close()

		// 输出距离分布（加权）
		fd := osUtil.Create(*id + "_levenshtein_dist.txt")
		fmtUtil.Fprintln(fd, "Distance\tWeightedCount")
		for d := 0; d <= maxDist; d++ { // 保证顺序输出
			if c, ok := distCount[d]; ok {
				fmt.Fprintf(fd, "%d\t%d\n", d, c)
			} else {
				fmt.Fprintf(fd, "%d\t0\n", d)
			}
		}
		fd.Close()

		// 画图
		plotWidthInch := 16.0
		plotHeightInch := 9.0
		// 缩放系数（基准 8 英寸）
		scale := plotWidthInch / 8.0

		p := plot.New()
		// 设置标题
		p.Title.Text = "Levenshtein Distance Distribution"
		p.Title.TextStyle.Font.Size = vg.Points(16 * scale) // 标题字体 16pt

		// 设置 X 轴标签
		p.X.Label.Text = "Distance"
		p.X.Label.TextStyle.Font.Size = vg.Points(14 * scale) // X 轴标签字体 14pt

		// 设置 Y 轴标签
		p.Y.Label.Text = "Weighted Count"
		p.Y.Label.TextStyle.Font.Size = vg.Points(14 * scale) // Y 轴标签字体 14pt

		// 设置刻度字体
		p.X.Tick.Label.Font.Size = vg.Points(12 * scale) // X 轴刻度字体 12pt
		p.Y.Tick.Label.Font.Size = vg.Points(12 * scale) // Y 轴刻度字体 12pt

		if *logScale {
			p.Y.Scale = plot.LogScale{}
			p.Y.Tick.Marker = plot.LogTicks{}
		}

		// 转成 plotter.Values
		values := make(plotter.Values, maxDist+1)
		pts := make(plotter.XYs, maxDist+1)
		for d := 0; d <= maxDist; d++ {
			v := max(float64(distCount[d]), *offset)
			values[d] = v
			pts[d].Y = v
			pts[d].X = float64(d)
		}

		// 添加柱状图
		if !*logScale {
			// 自动计算 bar 宽度
			totalWidthPoints := plotWidthInch * 72
			barWidth := vg.Points(totalWidthPoints / float64(maxDist+1) * 0.8)

			bar := simpleUtil.HandleError(plotter.NewBarChart(values, barWidth))
			bar.LineStyle.Width = vg.Length(0)
			bar.Color = plotutil.Color(0)
			p.Add(bar)
		}

		// 曲线图
		if *line {
			line := simpleUtil.HandleError(plotter.NewLine(pts))
			line.LineStyle.Width = vg.Points(1.5 * scale)
			line.LineStyle.Color = plotutil.Color(1)
			p.Add(line)
		}

		// 留出右边界空白
		p.X.Min = -0.5
		p.X.Max = float64(maxDist) + 0.5
		// p.NominalX(makeLabels(maxDist)...)

		simpleUtil.CheckErr(p.Save(vg.Length(plotWidthInch)*vg.Inch, vg.Length(plotHeightInch)*vg.Inch, *id+"_levenshtein_dist.png"))

		done <- true
	}()

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF))
		)
		log.Printf("split %s", in)
		SplitSE(gr, filter, len(*barcode), len(*barcode)+len(*target), targetCh)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
	close(targetCh)

	<-done
}

func SplitSE(in io.Reader, filter *regexp.Regexp, start, end int, ch chan<- string) {
	var (
		n       = 0
		scanner = bufio.NewScanner(in)
	)
	for scanner.Scan() {
		var line = scanner.Text()
		n++
		if n%4 == 2 {
			var match = filter.FindStringSubmatchIndex(line)
			if match != nil {
				ch <- line[start:min(end, match[3])]
			} else {
				line = util.ReverseComplement(line)
				match = filter.FindStringSubmatchIndex(line)
				if match != nil {
					ch <- line[start:min(end, match[3])]
				}
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
}

// 生成 X 轴标签 （如果 n 太大则间隔显示）
func makeLabels(n int) []string {
	labels := make([]string, n+1)
	step := 1
	if n > 20 {
		step = n / 10 // 只显示约 10 个刻度
	}
	for i := 0; i <= n; i++ {
		if i%step == 0 {
			labels[i] = fmt.Sprintf("%d", i)
		} else {
			labels[i] = ""
		}
	}
	return labels
}
