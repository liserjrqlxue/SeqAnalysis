package main

import (
	"flag"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
)

// flag
var (
	workDir = flag.String(
		"w",
		"",
		"workdir",
	)
)

// global
var (
	BarCode = "AGTGCT"
)

func main() {
	flag.Parse()
	if *workDir != "" {
		simpleUtil.CheckErr(os.Chdir(*workDir))
	}

	var seqList = textUtil.File2Array("input.txt")
	var out = osUtil.Create("output.txt")
	defer simpleUtil.DeferClose(out)

	for _, s := range seqList {
		strings.TrimSuffix(s, "\r")
		var a = strings.Split(s, "\t")
		a = append(a, "", "", "")

		var seqInfo = &SeqInfo{
			Seq:         []byte(a[1]),
			IndexSeq:    a[2],
			BarCode:     BarCode,
			Fastq:       a[0],
			Stats:       make(map[string]int),
			HitSeqCount: make(map[string]int),
			ReadsLength: make(map[int]int),
		}
		seqInfo.Init()
		seqInfo.CountError4()

		fmtUtil.Fprintf(out, "#######################################  Summary of %s\n", s)
		seqInfo.WriteStats(out)
		seqInfo.WriteDistributionFreq(out)
		fmtUtil.Fprint(out, "\n\n\n")

		seqInfo.PlotLineACGT("ACGT.html")
	}
}

// regexp
var (
	minus3   = regexp.MustCompile(`---`)
	plus3    = regexp.MustCompile(`\+\+\+`)
	minus1   = regexp.MustCompile(`-`)
	dup2     = regexp.MustCompile(`AA|TT|CC|GG`)
	regPolyA = regexp.MustCompile(`AAAAAAAA`)
	regN     = regexp.MustCompile(`N`)
	regA     = regexp.MustCompile(`A`)
	regC     = regexp.MustCompile(`C`)
	regG     = regexp.MustCompile(`G`)
	regT     = regexp.MustCompile(`T`)
)

type SeqInfo struct {
	Seq         []byte
	Align       []byte
	AlignInsert []byte
	AlignMut    []byte

	IndexSeq string
	BarCode  string
	Fastq    string

	HitSeq      []string
	HitSeqCount map[string]int
	Stats       map[string]int

	DistributionNum  [4][]int
	DistributionFreq [4][]float64

	// fastq
	ReadsLength map[int]int
	A           [151]int
	C           [151]int
	G           [151]int
	T           [151]int
}

func (seqInfo *SeqInfo) Init() {
	if seqInfo.Seq == nil {
		seqInfo.Seq = []byte("CTCTCTCTCTCTCTCTCTCT")
	}
	if seqInfo.IndexSeq == "" {
		seqInfo.IndexSeq = "ACTAGGACGACTCGAATT"
	}
	for i := 0; i < len(seqInfo.Seq); i++ {
		for j := 0; j < 4; j++ {
			seqInfo.DistributionNum[j] = append(seqInfo.DistributionNum[j], 0)
			seqInfo.DistributionFreq[j] = append(seqInfo.DistributionFreq[j], 0)
		}
	}
}

func (seqInfo *SeqInfo) WriteSeqResult(path string) {
	var (
		tarSeq      = string(seqInfo.Seq)
		indexSeq    = seqInfo.IndexSeq
		tarLength   = len(tarSeq) + 10
		seqHit      = regexp.MustCompile(indexSeq + tarSeq)
		polyA       = regexp.MustCompile(`(.*?)` + indexSeq + `(.*?)AAAAAAAA`)
		regIndexSeq = regexp.MustCompile(indexSeq)
		regTarSeq   = regexp.MustCompile(tarSeq)

		output        = osUtil.Create(path)
		output90      = osUtil.Create(path + ".90.txt")
		outputUnmatch = osUtil.Create(path + ".unmatch.txt")
		outputHit     = osUtil.Create(path + ".Hit.txt")
	)
	defer simpleUtil.DeferClose(output)
	defer simpleUtil.DeferClose(output90)
	defer simpleUtil.DeferClose(outputUnmatch)

	fmtUtil.Fprintf(outputUnmatch, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "#Seq", "A", "C", "G", "T", "TargetSeq", "IndexSeq", "PloyA")

	for i, s := range textUtil.File2Array(seqInfo.Fastq) {
		if i%4 != 1 {
			continue
		}
		seqInfo.ReadsLength[len(s)]++

		if len(s) <= 90 {
			seqInfo.Stats["shortReadsNum"]++
			fmtUtil.Fprintf(output90, "%s\t%d\n", s, len(s))
			continue
		}
		seqInfo.Stats["allReadsNum"]++
		var tSeq = tarSeq
		if seqHit.MatchString(s) {
			seqInfo.Stats["analyzedReadsNum"]++
			seqInfo.Stats["seqHitReadsNum"]++
			seqInfo.HitSeqCount[tSeq]++
			fmtUtil.Fprintf(output, "%s\t%s\n", tSeq, seqInfo.BarCode)
			fmtUtil.Fprintf(outputHit, "%s\t%s\t%s\t%+v\n")
			for i2, c := range []byte(s) {
				switch c {
				case 'A':
					seqInfo.A[i2]++
				case 'C':
					seqInfo.C[i2]++
				case 'G':
					seqInfo.G[i2]++
				case 'T':
					seqInfo.T[i2]++
				}
			}

		} else if polyA.MatchString(s) {
			seqInfo.Stats["analyzedReadsNum"]++
			var m = polyA.FindStringSubmatch(s)
			tSeq = m[2]

			if len(tSeq) == 0 {
				tSeq = "X"
				seqInfo.HitSeqCount[tSeq]++
			} else if len(tSeq) > 1 && !regN.MatchString(tSeq) && len(tSeq) < tarLength {
				seqInfo.HitSeqCount[tSeq]++
				fmtUtil.Fprintf(output, "%s\t%s\n", tSeq, seqInfo.BarCode)
			} else {
				seqInfo.Stats["analyzedExcludeReadsNum"]++
			}
		} else {
			fmtUtil.Fprintf(
				outputUnmatch,
				"%s\t%d\t%d\t%d\t%d\t%v\t%v\t%v\n",
				s,
				len(regA.FindAllString(s, -1)),
				len(regC.FindAllString(s, -1)),
				len(regG.FindAllString(s, -1)),
				len(regT.FindAllString(s, -1)),
				regTarSeq.MatchString(s),
				regIndexSeq.MatchString(s),
				regPolyA.MatchString(s),
			)
		}
	}
}

func (seqInfo *SeqInfo) GetHitSeq() {
	for k := range seqInfo.HitSeqCount {
		seqInfo.HitSeq = append(seqInfo.HitSeq, k)
	}
	sort.Slice(seqInfo.HitSeq, func(i, j int) bool {
		return seqInfo.HitSeqCount[seqInfo.HitSeq[i]] > seqInfo.HitSeqCount[seqInfo.HitSeq[j]]
	})
}

func (seqInfo *SeqInfo) WriteSeqResultBarCode(output *os.File) {
	for _, s := range seqInfo.HitSeq {
		fmtUtil.Fprintf(output, "%s\t%d\n", s, seqInfo.HitSeqCount[s])
	}
}

func (seqInfo *SeqInfo) WriteSeqResultNum() {
	var (
		outputDel   = osUtil.Create("SeqResult_Num_Deletion.txt")
		outputIns   = osUtil.Create("SeqResult_Num_Insertion.txt")
		outputMut   = osUtil.Create("SeqResult_Num_Mutation.txt")
		outputOther = osUtil.Create("SeqResult_Num_Other.txt")

		keys = seqInfo.HitSeq
	)
	defer simpleUtil.DeferClose(outputDel)
	defer simpleUtil.DeferClose(outputIns)
	defer simpleUtil.DeferClose(outputMut)
	defer simpleUtil.DeferClose(outputOther)

	fmtUtil.Fprintf(outputDel, "%s\t%s\t%s\t%s\n", "#TargetSeq", "SubMatchSeq", "Count", "AlignResult")
	fmtUtil.Fprintf(outputIns, "%s\t%s\t%s\t%s\n", "#TargetSeq", "SubMatchSeq", "Count", "AlignResult")
	fmtUtil.Fprintf(outputMut, "%s\t%s\t%s\t%s\n", "#TargetSeq", "SubMatchSeq", "Count", "AlignResult")
	fmtUtil.Fprintf(outputOther, "%s\t%s\t%s\t%s\t%s\t%s\n", "#TargetSeq", "SubMatchSeq", "Count", "AlignDeletion", "AlignInsertion", "AlignMutation")

	for _, key := range keys {
		if seqInfo.Align1(key, outputDel) {
			continue
		}

		if seqInfo.Align2(key, outputIns) {
			continue
		}

		if seqInfo.Align3(key, outputMut) {
			continue
		}

		fmtUtil.Fprintf(outputOther, "%s\t%s\t%d\t%s\t%s\t%s\n", seqInfo.Seq, key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut)
		seqInfo.Stats["errorOtherReadsNum"] += seqInfo.HitSeqCount[key]
	}
}

func (seqInfo *SeqInfo) UpdateDistributionStats() {
	seqInfo.Stats["errorReadsNum"] = seqInfo.Stats["errorDelReadsNum"] + seqInfo.Stats["errorInsReadsNum"] + seqInfo.Stats["errorMutReadsNum"] + seqInfo.Stats["errorOtherReadsNum"]
	seqInfo.Stats["excludeOtherReadsNum"] = seqInfo.Stats["rightReadsNum"] + seqInfo.Stats["errorReadsNum"] - seqInfo.Stats["errorOtherReadsNum"]
	seqInfo.Stats["accuReadsNum"] = seqInfo.Stats["excludeOtherReadsNum"] * len(seqInfo.Seq)

	for i := range seqInfo.Seq {
		// right reads num
		seqInfo.DistributionNum[3][i] = seqInfo.Stats["excludeOtherReadsNum"] - seqInfo.DistributionNum[0][i] - seqInfo.DistributionNum[1][i] - seqInfo.DistributionNum[2][i]
		for j := 0; j < 4; j++ {
			seqInfo.DistributionFreq[j][i] = math.DivisionInt(seqInfo.DistributionNum[j][i], seqInfo.Stats["excludeOtherReadsNum"])
		}

		seqInfo.Stats["accuRightNum"] += seqInfo.DistributionNum[3][i]
	}
}

func (seqInfo *SeqInfo) WriteDistributionNum(output *os.File) {
	var distribution = seqInfo.DistributionNum

	fmtUtil.Fprintf(output, "%s\t%s\t%s\t%s\t%s\n", "Tar", "Del", "Ins", "Mut", "Right")

	for i, b := range seqInfo.Seq {
		fmtUtil.Fprintf(output,
			"%c\t%d\t%d\t%d\t%d\n",
			b,
			distribution[0][i],
			distribution[1][i],
			distribution[2][i],
			distribution[3][i],
		)
	}
}

func (seqInfo *SeqInfo) WriteDistributionFreq(output *os.File) {
	var distribution = seqInfo.DistributionFreq

	fmtUtil.Fprintf(output, "%s\t%s\t%s\t%s\t%s\n", "Tar", "Del", "Ins", "Mut", "Right")

	for i, b := range seqInfo.Seq {
		fmtUtil.Fprintf(output,
			"%c\t%0.4f\t%0.4f\t%0.4f\t%0.4f\n",
			b,
			distribution[0][i],
			distribution[1][i],
			distribution[2][i],
			distribution[3][i],
		)
	}
}

func (seqInfo *SeqInfo) WriteStats(output *os.File) {
	var stats = seqInfo.Stats

	fmtUtil.Fprintf(output, "AllReadsNum = %d\n", stats["allReadsNum"])
	fmtUtil.Fprintf(output, "++AnalyzedReadsNum = %d\n", stats["analyzedReadsNum"])
	fmtUtil.Fprintf(
		output,
		"++++RightReadsNum = %d   Accuracy = %f,\n",
		stats["rightReadsNum"],
		math.DivisionInt(stats["rightReadsNum"], stats["analyzedReadsNum"]-stats["errorOtherReadsNum"]),
	)
	fmtUtil.Fprintf(output, "++++ErrorReadsNum = %d\n", stats["errorReadsNum"])
	fmtUtil.Fprintf(output, "++++++ErrorDelReadsNum = %d\n", stats["errorDelReadsNum"])
	fmtUtil.Fprintf(output, "++++++ErrorInsReadsNum = %d\n", stats["errorInsReadsNum"])
	fmtUtil.Fprintf(output, "++++++ErrorMutReadsNum = %d\n", stats["errorMutReadsNum"])
	fmtUtil.Fprintf(output, "++++++ErrorOtherReadsNum = %d\n", stats["errorOtherReadsNum"])
	fmtUtil.Fprintf(
		output,
		"++++++AverageBaseAccuracy = %f\n",
		math.DivisionInt(stats["accuRightNum"], stats["accuReadsNum"]),
	)
	fmtUtil.Fprint(output, "\n\n")
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4() {
	// 1. 统计不同测序结果出现的频数
	seqInfo.WriteSeqResult("SeqResult.txt")

	seqInfo.GetHitSeq()
	// SeqResult_BarCode.txt
	var seqBarCode = osUtil.Create("SeqResult_BarCode.txt")
	seqInfo.WriteSeqResultBarCode(seqBarCode)

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	seqInfo.WriteSeqResultNum()

	seqInfo.UpdateDistributionStats()

	// write distribution_Num.txt
	var disNum = osUtil.Create("distribution_Num.txt")
	defer simpleUtil.DeferClose(disNum)
	seqInfo.WriteDistributionNum(disNum)

	// distribution_Frequency.txt
	var disFrequency = osUtil.Create("distribution_Frequency.txt")
	defer simpleUtil.DeferClose(disFrequency)
	seqInfo.WriteDistributionFreq(disFrequency)

	// write Summary.txt
	var summary = osUtil.Create("Summary.txt")
	defer simpleUtil.DeferClose(summary)
	seqInfo.WriteStats(summary)
	seqInfo.WriteDistributionFreq(summary)
}

func (seqInfo *SeqInfo) Align1(key string, output *os.File) bool {
	var (
		a = seqInfo.Seq
		b = []byte(key)
		c []byte

		count = seqInfo.HitSeqCount[key]
	)

	if len(a) == 1 && len(b) == 1 && b[0] == 'X' {
		c = append(c, '-')
		seqInfo.Align = c
		seqInfo.DistributionNum[0][0] += count
		if string(c) == string(seqInfo.Seq) {
			seqInfo.Stats["rightReadsNum"] += count
		} else {
			seqInfo.Stats["errorDelReadsNum"] += count
		}
		return true
	}

	var k = 0 // match count to Seq
	for i := range a {
		if k < len(b) && a[i] == b[k] {
			c = append(c, b[k])
			k++
		} else {
			c = append(c, '-')
		}
	}
	seqInfo.Align = c
	if k >= len(b) { // all match
		fmtUtil.Fprintf(output, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
		for i, c1 := range c {
			if c1 == '-' {
				seqInfo.DistributionNum[0][i] += count
			}
		}
		if string(c) == string(seqInfo.Seq) {
			seqInfo.Stats["rightReadsNum"] += count
		} else {
			seqInfo.Stats["errorDelReadsNum"] += count
		}
		return true
	}
	return false
}

func (seqInfo *SeqInfo) Align2(key string, output *os.File) bool {
	var (
		a      = seqInfo.Seq
		b      = []byte(key)
		c      []byte
		k      = 0
		maxLen = len(a)

		count = seqInfo.HitSeqCount[key]
	)

	if len(b) > maxLen {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		if k < maxLen || i < len(a) {
			if i < len(a) && k < len(b) && a[i] == b[k] { // match to Seq
				c = append(c, b[k])
				k += 1
			} else if i > 0 && i <= len(a) && k < len(b) && a[i-1] == b[k] { // match to Seq -1 bp
				c = append(c, '+')
				k += 1
				i--
				/*
					} else if i < len(a) && k < len(b)-1 && a[i] == b[k+1] { // match to next
						c = append(c, '+', b[k+1])
						k += 2
					} else if i < len(a) && k < len(b)-2 && a[i] == b[k+2] { // match to next 2
						c = append(c, '+', '+', b[k+2])
						k += 3
				*/
			} else {
				c = append(c, '-')
			}
		}
	}
	seqInfo.AlignInsert = c
	if k >= len(b)-1 && c[0] != '+' {
		if !minus3.Match(c) && !plus3.Match(c) && !minus1.Match(c) {
			fmtUtil.Fprintf(output, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			seqInfo.Stats["errorInsReadsNum"] += count
			var i = 0
			for _, c1 := range c[1:] {
				if c1 == '+' {
					seqInfo.DistributionNum[1][i] += count
				} else {
					i++
				}
			}
			return true
		}
	}
	return false
}

func (seqInfo *SeqInfo) Align3(key string, output *os.File) bool {
	var (
		a = seqInfo.Seq
		b = []byte(key)
		c []byte
		k = 0

		count = seqInfo.HitSeqCount[key]
	)

	if len(a) == len(b) {
		for i, s := range a {
			if i < len(b) && s == b[i] {
				c = append(c, s)
			} else {
				k++
				c = append(c, 'X')
			}
		}
	}
	seqInfo.AlignMut = c
	if k < 2 && len(c) > 0 {
		if !dup2.MatchString(key) {
			fmtUtil.Fprintf(output, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			seqInfo.Stats["errorMutReadsNum"] += count
			for i, c1 := range c {
				if c1 == 'X' {
					seqInfo.DistributionNum[2][i] += count
				}
			}
			return true
		}
	}
	return false
}

func (seqInfo *SeqInfo) PlotLineACGT(path string) {
	var (
		line   = charts.NewLine()
		xaxis  [151]int
		output = osUtil.Create(path)
	)
	defer simpleUtil.DeferClose(output)
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "A C G T Distribution",
			Subtitle: "in SE150",
		}))

	for i := 0; i < 151; i++ {
		xaxis[i] = i + 1
	}

	line.SetXAxis(xaxis).
		AddSeries("A", generateLineItems(seqInfo.A[:])).
		AddSeries("C", generateLineItems(seqInfo.C[:])).
		AddSeries("G", generateLineItems(seqInfo.G[:])).
		AddSeries("T", generateLineItems(seqInfo.T[:]))
	// SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	simpleUtil.CheckErr(line.Render(output))
}

func generateLineItems(vs []int) []opts.LineData {
	var items = make([]opts.LineData, 0)
	for _, v := range vs {
		items = append(items, opts.LineData{Value: v})
	}
	return items
}
