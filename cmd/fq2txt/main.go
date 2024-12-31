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

	"github.com/liserjrqlxue/DNA/pkg/util"

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
	reg = flag.String(
		"r",
		"",
		"reg for split",
	)
	skip = flag.Bool(
		"s",
		false,
		"skip with reg",
	)
	rc = flag.Bool(
		"rc",
		false,
		"if use RC",
	)
)

func main() {
	flag.Parse()
	if *input == "" || *output == "" {
		flag.PrintDefaults()
		log.Fatal("-i/-o required!")
	}

	var (
		inList = strings.Split(*input, ",")
		outF   = osUtil.Create(*output)
		gw     = gzip.NewWriter(outF)

		filter *regexp.Regexp
	)
	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)
	if *reg != "" {
		filter = regexp.MustCompile(*reg)
	}

	for _, in := range inList {
		var (
			inF = osUtil.Open(in)
			gr  = simpleUtil.HandleError(gzip.NewReader(inF))
		)
		log.Printf("split %s", in)
		SplitSE(gr, gw, filter, *rc, *skip)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
}

// SplitSE 根据skipReg和cut进行分流
func SplitSE(in io.Reader, out io.Writer, filter *regexp.Regexp, rc, skip bool) {
	if filter == nil {
		fq2seq(in, out)
	} else if skip {
		splitSeqSkip(in, out, filter, rc)
	} else {
		splitSeq(in, out, filter, rc)
	}
}

func splitSeqSkip(in io.Reader, out io.Writer, filter *regexp.Regexp, rc bool) {
	var (
		n       = 0
		scanner = bufio.NewScanner(in)
	)

	if rc {
		for scanner.Scan() {
			var line = scanner.Text()
			n++
			if n%4 == 2 {
				if !filter.MatchString(line) && !filter.MatchString(util.ReverseComplement(line)) {
					simpleUtil.HandleError(out.Write([]byte(line + "\n")))
				}

			}
		}
	} else {
		for scanner.Scan() {
			var line = scanner.Text()
			n++
			if n%4 == 2 {
				if !filter.MatchString(line) {
					simpleUtil.HandleError(out.Write([]byte(line + "\n")))
				}

			}
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}

func splitSeq(in io.Reader, out io.Writer, filter *regexp.Regexp, rc bool) {
	var (
		n       = 0
		scanner = bufio.NewScanner(in)
	)

	if rc {
		for scanner.Scan() {
			var line = scanner.Text()
			n++
			if n%4 == 2 {
				m := filter.FindStringSubmatch(line)
				if m == nil {
					m = filter.FindStringSubmatch(util.ReverseComplement(line))
				}
				if m != nil {
					line = strings.Join(m[1:], "\t") + "\n"
					simpleUtil.HandleError(out.Write([]byte(line)))
				}

			}
		}
	} else {
		for scanner.Scan() {
			var line = scanner.Text()
			n++
			if n%4 == 2 {
				m := filter.FindStringSubmatch(line)
				if m != nil {
					line = strings.Join(m[1:], "\t") + "\n"
					simpleUtil.HandleError(out.Write([]byte(line)))
				}

			}
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}

func fq2seq(in io.Reader, out io.Writer) {
	var (
		n       = 0
		scanner = bufio.NewScanner(in)
	)

	for scanner.Scan() {
		var line = scanner.Text()
		n++
		if n%4 == 2 {
			simpleUtil.HandleError(out.Write([]byte(line + "\n")))
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}
