package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"regexp"
	"strings"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	input = flag.String(
		"i",
		"",
		"input",
	)
	output = flag.String(
		"o",
		"",
		"output",
	)
	barcode = flag.String(
		"b",
		"",
		"barcode",
	)
)

func main() {
	flag.Parse()
	if *input == "" || *output == "" || *barcode == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2/-p required!")
	}

	var (
		filter = regexp.MustCompile(`^` + strings.ToUpper(*barcode))

		inList = strings.Split(*input, ",")

		outF = osUtil.Create(*output)
		gw   = gzip.NewWriter(outF)
	)

	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF)).(*gzip.Reader)
		)
		log.Printf("split %s", in)
		SplitSE(gr, gw, filter)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}

}

func SplitSE(in io.Reader, out io.Writer, filter *regexp.Regexp) {
	var (
		n    = 0
		name string
		seq  string
		note string
		qual string

		scanner = bufio.NewScanner(in)
	)
	for scanner.Scan() {
		var line = scanner.Text()
		n++
		switch n % 4 {
		case 1:
			name = line
		case 2:
			seq = line
		case 3:
			note = line
		case 0:
			qual = line
			if filter.MatchString(seq) {
				simpleUtil.HandleError(out.Write([]byte(name + "\n")))
				simpleUtil.HandleError(out.Write([]byte(seq + "\n")))
				simpleUtil.HandleError(out.Write([]byte(note + "\n")))
				simpleUtil.HandleError(out.Write([]byte(qual + "\n")))
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
}
