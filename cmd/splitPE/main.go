package main

import (
	"PrimerDesigner/util"
	"bufio"
	"flag"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

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
		"output prefix_[12].fq.gz",
	)
	barcode = flag.String(
		"b",
		"",
		"barcode",
	)
	target = flag.String(
		"t",
		"",
		"target seq",
	)
	length = flag.Int(
		"l",
		150,
		"PE reads length",
	)
	tail = flag.String(
		"e",
		"",
		"tail seq",
	)
	merge = flag.Bool(
		"m",
		false,
		"if merged",
	)
)

type IOs struct {
	Fq1  io.ReadCloser
	Fq2  io.ReadCloser
	Gr1  *gzip.Reader
	Gr2  *gzip.Reader
	Out1 io.WriteCloser
	Out2 io.WriteCloser
	OutM io.WriteCloser
	Gw1  *gzip.Writer
	Gw2  *gzip.Writer
	GwM  *gzip.Writer // merged

	InsertSeq string
}

func main() {
	t0 := time.Now()
	flag.Parse()
	if *fq1 == "" || *fq2 == "" || *prefix == "" || *barcode == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2/-p/-b required!")
	}

	var (
		header  = strings.ToUpper(*barcode)
		trailer = strings.ToUpper(*tail)

		filter = regexp.MustCompile(`^` + header)

		ios = CreateIOs(*fq1, *fq2, *prefix, *merge)

		allSeq = *barcode + *target
	)
	defer CloseIOs(ios)

	ios.InsertSeq = allSeq[*length : len(allSeq)-*length]
	log.Print("InsertSeq: ", ios.InsertSeq)

	if trailer != "" {
		filter = regexp.MustCompile(`^` + header + `.*` + trailer + ``)
	}

	SplitPE(ios, filter, *merge)

	log.Print("Done in ", time.Since(t0))
}

func CreateIOs(fq1, fq2, prefix string, merged bool) (ios *IOs) {
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
	if merged {
		ios.OutM = osUtil.Create(prefix + "_merged.fq.gz")
		ios.GwM = gzip.NewWriter(ios.OutM)
	} else {
		ios.Out1 = osUtil.Create(prefix + "_1.fq.gz")
		ios.Out2 = osUtil.Create(prefix + "_2.fq.gz")
		ios.Gw1 = gzip.NewWriter(ios.Out1)
		ios.Gw2 = gzip.NewWriter(ios.Out2)
	}

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
	if ios.Out1 != nil {
		if ios.Gw1 != nil {
			ios.Gw1.Close()
		}
		ios.Out1.Close()
	}
	if ios.Out2 != nil {
		if ios.Gw2 != nil {
			ios.Gw2.Close()
		}
		ios.Out2.Close()
	}
	if ios.OutM != nil {
		ios.OutM.Close()
		if ios.GwM != nil {
			ios.GwM.Close()
		}
	}
}

func SplitPE(ios *IOs, filter *regexp.Regexp, merged bool) {
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
			if merged {
				WriteMerge(ios, name1, name2, seq1, seq2, note1, note2, qual1, qual2, filter)
			} else {
				WritePE(ios, name1, name2, seq1, seq2, note1, note2, qual1, qual2, filter)
			}
		}
	}

	simpleUtil.CheckErr(scanner1.Err())
	simpleUtil.CheckErr(scanner2.Err())
}

func WritePE(ios *IOs, name1, name2, seq1, seq2, note1, note2, qual1, qual2 string, filter *regexp.Regexp) {
	if filter.MatchString(seq1) || filter.MatchString(seq2) {
		simpleUtil.HandleError(ios.Gw1.Write([]byte(name1 + "\n" + seq1 + "\n" + note1 + "\n" + qual1 + "\n")))
		simpleUtil.HandleError(ios.Gw2.Write([]byte(name2 + "\n" + seq2 + "\n" + note2 + "\n" + qual2 + "\n")))
	}
}
func WriteMerge(ios *IOs, name1, name2, seq1, seq2, note1, note2, qual1, qual2 string, filter *regexp.Regexp) {
	if filter.MatchString(seq1) {
		simpleUtil.HandleError(ios.GwM.Write([]byte(name1 + "\n" + seq1 + ios.InsertSeq + util.ReverseComplement(seq2) + "\n" + note1 + "\n" + qual1 + ios.InsertSeq + string(util.Reverse([]byte(qual2))) + "\n")))
	} else if filter.MatchString(seq2) {
		simpleUtil.HandleError(ios.GwM.Write([]byte(name2 + "\n" + seq2 + ios.InsertSeq + util.ReverseComplement(seq1) + "\n" + note2 + "\n" + qual2 + ios.InsertSeq + string(util.Reverse([]byte(qual1))) + "\n")))
	}
}
