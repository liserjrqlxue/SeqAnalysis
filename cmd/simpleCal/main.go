package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"regexp"
	"strings"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/DNA/pkg/util"
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
	tail = flag.String(
		"t",
		"",
		"tail barcode",
	)
	length = flag.Int(
		"l",
		10,
		"length of mutation",
	)
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
			count  = make(map[string]int, 1024*1024)
			total  = 0
			max    = 0
			maxSeq string
		)

		for seq := range targetCh {
			count[seq]++
			total++
		}

		for k, v := range count {
			if v > max {
				max = v
				maxSeq = k
			}
		}
		rate := float64(max) / float64(total)
		meanRate := math.Pow(rate, 1/float64(*length))
		fmt.Printf("%s\t%s+%s\t%d\t%s\t%d\t%.4f%%\t%.4f%%\n", *id, *barcode, *tail, total, maxSeq, max, rate*100, meanRate*100)
		done <- true
	}()

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF))
		)
		log.Printf("split %s", in)
		SplitSE(gr, filter, len(*barcode), len(*barcode)+*length, targetCh)
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
