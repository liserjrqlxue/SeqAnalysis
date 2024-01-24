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
		log.Fatal("-i/-o/-b required!")
	}

	var (
		filter = regexp.MustCompile(strings.ToUpper(*barcode))

		inList = strings.Split(*input, ",")

		outF = osUtil.Create(*output)
	)

	defer simpleUtil.DeferClose(outF)

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF)).(*gzip.Reader)
		)
		log.Printf("split %s", in)
		SplitSE(gr, outF, filter)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}

}

func SplitSE(in io.Reader, out io.Writer, filter *regexp.Regexp) {
	var (
		n       = 0
		scanner = bufio.NewScanner(in)
	)
	for scanner.Scan() {
		var line = scanner.Text()
		n++
		if n%4 == 2 {
			var match = filter.FindStringIndex(line)
			if match != nil {
				simpleUtil.HandleError(out.Write([]byte(line[:match[0]] + "\t" + line[match[0]:match[1]] + "\t" + line[match[1]:] + "\n")))
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
}
