package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"log/slog"
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
	skip = flag.String(
		"skip",
		"",
		"skip seq",
	)
)

func main() {
	flag.Parse()
	if *input == "" || *output == "" || *barcode == "" {
		flag.PrintDefaults()
		log.Fatal("-i/-o/-b required!")
	}

	var (
		header  = strings.ToUpper(*barcode)
		trailer = strings.ToUpper(*tail)

		filter  = regexp.MustCompile(`^` + header)
		skipReg *regexp.Regexp

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

	if *skip != "" {
		skipReg = regexp.MustCompile(`^` + header + *skip)
	}

	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF))
		)
		log.Printf("split %s", in)
		SplitSE(gr, gw, filter, skipReg, *cut)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
}

// SplitSE 根据skipReg和cut进行分流
func SplitSE(in io.Reader, out io.Writer, filter, skipReg *regexp.Regexp, cut bool) {
	if cut {
		if skipReg == nil {
			splitCutSE(in, out, filter)
		} else {
			splitCutSkipSE(in, out, filter, skipReg)
		}
	} else {
		if skipReg == nil {
			splitSE(in, out, filter)
		} else {
			splitSkipSE(in, out, filter, skipReg)
		}
	}
}

func splitSE(in io.Reader, out io.Writer, filter *regexp.Regexp) {
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
				simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
			}
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}

func splitSkipSE(in io.Reader, out io.Writer, filter, skipReg *regexp.Regexp) {
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
				if !skipReg.MatchString(seq) {
					simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
				}
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
	slog.Info("finish", "n", n, "reads", n/4)
}

func splitCutSE(in io.Reader, out io.Writer, filter *regexp.Regexp) {
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
			var match = filter.FindStringSubmatchIndex(seq)
			if match != nil {
				simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq[:match[2]] + "\n" + note + "\n" + qual[:match[2]] + "\n")))
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
	slog.Info("finish", "n", n, "reads", n/4)
}

func splitCutSkipSE(in io.Reader, out io.Writer, filter, skipReg *regexp.Regexp) {
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
			var match = filter.FindStringSubmatchIndex(seq)
			if match != nil {
				if !skipReg.MatchString(seq) {
					simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq[:match[2]] + "\n" + note + "\n" + qual[:match[2]] + "\n")))
				}
			}
		}
	}
	simpleUtil.CheckErr(scanner.Err())
	slog.Info("finish", "n", n, "reads", n/4)
}
