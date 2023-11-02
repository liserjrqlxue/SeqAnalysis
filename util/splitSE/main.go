package main

import (
	"bufio"
	"flag"
	"log"
	"regexp"

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
		match   = regexp.MustCompile(`^` + *barcode)
		inF     = osUtil.Open(*input)
		outF    = osUtil.Create(*output)
		gr      = simpleUtil.HandleError(gzip.NewReader(inF)).(*gzip.Reader)
		gw      = gzip.NewWriter(outF)
		scanner = bufio.NewScanner(gr)

		n    = 0
		name []byte
		seq  []byte
		note []byte
		qual []byte
	)

	defer simpleUtil.DeferClose(inF)
	defer simpleUtil.DeferClose(gr)

	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)

	for scanner.Scan() {
		var line = scanner.Bytes()
		n++
		switch n % 4 {
		case 1:
			name = line
		case 2:
			seq = line
		case 3:
			note = line
		case 4:
			qual = line
			if match.Match(seq) {
				simpleUtil.HandleError(gw.Write(name))
				simpleUtil.HandleError(gw.Write(seq))
				simpleUtil.HandleError(gw.Write(note))
				simpleUtil.HandleError(gw.Write(qual))
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
}
