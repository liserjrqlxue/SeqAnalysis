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
	rc = flag.Bool(
		"rc",
		false,
		"if use rc",
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
		tail    bool

		filter  = regexp.MustCompile(`^` + header)
		skipReg *regexp.Regexp

		inList = strings.Split(*input, ",")

		outF = osUtil.Create(*output)
		gw   = gzip.NewWriter(outF)
	)

	if trailer != "" {
		tail = true
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
		SplitSE(gr, gw, filter, skipReg, *cut, *rc, tail)
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
}

// out io.Writer, name, seq, note, qual string, filter *regexp.Regexp, rc bool
type PrintFQ func(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool)

// SplitSE 根据skipReg和cut进行分流
func SplitSE(in io.Reader, out io.Writer, filter, skipReg *regexp.Regexp, cut, rc, tail bool) {
	if cut { // 切除尾靶标
		if skipReg == nil {
			splitSE(in, out, filter, skipReg, rc, PrintMatchCut)
		} else {
			splitSE(in, out, filter, skipReg, rc, PrintSkipMatchCut)
		}
	} else if tail { // 切除尾靶标后面的
		if skipReg == nil {
			splitSE(in, out, filter, skipReg, rc, PrintMatchTrailer)
		} else {
			splitSE(in, out, filter, skipReg, rc, PrintSkipMatchTrailer)
		}
	} else { // 不切除尾部
		if skipReg == nil {
			splitSE(in, out, filter, skipReg, rc, PrintMatch)
		} else {
			splitSE(in, out, filter, skipReg, rc, PrintSkipMatch)
		}
	}
}

func PrintMatch(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.MatchString(seq)
	if match {
		simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.MatchString(seq)
		if match {
			rcQual := string(util.Reverse([]byte(qual)))
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + rcSeq + "\n" + note + "\n" + rcQual + "\n")))
		}
	}
}

func PrintMatchTrailer(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.FindString(seq)
	if match != "" {
		simpleUtil.HandleError(out.Write([]byte(name + "\n" + match + "\n" + note + "\n" + qual[:len(match)] + "\n")))
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.FindString(rcSeq)
		if match != "" {
			rcQual := string(util.Reverse([]byte(qual)))
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + match + "\n" + note + "\n" + rcQual[:len(match)] + "\n")))
		}
	}
}

func PrintMatchCut(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.FindStringSubmatchIndex(seq)
	if match != nil {
		simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq[:match[2]] + "\n" + note + "\n" + qual[:match[2]] + "\n")))
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.FindStringSubmatchIndex(rcSeq)
		if match != nil {
			rcQual := string(util.Reverse([]byte(qual)))
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + rcSeq[:match[2]] + "\n" + note + "\n" + rcQual[:match[2]] + "\n")))

		}
	}
}

func PrintSkipMatch(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.MatchString(seq)
	if match {
		if !skipReg.MatchString(seq) {
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
		}
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.MatchString(rcSeq)
		if match {
			if !skipReg.MatchString(rcSeq) {
				rcQual := string(util.Reverse([]byte(qual)))
				simpleUtil.HandleError(out.Write([]byte(name + "\n" + rcSeq + "\n" + note + "\n" + rcQual + "\n")))
			}
		}
	}
}

func PrintSkipMatchTrailer(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.FindString(seq)
	if match != "" {
		if !skipReg.MatchString(seq) {
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + match + "\n" + note + "\n" + qual[:len(match)] + "\n")))
		}
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.FindString(rcSeq)
		if match != "" {
			if !skipReg.MatchString(rcSeq) {
				rcQual := string(util.Reverse([]byte(qual)))
				simpleUtil.HandleError(out.Write([]byte(name + "\n" + match + "\n" + note + "\n" + rcQual[:len(match)] + "\n")))
			}
		}
	}
}

func PrintSkipMatchCut(out io.Writer, name, seq, note, qual string, filter, skipReg *regexp.Regexp, rc bool) {
	match := filter.FindStringSubmatchIndex(seq)
	if match != nil {
		if !skipReg.MatchString(seq) {
			simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq[:match[2]] + "\n" + note + "\n" + qual[:match[2]] + "\n")))
		}
	} else if rc {
		rcSeq := util.ReverseComplement(seq)
		match := filter.FindStringSubmatchIndex(rcSeq)
		if match != nil {
			if !skipReg.MatchString(rcSeq) {
				rcQual := string(util.Reverse([]byte(qual)))
				simpleUtil.HandleError(out.Write([]byte(name + "\n" + rcSeq[:match[2]] + "\n" + note + "\n" + rcQual[:match[2]] + "\n")))
			}
		}
	}
}

func splitSE(in io.Reader, out io.Writer, filter, skipReg *regexp.Regexp, rc bool, printFQ PrintFQ) {
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
			printFQ(out, name, seq, note, qual, filter, skipReg, rc)
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}
