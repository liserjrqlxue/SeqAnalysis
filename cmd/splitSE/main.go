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
	tail = flag.String(
		"e",
		"",
		"tail seq",
	)
	cut = flag.Bool(
		"cut",
		false,
		"if cut trailer",
	)
)

func main() {
	flag.Parse()
	if *input == "" || *output == "" || *barcode == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2/-p required!")
	}

	var (
		header  = strings.ToUpper(*barcode)
		trailer = strings.ToUpper(*tail)

		filter = regexp.MustCompile(`^` + header)

		inList = strings.Split(*input, ",")

		outF = osUtil.Create(*output)
		gw   = gzip.NewWriter(outF)
	)

	if trailer != "" {
		if *cut {
			filter = regexp.MustCompile(`^` + header + `.*?(` + trailer + `)`)
		} else {
			filter = regexp.MustCompile(`^` + header + `.*?` + trailer + ``)
		}
	} else if *cut {
		log.Printf("empty trailer, not cut!")
		*cut = false
	}

	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF))
		)
		log.Printf("split %s", in)
		SplitSE(gr, gw, filter, *cut)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
}

func SplitSE(in io.Reader, out io.Writer, filter *regexp.Regexp, cut bool) {
	var (
		n    = 0
		name string
		seq  string
		note string
		qual string

		scanner = bufio.NewScanner(in)
	)
	if cut {
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
				var match = filter.FindStringSubmatchIndex(seq)
				if match != nil {
					simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq[:match[2]] + "\n" + note + "\n" + qual[:match[2]] + "\n")))
				}
			}
		}
	} else {
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
					simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
				}
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
}
