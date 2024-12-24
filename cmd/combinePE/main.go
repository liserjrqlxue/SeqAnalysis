package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"time"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/DNA/pkg/util"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	fq1 = flag.String(
		"1",
		"",
		"input fq1",
	)
	fq2 = flag.String(
		"2",
		"",
		"input fq2",
	)
	prefix = flag.String(
		"p",
		"",
		"output prefix.fq.gz",
	)
	insertSeq = flag.String(
		"i",
		"",
		"insert string",
	)
)

type IOs struct {
	Fq1  io.ReadCloser
	Fq2  io.ReadCloser
	Gr1  *gzip.Reader
	Gr2  *gzip.Reader
	OutM io.WriteCloser
	GwM  *gzip.Writer // merged

	InsertSeq string
}

func main() {
	t0 := time.Now()
	flag.Parse()
	if *fq1 == "" || *fq2 == "" || *prefix == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2/-p required!")
	}

	var ios = CreateIOs(*fq1, *fq2, *prefix)
	defer CloseIOs(ios)

	ios.InsertSeq = *insertSeq
	log.Print("InsertSeq: ", ios.InsertSeq)
	CombinePE(ios)

	log.Print("Done in ", time.Since(t0))
}

func CreateIOs(fq1, fq2, prefix string) (ios *IOs) {
	var (
		inF1 = osUtil.Open(fq1)
		inF2 = osUtil.Open(fq2)
		gr1  = simpleUtil.HandleError(gzip.NewReader(inF1))
		gr2  = simpleUtil.HandleError(gzip.NewReader(inF2))
	)

	ios = &IOs{
		Fq1: inF1,
		Fq2: inF2,
		Gr1: gr1,
		Gr2: gr2,
	}
	ios.OutM = osUtil.Create(prefix + ".fq.gz")
	ios.GwM = gzip.NewWriter(ios.OutM)

	return
}

func CloseIOs(ios *IOs) {
	if ios.Fq1 != nil {
		if ios.Gr1 != nil {
			ios.Gr1.Close()
		}
		ios.Fq1.Close()
	}
	if ios.Fq2 != nil {
		if ios.Gr2 != nil {
			ios.Gr2.Close()
		}
		ios.Fq2.Close()
	}
	if ios.OutM != nil {
		if ios.GwM != nil {
			ios.GwM.Flush()
			ios.GwM.Close()
		}
		ios.OutM.Close()
	}
}

func CombinePE(ios *IOs) {
	var (
		n     = 0
		name1 string
		seq1  string
		note1 string
		qual1 string
		name2 string
		seq2  string
		note2 string
		qual2 string

		scanner1 = bufio.NewScanner(ios.Gr1)
		scanner2 = bufio.NewScanner(ios.Gr2)
	)

	for scanner1.Scan() && scanner2.Scan() {
		line1 := scanner1.Text()
		line2 := scanner2.Text()
		n++
		switch n % 4 {
		case 1:
			name1 = line1
			name2 = line2
		case 2:
			seq1 = line1
			seq2 = line2
		case 3:
			note1 = line1
			note2 = line2
		case 0:
			qual1 = line1
			qual2 = line2
			WriteMerge(ios, name1, name2, seq1, seq2, note1, note2, qual1, qual2)
		}
	}

	simpleUtil.CheckErr(scanner1.Err())
	simpleUtil.CheckErr(scanner2.Err())
	log.Printf("Lines: %d, PairEnds: %d", n, n/4)
}

func WriteMerge(ios *IOs, name1, name2, seq1, seq2, note1, note2, qual1, qual2 string) {
	simpleUtil.HandleError(ios.GwM.Write([]byte(name1 + "\n" + seq1 + ios.InsertSeq + util.ReverseComplement(seq2) + "\n" + note1 + "\n" + qual1 + ios.InsertSeq + string(util.Reverse([]byte(qual2))) + "\n")))
}
