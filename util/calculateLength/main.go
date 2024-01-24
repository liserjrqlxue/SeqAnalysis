package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
)

var (
	length = flag.Int("l", 10, "length of mutation")
	output = flag.String("o", "", "output file")
	input  = flag.String("i", "", "input file")
)

// const
const (
	// 加权概率
	plusP = 0.9
	P0    = 0.99
	P1    = 0.5
)

// struct
type SEQ struct {
	seq      string
	fail     int
	currentP float64
}

// global
var (
	Ns               = [2]string{"Y", "N"}
	out              *os.File
	pArray           [20]float64
	qArray           [20]float64
	probabilityArray [21]float64
)

func main() {
	flag.Parse()
	if *output == "" {
		flag.Usage()
		log.Fatal("-o required!")
	}

	pArray[0] = P0
	qArray[0] = 1 - P0
	pArray[1] = P1
	qArray[1] = 1 - P1
	for i := 2; i < 20; i++ {
		pArray[i] = pArray[i-1] * plusP
		qArray[i] = 1 - pArray[i]
	}

	out = osUtil.Create(*output)
	defer simpleUtil.DeferClose(out)
	var outHist = osUtil.Create(*output + ".histogram.txt")
	defer simpleUtil.DeferClose(outHist)

	log.Print("start")
	var (
		seqList []string
		wg      sync.WaitGroup
	)
	if *input == "" {
		seqList = []string{""}
	} else {
		seqList = textUtil.File2Array(*input)
	}
	var count = make(chan int, 1)
	count <- 0
	for i, s := range seqList {
		for _, n := range Ns {
			wg.Add(1)
			go addX(SEQ{s, 0, 1}, n, &wg, count)
		}
		if i%10000 == 9999 {
			wg.Wait()
		}
	}
	wg.Wait()

	var count1 = <-count
	log.Printf("count:%d", count1)
	for i := 0; i < 21; i++ {
		fmtUtil.Fprintf(outHist, "%d\t%f\n", i, probabilityArray[i])
	}
}

func addX(Seq SEQ, N string, wg *sync.WaitGroup, count chan int) {
	defer wg.Done()
	var (
		b        = Seq.seq
		fail     = Seq.fail
		currentP = Seq.currentP
	)
	b += N
	switch N {
	case "Y":
		currentP *= pArray[fail]
	case "N":
		currentP *= qArray[fail]
		fail++
	}

	if len(b) >= *length {
		var t = <-count
		t++
		var hit = len(b) - fail
		fmtUtil.Fprintf(
			out,
			"%s\t%d\t%d\t%d\t%.10f\n",
			b, len(b), hit, fail, currentP,
		)
		probabilityArray[hit] += currentP

		count <- t
		return
	}

	var wg1 sync.WaitGroup

	for _, n := range Ns {
		wg1.Add(1)
		go addX(SEQ{b, fail, currentP}, n, &wg1, count)
	}
	wg1.Wait()
}
