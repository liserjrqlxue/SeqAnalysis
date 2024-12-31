package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math"
	"regexp"
	"strings"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"
	"github.com/xuri/excelize/v2"

	"github.com/liserjrqlxue/DNA/pkg/util"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	input = flag.String(
		"i",
		"",
		"input txt, use summary.xlsx",
	)
	fq = flag.String(
		"fq",
		"",
		"fq, comma separated",
	)
	output = flag.String(
		"o",
		"",
		"output",
	)
	trailer = flag.String(
		"t",
		"",
		"trailer",
	)
)

type INFO struct {
	Name    string
	Header  string
	Target  string
	Trailer string
	FqPath  string
	Hit     int
	TopSeq  string
	TopNum  int
	TopRate float64
	//几何平均数
	TopGeoMean float64

	filter     *regexp.Regexp
	readBuffer chan string
	status     chan bool
}

func (i *INFO) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%d\t%.2f%%\t%.2f%%", i.Name, i.Header, i.Target, i.Trailer, i.Hit, i.TopSeq, i.TopNum, i.TopRate*100, i.TopGeoMean*100)
}

func (i *INFO) Cal() {
	slog.Info("Cal", "name", i.Name)
	var (
		count  = make(map[string]int, 1024*1024)
		total  = 0
		max    = 0
		maxSeq string
	)

	slog.Info("load from i.readBuffer", "name", i.Name)
	for seq := range i.readBuffer {
		count[seq]++
		total++
	}
	slog.Info("load from i.readBuffer done", "name", i.Name)

	for k, v := range count {
		if v > max {
			max = v
			maxSeq = k
		}
	}

	i.Hit = total
	i.TopSeq = maxSeq
	i.TopNum = max
	i.TopRate = float64(max) / float64(total)
	i.TopGeoMean = math.Pow(i.TopRate, 1/float64(len(i.TopSeq)))
	i.status <- true
}

func LoadInput(input string) (infos []*INFO) {
	var xlsx = simpleUtil.HandleError(excelize.OpenFile(input))
	var rows = simpleUtil.HandleError(xlsx.GetRows("Summary"))
	var title = rows[0]
	for i := 1; i < len(rows); i++ {
		data := make(map[string]string)
		for j, cell := range rows[i] {
			data[title[j]] = cell
		}
		var info = &INFO{
			Name:       data["样品名称"],
			Header:     data["靶标序列"],
			Target:     data["合成序列"],
			Trailer:    data["后靶标"],
			FqPath:     data["路径-R1"],
			readBuffer: make(chan string, 1024*1024),
			status:     make(chan bool, 1),
		}
		if info.Trailer != "" {
			info.filter = regexp.MustCompile(`^` + info.Header + `(.*)` + info.Trailer)
		} else {
			info.filter = regexp.MustCompile(`^` + info.Header + `(.*)`)
		}
		infos = append(infos, info)
	}
	return
}

func main() {
	flag.Parse()
	if *input == "" {
		flag.Usage()
		log.Fatal("-i required!")
	}

	var inList = strings.Split(*fq, ",")

	var infos = LoadInput(*input)

	var (
		chMap = make(map[string]chan string)
		done  = make(chan bool, len(inList))
	)

	// 读取fq
	for _, in := range inList {
		slog.Info("ReadFq", "fq", in)
		chMap[in] = make(chan string, 1024*1024)
		go ReadFq(in, chMap[in], done)
	}

	// 分配给每个info
	go func() {
		for fqPath, ch := range chMap {
			for seq := range ch {
				for _, info := range infos {
					if info.FqPath == fqPath { // fqPath是路径,不是名字
						if info.filter.MatchString(seq) {
							info.readBuffer <- seq[len(info.Header) : len(seq)-len(info.Trailer)]
						} else if info.filter.MatchString(util.ReverseComplement(seq)) {
							info.readBuffer <- util.ReverseComplement(seq)[len(info.Header) : len(seq)-len(info.Trailer)]
						}
					}
				}

			}
		}
		for _, info := range infos {
			slog.Info("close", "info", info.Name)
			close(info.readBuffer)
		}
	}()

	// 每个info计算
	for _, info := range infos {
		go info.Cal()
	}

	// wait
	for range inList {
		<-done
	}
	for _, ch := range chMap {
		close(ch)
	}
	// 等待所有info计算完成
	for _, info := range infos {
		slog.Info("wait", "info", info.Name)
		<-info.status
		fmt.Println(info)
	}

	if *output != "" {
		slog.Info("write", "output", *output)
		out := osUtil.Create(*output)
		defer simpleUtil.DeferClose(out)
		fmtUtil.Fprintf(out, "name\theader\ttarget\ttrailer\thit\ttopSeq\ttopNum\ttopRate\ttopGeoMean\n")
		for _, info := range infos {
			fmtUtil.Fprintf(out, "%s\n", info)
		}
	}

	slog.Info("done")

}

func ReadFq(in string, ch chan string, done chan bool) {
	var (
		inF     = osUtil.Open(in)
		gr      = simpleUtil.HandleError(gzip.NewReader(inF))
		scanner = bufio.NewScanner(gr)

		n = 0
	)
	defer simpleUtil.DeferClose(gr)
	defer simpleUtil.DeferClose(inF)

	for scanner.Scan() {
		var line = scanner.Text()
		n++
		if n%4 == 2 {
			ch <- line
		}
	}

	if scanner.Err() != nil {
		log.Fatal(scanner.Err())
	}
	done <- true
}
