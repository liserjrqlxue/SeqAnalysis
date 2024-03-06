package seqAnalysis

import (
	"PrimerDesigner/util"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

// from https://forum.golangbridge.org/t/easy-way-for-letter-substitution-reverse-complementary-dna-sequence/20101
// from https://go.dev/play/p/IXI6PY7XUXN
var dnaComplement = strings.NewReplacer(
	"A", "T",
	"T", "A",
	"G", "C",
	"C", "G",
	"a", "t",
	"t", "a",
	"g", "c",
	"c", "g",
)

func Complement(s string) string {
	return dnaComplement.Replace(s)
}

// Reverse returns its argument string reversed rune-wise left to right.
// from https://github.com/golang/example/blob/master/stringmain/reverse.go
func Reverse(r []byte) []byte {
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func ReverseComplement(s string) string {
	return Complement(string(Reverse([]byte(s))))
}

// WriteHistogram sort hist and write to path with title [length weight]
func WriteHistogram(path string, hist map[int]int) {
	out := osUtil.Create(path)
	fmtUtil.Fprintln(out, "length\tweight")
	var seqLengths []int
	for k := range hist {
		seqLengths = append(seqLengths, k)
	}
	sort.Ints(seqLengths)
	for _, k := range seqLengths {
		fmtUtil.Fprintf(out, "%d\t%d\n", k, hist[k])
	}
	simpleUtil.CheckErr(out.Close())
}

func MatchSeq(seq string, polyA, regIndexSeq *regexp.Regexp, useRC, assemblerMode bool) (submatch []string, byteS []byte, indexSeqMatch bool) {
	var (
		seqRC string

		regIndexSeqMatch   bool
		regIndexSeqRcMatch bool
	)
	if useRC {
		seqRC = util.ReverseComplement(seq)
	}

	submatch = polyA.FindStringSubmatch(seq)
	if submatch != nil { // SubMatch -> regIndexSeqMatch
		regIndexSeqMatch = true
	} else { // A尾不匹配
		if useRC { // RC时考虑RC的A尾SubMatch
			submatch = polyA.FindStringSubmatch(seqRC)
			if submatch != nil { // SubMatch -> regIndexSeqRcMatch
				regIndexSeqRcMatch = true
			}
		}

		if submatch == nil { // A尾不匹配
			if assemblerMode { // AseemblerMode 时 考虑靶标SubMatch
				submatch = regIndexSeq.FindStringSubmatch(seq)
				if submatch != nil { // SubMatch -> regIndexSeqMatch
					regIndexSeqMatch = true
				} else if useRC { // RC时考虑RC的靶标SubMatch
					submatch = regIndexSeq.FindStringSubmatch(seqRC)
					if submatch != nil { // SubMatch -> regIndexSeqRcMatch
						regIndexSeqRcMatch = true
					}
				}
			} else { // 非AseemblerMode 时 考虑靶标Match
				regIndexSeqMatch = regIndexSeq.MatchString(seq)
				if !regIndexSeqMatch && useRC {
					regIndexSeqRcMatch = regIndexSeq.MatchString(seqRC)
				}
			}
		} else { // RC的A尾SubMatch, 考察靶标Match
			regIndexSeqMatch = regIndexSeq.MatchString(seq)
		}
	}

	if regIndexSeqMatch {
		byteS = []byte(seq)
		indexSeqMatch = true
	} else if regIndexSeqRcMatch {
		byteS = []byte(seqRC)
		indexSeqMatch = true
	}
	return
}

// write upper and down
func WriteUpperDown(out *os.File, indexSeq, refSeq string, offset, count int, m [][]int) {
	refSeq = indexSeq[len(indexSeq)-offset:] + refSeq
	for i := range m {
		var end = m[i][0]
		fmtUtil.Fprintf(out, "%d\t%d\t%s\t%s\n", end, count, refSeq[end:end+offset], refSeq[end+offset:end+offset*2])
	}
}

func WriteUpperDownNIL(out *os.File, indexSeq, refSeq string, offset int) {
	// fill with 0
	refSeq = indexSeq[len(indexSeq)-offset:] + refSeq
	var n = len(refSeq) - offset*2
	for end := 0; end <= n; end++ {
		fmtUtil.Fprintf(out, "%d\t%d\t%s\t%s\n", end, 0, refSeq[end:end+offset], refSeq[end+offset:end+offset*2])
	}
}
