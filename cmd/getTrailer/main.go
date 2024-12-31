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
		info.filter = regexp.MustCompile(`^` + info.Header + `(.*)`)
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

		ch1           = make(chan string, 1024*1024)
		ch2           = make(chan string, 1024*1024) // rc
		writeFileDone = make(chan bool, 2)
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
				if info.filter.MatchString(seq) {
					ch1 <- seq[len(seq)-*length:]
				} else {
					seqRC := util.ReverseComplement(seq)
					if info.filter.MatchString(seqRC) {
						ch2 <- seqRC[len(seq)-*length:]
					}
				}
			}
		}
		close(ch1)
		close(ch2)
	}()

	// write
	go WriteFile(ch1, *output+".1.txt", writeFileDone)
	go WriteFile(ch2, *output+".2.txt", writeFileDone)

	// readFq done
	for range inList {
		<-readFqDone
	}
	close(ch)

	// writeFile done
	for range []int{1, 2} {
		<-writeFileDone
	}

	slog.Info("done")

}

func WriteCh(in, out chan string, done chan bool) {
	for v := range in {
		out <- v
	}
	done <- true
}

func WriteFile(in chan string, out string, done chan bool) {
	f := osUtil.Create(out)
	defer simpleUtil.DeferClose(f)

	for v := range in {
		fmtUtil.Fprintln(f, v)
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
