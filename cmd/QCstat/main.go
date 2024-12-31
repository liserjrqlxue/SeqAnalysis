package main

import (
	"bufio"
	"flag"
	"log"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	// "compress/gzip"
	gzip "github.com/klauspost/pgzip"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// flag
var (
	input = flag.String(
		"i",
		"",
		"input fq, separated by comma",
	)
	output = flag.String(
		"o",
		"",
		"output",
	)
)

// global
var (
	countAT = 0
	countGC = 0
	gz      = regexp.MustCompile(`\.gz$`)
)

func main() {
	t0 := time.Now()
	flag.Parse()
	// if *input == "" || *output == "" {
	if *input == "" {
		flag.PrintDefaults()
		// log.Fatal("-i/-o required!")
		log.Fatal("-i required!")
	}

	var (
		fqList = strings.Split(*input, ",")
		wg     sync.WaitGroup
		chAT   = make(chan int, 1024*1024)
		chGC   = make(chan int, 1024*1024)
		done   = make(chan bool, 2)
	)

	for _, fq := range fqList {
		wg.Add(1)
		go func(fq string) {
			defer wg.Done()
			ReadFastq(fq, chAT, chGC)
		}(fq)
	}
	go func() {
		for i := range chAT {
			countAT += i
		}
		done <- true
	}()
	go func() {
		for i := range chGC {
			countGC += i
		}
		done <- true
	}()
	wg.Wait()
	close(chAT)
	close(chGC)

	<-done
	<-done

	var GCcontent = float64(countGC*100) / float64(countAT+countGC)
	slog.Info("Summary", "AT", countAT, "GC", countGC, "GC%", GCcontent)
	var out = osUtil.Create(*output)
	defer simpleUtil.DeferClose(out)
	fmtUtil.Fprintf(out, "AT:\t%d\nGC:\t%d\nGC%%:\t%.2f%%\n", countAT, countGC, GCcontent)

	slog.Info("Done", "time", time.Since(t0))
}

func ReadFastq(fq string, chAT, chGC chan int) {
	var (
		file    = osUtil.Open(fq)
		scanner *bufio.Scanner
		i       = -1
		gr      *gzip.Reader
	)
	defer simpleUtil.DeferClose(file)
	if gz.MatchString(fq) {
		gr = simpleUtil.HandleError(gzip.NewReader(file))
		defer simpleUtil.DeferClose(gr)
		scanner = bufio.NewScanner(gr)
	} else {
		scanner = bufio.NewScanner(file)
	}
	for scanner.Scan() {
		var line = scanner.Bytes()
		i++
		if i%4 != 1 {
			continue
		}
		var AT = 0
		var GC = 0
		for _, c := range line {
			if c == 'A' || c == 'T' {
				AT++
			} else if c == 'G' || c == 'C' {
				GC++
			}
		}
		chAT <- AT
		chGC <- GC
	}
}
