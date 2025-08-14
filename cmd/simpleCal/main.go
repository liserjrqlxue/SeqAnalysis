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
			count = make(map[string]int, 1024*1024)
			// distance = make(map[string]int, 1024*1024)
			seqs []string

			total  = 0
			top    = 0
			topSeq string
		)

		for seq := range targetCh {
			count[seq]++
		}

		for k, v := range count {
			seqs = append(seqs, k)
			total += v
			// distance[k] = levenshtein(*target, k)

			if v > top {
				top = v
				topSeq = k
			}
		}
		rate := float64(top) / float64(total)
		meanRate := math.Pow(rate, 1/float64(*length))
		fmt.Printf("%s\t%s+%s\t%d\t%s\t%d\t%.4f%%\t%.4f%%\n", *id, *barcode, *tail, total, topSeq, top, rate*100, meanRate*100)

		sort.Slice(seqs, func(i, j int) bool { return count[seqs[i]] > count[seqs[j]] })

		f := osUtil.Create(*id + "_levenshtein.txt")
		for _, k := range seqs {
			dist := levenshtein(*target, k)
			fmtUtil.Fprintf(f, "%s\t%s\t%d\t%d\n", *id, k, count[k], dist)
		}
		f.Close()

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
