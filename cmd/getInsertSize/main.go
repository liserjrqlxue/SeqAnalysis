/*
提取尾部序列，区分RC
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"regexp"
	"strings"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/DNA/pkg/util"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
)

var (
	input = flag.String(
		"i",
		"",
		"input txt, use summary.txt",
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
	length = flag.Int(
		"l",
		20,
		"length",
	)
)

type INFO struct {
	Name    string
	Header  string
	Target  string
	Trailer string
	Hit     int
	TopSeq  string
	TopNum  int
	TopRate float64
	//几何平均数
	TopGeoMean float64

	filter       *regexp.Regexp
	filterRC     *regexp.Regexp
	readBuffer   chan string
	readBufferRC chan string
	status       chan bool
}

func (i *INFO) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%d\t%.2f%%\t%.2f%%", i.Name, i.Header, i.Target, i.Trailer, i.Hit, i.TopSeq, i.TopNum, i.TopRate*100, i.TopGeoMean*100)
}

func LoadInput(input string) (infos []*INFO) {
	data, _ := textUtil.File2MapArray(input, "\t", nil)
	for _, item := range data {
		var info = &INFO{
			Name:         item["名字"],
			Header:       item["靶标"],
			Target:       item["合成"],
			readBuffer:   make(chan string, 1024*1024),
			readBufferRC: make(chan string, 1024*1024),
			status:       make(chan bool, 1),
		}
		info.filter = regexp.MustCompile(`^` + info.Header)
		info.filterRC = regexp.MustCompile(util.ReverseComplement(info.Header) + "$")
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
		ch         = make(chan string, 1024*1024)
		readFqDone = make(chan bool, len(inList))

		chLength       = make(chan int, 1024*1024)
		writeStatsDone = make(chan bool)
	)

	// 读取fq
	for _, in := range inList {
		slog.Info("ReadFq", "fq", in)
		go ReadFq(in, ch, readFqDone)
	}

	// 分配给每个info,每个info有两个ch
	go func() {
		for seq := range ch {
			for _, info := range infos {
				if info.filter.MatchString(seq) || info.filterRC.MatchString(seq) {
					chLength <- len(seq)
				}
			}
		}
		close(chLength)
	}()

	// write
	go WriteStatsFile(chLength, *output+".stats.txt", writeStatsDone)

	// readFq done
	for range inList {
		<-readFqDone
	}
	close(ch)

	// writeFile done
	<-writeStatsDone

	slog.Info("done")

}

func WriteStatsFile(in chan int, out string, done chan bool) {
	f := osUtil.Create(out)
	defer simpleUtil.DeferClose(f)
	var stats = [300]int{}

	for v := range in {
		if v <= 300 {
			stats[v-1]++
		} else {
			stats[299]++
		}
	}
	for i, v := range stats {
		fmtUtil.Fprintln(f, i+1, "\t", v)
	}
	done <- true
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
