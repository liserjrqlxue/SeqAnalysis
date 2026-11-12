package main

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"log"
	"log/slog"
	"strings"

	gzip "github.com/klauspost/pgzip"

	"github.com/liserjrqlxue/DNA/pkg/util"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	input = flag.String(
		"i",
		"",
		"input fq.gz, multiple files separated by \",\"",
	)
	output = flag.String(
		"o",
		"",
		"output fq.gz",
	)
	barcode = flag.String(
		"b",
		"",
		"barcode; may appear anywhere in the read, the last (3'-most) occurrence is used",
	)
	tail = flag.String(
		"e",
		"",
		"trailer/tail seq, searched after the barcode",
	)
	cut = flag.Bool(
		"cut",
		false,
		"cut at the trailer start instead of keeping the trailer",
	)
	skip = flag.String(
		"skip",
		"",
		"seq right after the barcode; reads starting with barcode+skip are dropped",
	)
	rc = flag.Bool(
		"rc",
		false,
		"also try the reverse complement of each unmatched read",
	)
	minLen = flag.Int(
		"minLen",
		0,
		"min length threshold (length to tail), reads shorter than this will not be output",
	)
)

var (
	headerB []byte
	tailB   []byte
	skipB   []byte
	hasTail bool
	hasCut  bool
	hasSkip bool
	hasRC   bool
)

func main() {
	flag.Parse()
	if *input == "" || *output == "" {
		flag.PrintDefaults()
		log.Fatal("-i/-o required!")
	}
	if *barcode == "" && *tail == "" {
		flag.PrintDefaults()
		log.Fatal("at least one of -b (barcode) or -e (tail) is required!")
	}

	headerB = []byte(strings.ToUpper(*barcode))
	tailB = []byte(strings.ToUpper(*tail))
	hasTail = *tail != ""
	hasCut = *cut
	hasSkip = *skip != ""
	hasRC = *rc

	if hasSkip {
		if *barcode != "" {
			skipB = append(skipB, headerB...)
		}
		skipB = append(skipB, []byte(strings.ToUpper(*skip))...)
	}

	if *barcode == "" {
		log.Printf("barcode is empty, using tail-only mode")
	}
	if hasCut && !hasTail {
		log.Printf("empty trailer, not cut!")
		hasCut = false
	}

	inList := strings.Split(*input, ",")
	outF := osUtil.Create(*output)
	gw := gzip.NewWriter(outF)

	defer simpleUtil.DeferClose(outF)
	defer simpleUtil.DeferClose(gw)

	for _, in := range inList {
		inF := osUtil.Open(in)
		gr := simpleUtil.HandleError(gzip.NewReader(inF))
		log.Printf("split %s", in)
		splitSE(gr, gw, SwithPrintFunc(hasSkip))
		simpleUtil.DeferClose(gr)
		simpleUtil.DeferClose(inF)
	}
}

type PrintFQ func(out io.Writer, name, seq, note, qual string)

// SwithPrintFunc picks the writer: whether -skip has to be honoured is the only
// thing that changes the output; -e/-cut only decide how a read is truncated,
// which findMatch already handles from the flags.
func SwithPrintFunc(hasSkip bool) PrintFQ {
	if hasSkip {
		return PrintSkipMatch
	}
	return PrintMatch
}

// matchOne applies the matching rules to s:
//   - barcode (-b) may appear anywhere in s, it is NOT required to be a prefix.
//     The last occurrence (the one closest to the 3' end) is taken as the barcode,
//     so random prefix bases/adapter before the barcode are tolerated.
//   - trailer (-e) is the first occurrence located after the barcode; anything
//     before the barcode is kept as well, i.e. the output is always a prefix of s.
//
// ok reports whether s matched; dist is the offset of the trailer start (or len(s)
// when no trailer is given) and is used by the -minLen threshold.
func matchOne(s string) (outputSeq string, ok bool, dist int) {
	start := 0
	if len(headerB) > 0 {
		headerPos := bytes.LastIndex([]byte(s), headerB)
		if headerPos < 0 {
			return "", false, 0
		}
		start = headerPos + len(headerB)
	}
	if !hasTail {
		return s, true, len(s)
	}
	idx := bytes.Index([]byte(s[start:]), tailB)
	if idx < 0 {
		return "", false, 0
	}
	tailPos := start + idx
	if hasCut {
		return s[:tailPos], true, tailPos
	}
	return s[:tailPos+len(tailB)], true, tailPos
}

// findMatch matches the read on the forward strand first, then optionally on its
// reverse complement (-rc). A read sequenced from the opposite end is the reverse
// complement of the expected layout, so reverse complementing it restores the
// normal layout; the very same rules therefore apply, using the plain barcode and
// the plain trailer rather than their reverse complements.
//
// Returns: outputSeq (full or truncated), found, isRC, distToTail (for -minLen).
// The output keeps the original letter case of the read: only lengths are taken
// from the matched (upper-cased) copy.
func findMatch(seq string) (outputSeq string, found bool, isRC bool, distToTail int) {
	if out, ok, dist := matchOne(strings.ToUpper(seq)); ok {
		return seq[:len(out)], true, false, dist
	}

	if hasRC {
		rcSeq := util.ReverseComplement(seq)
		if out, ok, dist := matchOne(strings.ToUpper(rcSeq)); ok {
			return rcSeq[:len(out)], true, true, dist
		}
	}

	return "", false, false, 0
}

// shouldSkip reports whether the sequence right after the barcode is the -skip one.
// seq is the matched output, which may still carry bases in front of the barcode,
// so the barcode is located first with the same rule as the matcher (last
// occurrence) and the check starts right behind it.
func shouldSkip(seq string) bool {
	if !hasSkip {
		return false
	}
	seqBytes := []byte(strings.ToUpper(seq))
	pos := 0
	if len(headerB) > 0 {
		pos = bytes.LastIndex(seqBytes, headerB)
		if pos < 0 {
			return false
		}
	}
	return bytes.HasPrefix(seqBytes[pos:], skipB)
}

func writeFQ(out io.Writer, name, seq, note, qual string) {
	// seq may be truncated by -e/-cut, so qual must follow the output length,
	// otherwise the emitted record is not a valid FASTQ record.
	if len(qual) > len(seq) {
		qual = qual[:len(seq)]
	}
	simpleUtil.HandleError(out.Write([]byte(name + "\n" + seq + "\n" + note + "\n" + qual + "\n")))
}

func writeFQRC(out io.Writer, name, seq, note, qual string) {
	// Reverse the whole qual first, then let writeFQ truncate it: for a truncated
	// rc output the kept bases are the last len(seq) bases of the read.
	rcQual := string(util.Reverse([]byte(qual)))
	writeFQ(out, name, seq, note, rcQual)
}

// emit is the single writer for all dispatch variants: it matches the read, then
// applies the -skip and -minLen filters and writes the surviving record.
func emit(out io.Writer, name, seq, note, qual string, skip bool) {
	outputSeq, found, isRC, dist := findMatch(seq)
	if !found {
		return
	}
	if skip && shouldSkip(outputSeq) {
		return
	}
	if dist < *minLen {
		return
	}
	if isRC {
		writeFQRC(out, name, outputSeq, note, qual)
		return
	}
	writeFQ(out, name, outputSeq, note, qual)
}

func PrintMatch(out io.Writer, name, seq, note, qual string) {
	emit(out, name, seq, note, qual, false)
}

func PrintSkipMatch(out io.Writer, name, seq, note, qual string) {
	emit(out, name, seq, note, qual, true)
}

func splitSE(in io.Reader, out io.Writer, printFQ PrintFQ) {
	var (
		n    = 0
		name string
		seq  string
		note string
		qual string

		scanner = bufio.NewScanner(in)
	)

	for scanner.Scan() {
		line := scanner.Text()
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
			printFQ(out, name, seq, note, qual)
		}
	}
	slog.Info("finish", "n", n, "reads", n/4)
	simpleUtil.CheckErr(scanner.Err())
}
