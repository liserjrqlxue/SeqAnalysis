package main

// find highest frequency sequence with count from fastq
// zcat input.fastq.gz | awk 'NR==2'| sort | uniq -c | sort -rn | head -n 20

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
)

// global
var (
	counts    = make(map[string]int)
	seqList   = make(map[int][]string)
	countList []int
	seq       []string
	count     []int

	gz = regexp.MustCompile(`gz$`)
)

func main() {
	flag.Parse()
	var fqList = flag.Args()

	for _, fq := range fqList {
		fqCount(fq)
	}

	for k, v := range counts {
		countList = append(countList, v)
		seqList[v] = append(seqList[v], k)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(countList)))

	for _, v := range countList[:*headCount] {
		for _, k := range seqList[v] {
			seq = append(seq, k)
			count = append(count, v)
		}
	}

	if len(seq) < *headCount {
		*headCount = len(seq)
	}

	for i := 0; i < *headCount; i++ {
		fmt.Printf("%d\t%s\n", count[i], seq[i])
	}
}

func fqCount(fastq string) {
	var (
		file    = osUtil.Open(fastq)
		scanner *bufio.Scanner
		i       = -1
	)
	if gz.MatchString(fastq) {
		scanner = bufio.NewScanner(simpleUtil.HandleError(gzip.NewReader(file)).(io.Reader))
	} else {
		scanner = bufio.NewScanner(file)
	}
	for scanner.Scan() {
		var line = scanner.Text()
		i++
		if i%4 != 1 {
			continue
		}
		counts[line]++
	}
}
