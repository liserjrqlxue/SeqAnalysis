package main

import (
	"log"
	"math"
	"os"
	"regexp"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	math2 "github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

type SeqInfo struct {
	Name  string
	Excel string

	xlsx         *excelize.File
	Sheets       []string
	Style        map[string]int
	rowDeletion  int
	rowDeletion1 int
	rowDeletion2 int
	rowDeletion3 int

	Seq         []byte
	Align       []byte
	AlignInsert []byte
	AlignMut    []byte

	IndexSeq string
	BarCode  string
	Fastqs   []string

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

var center = &excelize.Style{
	Alignment: &excelize.Alignment{
		Horizontal: "center",
	},
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

	seqInfo.Excel = seqInfo.Name + ".xlsx"
	seqInfo.xlsx = excelize.NewFile()
	seqInfo.Style = make(map[string]int)
	seqInfo.Style["center"] = simpleUtil.HandleError(seqInfo.xlsx.NewStyle(center)).(int)

	seqInfo.Sheets = []string{
		"Sheet",
		"SeqResult",
		"BarCode",
		"Deletion",
		"DeletionSingle",
		"DeletionDouble",
		"DeletionOther",
	}
	seqInfo.rowDeletion = 1
	seqInfo.rowDeletion1 = 1
	seqInfo.rowDeletion2 = 1
	seqInfo.rowDeletion3 = 1
	for i, sheet := range seqInfo.Sheets {
		if i == 0 {
			simpleUtil.CheckErr(seqInfo.xlsx.SetSheetName("Sheet1", sheet))
		} else {
			simpleUtil.HandleError(seqInfo.xlsx.NewSheet(sheet))
		}
	}

	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[0], "A", "A", 20))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[0], "M", "Q", 12))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[0], "R", "R", 14))

	simpleUtil.CheckErr(seqInfo.xlsx.SetRowStyle(seqInfo.Sheets[0], 1, 18, seqInfo.Style["center"]))
	SetCellStr(seqInfo.xlsx, seqInfo.Sheets[0], 1, 1, seqInfo.Name)
	simpleUtil.CheckErr(seqInfo.xlsx.MergeCell(seqInfo.Sheets[0], "A1", "R1"))
}

func (seqInfo *SeqInfo) Save() {
	log.Printf("seqInfo.xlsx.SaveAs(%s)", seqInfo.Excel)

	simpleUtil.CheckErr(seqInfo.xlsx.SaveAs(seqInfo.Excel))
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4() {
	// 1. 统计不同测序结果出现的频数
	log.Print("seqInfo.WriteSeqResult")
	seqInfo.WriteSeqResult("SeqResult.txt")

	log.Print("seqInfo.GetHitSeq")
	seqInfo.GetHitSeq()
	// SeqResult_BarCode.txt
	var seqBarCode = osUtil.Create("SeqResult_BarCode.txt")
	log.Print("seqInfo.WriteSeqResultBarCode")
	seqInfo.WriteSeqResultBarCode(seqBarCode)

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	log.Print("seqInfo.WriteSeqResultNum")
	seqInfo.WriteSeqResultNum()

	log.Print("seqInfo.UpdateDistributionStats")
	seqInfo.UpdateDistributionStats()

	// write distribution_Num.txt
	var disNum = osUtil.Create("distribution_Num.txt")
	defer simpleUtil.DeferClose(disNum)
	log.Print("seqInfo.WriteDistributionNum")
	seqInfo.WriteDistributionNum(disNum)

	// distribution_Frequency.txt
	var disFrequency = osUtil.Create("distribution_Frequency.txt")
	defer simpleUtil.DeferClose(disFrequency)
	log.Print("seqInfo.WriteDistributionFreq")
	seqInfo.WriteDistributionFreq(disFrequency)

	// write Summary.txt
	var summary = osUtil.Create("Summary.txt")
	defer simpleUtil.DeferClose(summary)
	log.Print("write Summary.txt")
	seqInfo.WriteStats(summary)
	seqInfo.WriteDistributionFreq(summary)
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
	)
	defer simpleUtil.DeferClose(output)
	defer simpleUtil.DeferClose(output90)
	defer simpleUtil.DeferClose(outputUnmatch)

	fmtUtil.Fprintf(outputUnmatch, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "#Seq", "A", "C", "G", "T", "TargetSeq", "IndexSeq", "PloyA")

	var row = 1
	for _, fastq := range seqInfo.Fastqs {
		log.Printf("load %s", fastq)
		for i, s := range textUtil.File2Array(fastq) {
			if i%4 != 1 {
				continue
			}
			seqInfo.ReadsLength[len(s)]++

			seqInfo.Stats["allReadsNum"]++
			if len(s) <= 90 {
				seqInfo.Stats["shortReadsNum"]++
				fmtUtil.Fprintf(output90, "%s\t%d\n", s, len(s))
				continue
			}
			var tSeq = tarSeq
			if seqHit.MatchString(s) {
				seqInfo.Stats["seqHitReadsNum"]++
				seqInfo.HitSeqCount[tSeq]++
				fmtUtil.Fprintf(output, "%s\t%s\n", tSeq, seqInfo.BarCode)
				SetRow(seqInfo.xlsx, seqInfo.Sheets[1], 1, row, []interface{}{tSeq, seqInfo.BarCode})
				row++
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
					seqInfo.Stats["indexPolyAReadsNum"]++
				} else if len(tSeq) > 1 && !regN.MatchString(tSeq) && len(tSeq) < tarLength {
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["indexPolyAReadsNum"]++
					fmtUtil.Fprintf(output, "%s\t%s\n", tSeq, seqInfo.BarCode)
					SetRow(seqInfo.xlsx, seqInfo.Sheets[1], 1, row, []interface{}{tSeq, seqInfo.BarCode})
					row++
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
	seqInfo.Stats["analyzedReadsNum"] = seqInfo.Stats["seqHitReadsNum"] + seqInfo.Stats["indexPolyAReadsNum"] + seqInfo.Stats["analyzedExcludeReadsNum"]
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
	for i, s := range seqInfo.HitSeq {
		fmtUtil.Fprintf(output, "%s\t%d\n", s, seqInfo.HitSeqCount[s])
		SetRow(seqInfo.xlsx, seqInfo.Sheets[2], 1, i+1, []interface{}{s, seqInfo.HitSeqCount[s]})
	}
}

func (seqInfo *SeqInfo) WriteSeqResultNum() {
	var (
		outputDel    = osUtil.Create("SeqResult_Num_Deletion.txt")
		outputDel1   = osUtil.Create("SeqResult_Num_Deletion1.txt")
		outputDel2   = osUtil.Create("SeqResult_Num_Deletion2.txt")
		outputDel3   = osUtil.Create("SeqResult_Num_Deletion3.txt")
		outputIns    = osUtil.Create("SeqResult_Num_Insertion.txt")
		outputInsDel = osUtil.Create("SeqResult_Num_InsertionDeletion.txt")
		outputMut    = osUtil.Create("SeqResult_Num_Mutation.txt")
		outputOther  = osUtil.Create("SeqResult_Num_Other.txt")

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
		if key == string(seqInfo.Seq) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets[3], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key]})
			seqInfo.rowDeletion++
			continue
		}
		if seqInfo.Align1(key, outputDel, outputDel1, outputDel2, outputDel3) {
			continue
		}

		if seqInfo.Align2(key, outputIns, outputInsDel) {
			continue
		}

		if seqInfo.Align3(key, outputMut) {
			continue
		}

		fmtUtil.Fprintf(outputOther, "%s\t%s\t%d\t%s\t%s\t%s\n", seqInfo.Seq, key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut)
		seqInfo.Stats["errorOtherReadsNum"] += seqInfo.HitSeqCount[key]
	}
}

func (seqInfo *SeqInfo) Align1(key string, output ...*os.File) bool {
	var (
		a = seqInfo.Seq
		b = []byte(key)
		c []byte

		count    = seqInfo.HitSeqCount[key]
		delCount = 0
	)

	if len(a) == 1 && len(b) == 1 && b[0] == 'X' {
		c = append(c, '-')
		seqInfo.Align = c
		seqInfo.DistributionNum[0][0] += count
		seqInfo.Stats["errorDelReadsNum"] += count
		return true
	}

	var k = 0 // match count to Seq
	for i := range a {
		if k < len(b) && a[i] == b[k] {
			c = append(c, b[k])
			k++
		} else {
			c = append(c, '-')
			delCount++
		}
	}
	seqInfo.Align = c
	if k >= len(b) { // all match
		fmtUtil.Fprintf(output[0], "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
		SetRow(seqInfo.xlsx, seqInfo.Sheets[3], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.rowDeletion++
		if delCount == 1 {
			fmtUtil.Fprintf(output[1], "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			SetRow(seqInfo.xlsx, seqInfo.Sheets[4], 1, seqInfo.rowDeletion1, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.rowDeletion1++
		} else if delCount == 2 {
			fmtUtil.Fprintf(output[2], "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			SetRow(seqInfo.xlsx, seqInfo.Sheets[5], 1, seqInfo.rowDeletion2, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.rowDeletion2++
		} else if delCount >= 3 {
			fmtUtil.Fprintf(output[3], "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			SetRow(seqInfo.xlsx, seqInfo.Sheets[6], 1, seqInfo.rowDeletion3, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.rowDeletion3++
		}
		for i, c1 := range c {
			if c1 == '-' {
				seqInfo.DistributionNum[0][i] += count
			}
		}
		seqInfo.Stats["errorDelReadsNum"] += count
		return true
	}
	return false
}

func (seqInfo *SeqInfo) Align2(key string, outputIns, outputInsDel *os.File) bool {
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
		if !plus3.Match(c) {
			if minus1.Match(c) {
				fmtUtil.Fprintf(outputInsDel, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			} else {
				fmtUtil.Fprintf(outputIns, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
			}
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
		fmtUtil.Fprintf(output, "%s\t%s\t%d\t%s\n", seqInfo.Seq, key, count, c)
		seqInfo.Stats["errorMutReadsNum"] += count
		for i, c1 := range c {
			if c1 == 'X' {
				seqInfo.DistributionNum[2][i] += count
			}
		}
		return true
	}
	return false
}

func (seqInfo *SeqInfo) UpdateDistributionStats() {
	seqInfo.Stats["errorReadsNum"] = seqInfo.Stats["errorDelReadsNum"] + seqInfo.Stats["errorInsReadsNum"] + seqInfo.Stats["errorMutReadsNum"] + seqInfo.Stats["errorOtherReadsNum"]
	seqInfo.Stats["excludeOtherReadsNum"] = seqInfo.Stats["seqHitReadsNum"] + seqInfo.Stats["errorReadsNum"] - seqInfo.Stats["errorOtherReadsNum"]
	seqInfo.Stats["accuReadsNum"] = seqInfo.Stats["excludeOtherReadsNum"] * len(seqInfo.Seq)

	for i := range seqInfo.Seq {
		// right reads num
		seqInfo.DistributionNum[3][i] = seqInfo.Stats["excludeOtherReadsNum"] - seqInfo.DistributionNum[0][i] - seqInfo.DistributionNum[1][i] - seqInfo.DistributionNum[2][i]
		for j := 0; j < 4; j++ {
			seqInfo.DistributionFreq[j][i] = math2.DivisionInt(seqInfo.DistributionNum[j][i], seqInfo.Stats["excludeOtherReadsNum"])
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

func (seqInfo *SeqInfo) WriteStats(output *os.File) {
	var stats = seqInfo.Stats

	fmtUtil.Fprintf(
		output,
		"AllReadsNum\t\t\t\t= %d\n",
		stats["allReadsNum"],
	)
	fmtUtil.Fprintf(
		output,
		"+ShortReadsNum\t\t\t= %d\t%7.4f%%)\n",
		stats["shortReadsNum"],
		math2.DivisionInt(stats["shortReadsNum"], stats["allReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"+AnalyzedReadsNum\t\t= %d\t%.4f%%\n",
		stats["analyzedReadsNum"],
		math2.DivisionInt(stats["analyzedReadsNum"], stats["allReadsNum"]-stats["shortReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"++ExcludeReadsNum\t\t= %d\t%7.4f%%\n",
		stats["analyzedExcludeReadsNum"],
		math2.DivisionInt(stats["analyzedExcludeReadsNum"], stats["analyzedReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"++SeqHitReadsNum\t\t= %d\t%.4f%%\tAccuracy = %.4f%%,\n",
		stats["seqHitReadsNum"],
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"])*100,
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"]-stats["errorOtherReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"++IndexPolyAReadsNum\t= %d\t%.4f%%\n",
		stats["indexPolyAReadsNum"],
		math2.DivisionInt(stats["indexPolyAReadsNum"], stats["analyzedReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"+++ErrorReadsNum\t\t= %d\n",
		stats["errorReadsNum"],
	)
	fmtUtil.Fprintf(output,
		"++++ErrorDelReadsNum\t= %d\t%.4f%%\n",
		stats["errorDelReadsNum"],
		math2.DivisionInt(stats["errorDelReadsNum"], stats["errorReadsNum"])*100,
	)
	fmtUtil.Fprintf(output,
		"++++ErrorInsReadsNum\t= %d\t%.4f%%\n",
		stats["errorInsReadsNum"],
		math2.DivisionInt(stats["errorInsReadsNum"], stats["errorReadsNum"])*100,
	)
	fmtUtil.Fprintf(output,
		"++++ErrorMutReadsNum\t= %d\t%7.4f%%\n",
		stats["errorMutReadsNum"],
		math2.DivisionInt(stats["errorMutReadsNum"], stats["errorReadsNum"])*100,
	)
	fmtUtil.Fprintf(output,
		"++++ErrorOtherReadsNum\t= %d\t%.4f%%\n",
		stats["errorOtherReadsNum"],
		math2.DivisionInt(stats["errorOtherReadsNum"], stats["errorReadsNum"])*100,
	)
	fmtUtil.Fprintf(
		output,
		"++AverageBaseAccuracy\t= %7.4f%%\t%d/%d\n",
		math2.DivisionInt(stats["accuRightNum"], stats["accuReadsNum"])*100,
		stats["accuRightNum"], stats["accuReadsNum"],
	)
	fmtUtil.Fprint(output, "\n\n")
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

func (seqInfo *SeqInfo) WriteExcel() {
	var stats = seqInfo.Stats
	var xlsx = seqInfo.xlsx

	var sheet = seqInfo.Sheets[0]
	var col1_2 = []interface{}{
		"AllReadsNum",
		"AnalyzedReadsNum",
		"靶标",
		"合成序列",
		"RightReadsNum",
		"Accuracy",
		"ErrorReadsNum",
		"ErrorDelReadsNum",
		"ErrorInsReadsNum",
		"ErrorMutReadsNum",
		"ErrorOtherReadsNum",
		"AverageBaseAccuracy",
	}
	var col2_2 = []interface{}{
		stats["allReadsNum"],
		stats["analyzedReadsNum"],
		seqInfo.IndexSeq,
		string(seqInfo.Seq),
		stats["seqHitReadsNum"],
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"]-stats["errorOtherReadsNum"]),
		stats["errorReadsNum"],
		stats["errorDelReadsNum"],
		stats["errorInsReadsNum"],
		stats["errorMutReadsNum"],
		stats["errorOtherReadsNum"],
		math2.DivisionInt(stats["accuRightNum"], stats["accuReadsNum"]),
	}
	SetCol(xlsx, sheet, 1, 2, col1_2)
	SetCol(xlsx, sheet, 2, 2, col2_2)

	for i := range col2_2 {
		MergeCells(seqInfo.xlsx, seqInfo.Sheets[0], 2, i+2, 18, i+2)
	}

	var row = len(col1_2) + 2
	var row1_14 = []interface{}{
		"Tar", "Del", "Ins", "Mut", "Right", "readsCount", "A", "T", "C", "G", "-", "收率", "单步准确率A", "单步准确率T", "单步准确率C", "单步准确率G", "单步准确率", "收率平均准确率",
	}
	SetRow(xlsx, sheet, 1, row, row1_14)
	row++
	var distribution = seqInfo.DistributionFreq
	var readsCount = stats["analyzedReadsNum"]
	for i, b := range seqInfo.Seq {
		var counts = make(map[byte]int)
		for seq, count := range seqInfo.HitSeqCount {
			if len(seq) <= i {
				delete(seqInfo.HitSeqCount, seq)
				continue
			}
			var c = seq[i]
			counts[c] += count
			if c != b {
				delete(seqInfo.HitSeqCount, seq)
			}
		}
		var del = readsCount - counts['A'] - counts['C'] - counts['G'] - counts['T']
		var yieldCoefficient = math2.DivisionInt(counts[b], stats["analyzedReadsNum"])
		var rows = []interface{}{
			string(b),
			distribution[0][i],
			distribution[1][i],
			distribution[2][i],
			distribution[3][i],
			readsCount,
			counts['A'],
			counts['C'],
			counts['G'],
			counts['T'],
			del,
			yieldCoefficient, // 收率
			math2.DivisionInt(counts['A'], readsCount),
			math2.DivisionInt(counts['C'], readsCount),
			math2.DivisionInt(counts['G'], readsCount),
			math2.DivisionInt(counts['T'], readsCount),
			math2.DivisionInt(counts[b], readsCount),     // 单步准确率
			math.Pow(yieldCoefficient, 1.0/float64(i+1)), // 收率平均准确率
		}
		readsCount = counts[b]
		SetRow(xlsx, sheet, 1, row, rows)
		row++
	}
	simpleUtil.CheckErr(seqInfo.xlsx.SetRowStyle(seqInfo.Sheets[0], 1, row, seqInfo.Style["center"]))
}
