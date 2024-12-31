package main

// find highest frequency sequence with count from fastq
// zcat input.fastq.gz | awk 'NR==2'| sort | uniq -c | sort -rn | head -n 20

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"regexp"
	"sort"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// flag
var (
	headCount = flag.Int(
		"n",
		20,
		"top N sequences",
	)
	startPos = flag.Int(
		"s",
		0,
		"start positon",
	)
	endPos = flag.Int(
		"e",
		0,
		"end position",
	)
)

// global
var (
	counts = make(map[string]int)

	gz = regexp.MustCompile(`gz$`)
)

func main() {
	flag.Parse()
	var fqList = flag.Args()

	for _, fq := range fqList {
		fqCount(fq, *startPos, *endPos)
	}

	var keys = make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })

	if len(counts) < *headCount {
		*headCount = len(counts)
	}

	for i := 0; i < *headCount; i++ {
		fmt.Printf("%d\t%s\n", counts[keys[i]], keys[i])
	}
}

func fqCount(fastq string, start, end int) {
	var (
		file    = osUtil.Open(fastq)
		scanner *bufio.Scanner
		i       = -1
	)
	if gz.MatchString(fastq) {
		scanner = bufio.NewScanner(simpleUtil.HandleError(gzip.NewReader(file)))
	} else {
		scanner = bufio.NewScanner(file)
	}
	for scanner.Scan() {
		var line = scanner.Text()
		i++
		if i%4 != 1 {
			continue
		}
		var key = line[start:min(end, len(line))]
		counts[key]++
	}
	slog.Info("fqCount Done", "fq", fastq, "count", (i+1)/4)
}
