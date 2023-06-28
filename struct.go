package main

import (
	"fmt"
	"log"
	"math"
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

	xlsx   *excelize.File
	Sheets []string
	Style  map[string]int

	rowDeletion          int
	rowDeletion1         int
	rowDeletion2         int
	rowDeletion3         int
	rowInsertion         int
	rowInsertionDeletion int
	rowMutation          int
	rowOther             int

	countDeletion  int
	countDeletion1 int
	countDeletion2 int
	countDeletion3 int

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
		"Insertion",
		"InsertionDeletion",
		"Mutation",
		"Other",
	}
	seqInfo.rowDeletion = 2
	seqInfo.rowDeletion1 = 2
	seqInfo.rowDeletion2 = 2
	seqInfo.rowDeletion3 = 2
	seqInfo.rowInsertion = 2
	seqInfo.rowInsertionDeletion = 2
	seqInfo.rowMutation = 2
	seqInfo.rowOther = 2
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
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[1], "A", "A", 25))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[2], "A", "A", 25))

	simpleUtil.CheckErr(seqInfo.xlsx.SetRowStyle(seqInfo.Sheets[0], 1, 18, seqInfo.Style["center"]))
	SetCellStr(seqInfo.xlsx, seqInfo.Sheets[0], 1, 1, seqInfo.Name)
	simpleUtil.CheckErr(seqInfo.xlsx.MergeCell(seqInfo.Sheets[0], "A1", "R1"))

	for i := 3; i < 10; i++ {
		SetRow(seqInfo.xlsx, seqInfo.Sheets[i], 1, 1, []interface{}{"#TargetSeq", "SubMatchSeq", "Count", "AlignResult"})
		simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[i], "A", "D", 25))
	}
	SetRow(seqInfo.xlsx, seqInfo.Sheets[10], 1, 1, []interface{}{"#TargetSeq", "SubMatchSeq", "Count", "AlignDeletion", "AlignInsertion", "AlignMutation"})
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[10], "A", "F", 25))
}

func (seqInfo *SeqInfo) Save() {
	log.Printf("seqInfo.xlsx.SaveAs(%s)", seqInfo.Excel)

	simpleUtil.CheckErr(seqInfo.xlsx.SaveAs(seqInfo.Excel))
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4() {
	// 1. 统计不同测序结果出现的频数
	seqInfo.WriteSeqResult(".SeqResult.txt")

	log.Print("seqInfo.GetHitSeq")
	seqInfo.GetHitSeq()
	seqInfo.SetBarCode()

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	log.Print("seqInfo.WriteSeqResultNum")
	seqInfo.WriteSeqResultNum()

	log.Print("seqInfo.UpdateDistributionStats")
	seqInfo.UpdateDistributionStats()

	//seqInfo.WriteStats()
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

		outputShort     = osUtil.Create(seqInfo.Name + path + ".short.txt")
		outputUnmatched = osUtil.Create(seqInfo.Name + path + ".unmatched.txt")
	)
	defer simpleUtil.DeferClose(outputShort)
	defer simpleUtil.DeferClose(outputUnmatched)

	fmtUtil.Fprintf(outputUnmatched, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "#Seq", "A", "C", "G", "T", "TargetSeq", "IndexSeq", "PloyA")

	var row = 1
	for _, fastq := range seqInfo.Fastqs {
		log.Printf("load %s", fastq)
		for i, s := range textUtil.File2Array(fastq) {
			if i%4 != 1 {
				continue
			}
			seqInfo.ReadsLength[len(s)]++

			seqInfo.Stats["allReadsNum"]++
			if len(s) < 50 {
				seqInfo.Stats["shortReadsNum"]++
				fmtUtil.Fprintf(outputShort, "%s\t%d\n", s, len(s))
				continue
			}
			var tSeq = tarSeq
			if seqHit.MatchString(s) {
				seqInfo.Stats["seqHitReadsNum"]++
				seqInfo.HitSeqCount[tSeq]++
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
					SetRow(seqInfo.xlsx, seqInfo.Sheets[1], 1, row, []interface{}{tSeq, seqInfo.BarCode})
					row++
				} else {
					seqInfo.Stats["analyzedExcludeReadsNum"]++
				}
			} else {
				fmtUtil.Fprintf(
					outputUnmatched,
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

func (seqInfo *SeqInfo) SetBarCode() {
	for i, s := range seqInfo.HitSeq {
		SetRow(seqInfo.xlsx, seqInfo.Sheets[2], 1, i+1, []interface{}{s, seqInfo.HitSeqCount[s]})
	}
}

func (seqInfo *SeqInfo) WriteSeqResultNum() {
	var (
		keys = seqInfo.HitSeq
	)
	for _, key := range keys {
		if key == string(seqInfo.Seq) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets[3], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key]})
			seqInfo.rowDeletion++
			seqInfo.countDeletion += seqInfo.HitSeqCount[key]
			continue
		}
		if seqInfo.Align1(key) {
			continue
		}

		if seqInfo.Align2(key) {
			continue
		}

		if seqInfo.Align3(key) {
			continue
		}

		SetRow(seqInfo.xlsx, seqInfo.Sheets[10], 1, seqInfo.rowOther, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
		seqInfo.rowOther++
		seqInfo.Stats["errorOtherReadsNum"] += seqInfo.HitSeqCount[key]
	}
	SetRow(seqInfo.xlsx, seqInfo.Sheets[3], 5, 1, []interface{}{"总数", seqInfo.countDeletion})
	SetRow(seqInfo.xlsx, seqInfo.Sheets[4], 5, 1, []interface{}{"总数", seqInfo.countDeletion1})
	SetRow(seqInfo.xlsx, seqInfo.Sheets[5], 5, 1, []interface{}{"总数", seqInfo.countDeletion2})
	SetRow(seqInfo.xlsx, seqInfo.Sheets[6], 5, 1, []interface{}{"总数", seqInfo.countDeletion3})
}

func (seqInfo *SeqInfo) Align1(key string) bool {
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
		SetRow(seqInfo.xlsx, seqInfo.Sheets[3], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.countDeletion += count
		seqInfo.rowDeletion++
		if delCount == 1 {
			SetRow(seqInfo.xlsx, seqInfo.Sheets[4], 1, seqInfo.rowDeletion1, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.countDeletion1 += count
			seqInfo.rowDeletion1++
		} else if delCount == 2 {
			SetRow(seqInfo.xlsx, seqInfo.Sheets[5], 1, seqInfo.rowDeletion2, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.countDeletion2 += count
			seqInfo.rowDeletion2++
		} else if delCount >= 3 {
			SetRow(seqInfo.xlsx, seqInfo.Sheets[6], 1, seqInfo.rowDeletion3, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.countDeletion3 += count
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

func (seqInfo *SeqInfo) Align2(key string) bool {
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
				SetRow(seqInfo.xlsx, seqInfo.Sheets[8], 1, seqInfo.rowInsertionDeletion, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowInsertionDeletion++
			} else {
				SetRow(seqInfo.xlsx, seqInfo.Sheets[7], 1, seqInfo.rowInsertion, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowInsertion++
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

func (seqInfo *SeqInfo) Align3(key string) bool {
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
		SetRow(seqInfo.xlsx, seqInfo.Sheets[9], 1, seqInfo.rowMutation, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.rowMutation++
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

func (seqInfo *SeqInfo) WriteStats() {
	var stats = seqInfo.Stats

	fmt.Printf(
		"AllReadsNum\t\t= %d\n",
		stats["allReadsNum"],
	)
	fmt.Printf(
		"+ShortReadsNum\t\t= %d\t%7.4f%%)\n",
		stats["shortReadsNum"],
		math2.DivisionInt(stats["shortReadsNum"], stats["allReadsNum"])*100,
	)
	fmt.Printf(
		"+AnalyzedReadsNum\t= %d\t%.4f%%\n",
		stats["analyzedReadsNum"],
		math2.DivisionInt(stats["analyzedReadsNum"], stats["allReadsNum"]-stats["shortReadsNum"])*100,
	)
	fmt.Printf(
		"++ExcludeReadsNum\t= %d\t%7.4f%%\n",
		stats["analyzedExcludeReadsNum"],
		math2.DivisionInt(stats["analyzedExcludeReadsNum"], stats["analyzedReadsNum"])*100,
	)
	fmt.Printf(
		"++SeqHitReadsNum\t= %d\t%.4f%%\tAccuracy = %.4f%%,\n",
		stats["seqHitReadsNum"],
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"])*100,
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"]-stats["errorOtherReadsNum"])*100,
	)
	fmt.Printf(
		"++IndexPolyAReadsNum\t= %d\t%.4f%%\n",
		stats["indexPolyAReadsNum"],
		math2.DivisionInt(stats["indexPolyAReadsNum"], stats["analyzedReadsNum"])*100,
	)
	fmt.Printf(
		"+++ErrorReadsNum\t= %d\n",
		stats["errorReadsNum"],
	)
	fmt.Printf(
		"++++ErrorDelReadsNum\t= %d\t%.4f%%\n",
		stats["errorDelReadsNum"],
		math2.DivisionInt(stats["errorDelReadsNum"], stats["errorReadsNum"])*100,
	)
	fmt.Printf(
		"++++ErrorInsReadsNum\t= %d\t%.4f%%\n",
		stats["errorInsReadsNum"],
		math2.DivisionInt(stats["errorInsReadsNum"], stats["errorReadsNum"])*100,
	)
	fmt.Printf(
		"++++ErrorMutReadsNum\t= %d\t%7.4f%%\n",
		stats["errorMutReadsNum"],
		math2.DivisionInt(stats["errorMutReadsNum"], stats["errorReadsNum"])*100,
	)
	fmt.Printf(
		"++++ErrorOtherReadsNum\t= %d\t%.4f%%\n",
		stats["errorOtherReadsNum"],
		math2.DivisionInt(stats["errorOtherReadsNum"], stats["errorReadsNum"])*100,
	)
	fmt.Printf(
		"++AverageBaseAccuracy\t= %7.4f%%\t%d/%d\n",
		math2.DivisionInt(stats["accuRightNum"], stats["accuReadsNum"])*100,
		stats["accuRightNum"], stats["accuReadsNum"],
	)
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
		math2.DivisionInt(stats["seqHitReadsNum"], stats["analyzedReadsNum"]),
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
