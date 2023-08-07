package main

import (
	"bufio"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	//"compress/gzip"
	gzip "github.com/klauspost/pgzip"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	math2 "github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/stringsUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
	"github.com/xuri/excelize/v2"
)

type SeqInfo struct {
	Name  string
	Excel string

	xlsx      *excelize.File
	Sheets    map[string]string
	SheetList []string
	Style     map[string]int

	rowDeletion          int
	rowDeletion1         int
	rowDeletion2         int
	rowDeletionDup       int
	rowDeletion3         int
	rowInsertion         int
	rowInsertionDeletion int
	rowMutation          int
	rowOther             int

	Seq         []byte
	Align       []byte
	AlignInsert []byte
	AlignMut    []byte
	Offset      int

	IndexSeq string
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

	// summary
	// 收率
	YieldCoefficient     float64
	AverageYieldAccuracy float64

	// One-step accuracy rate
	OSAR float64
}

var center = &excelize.Style{
	Alignment: &excelize.Alignment{
		Horizontal: "center",
	},
}

func (seqInfo *SeqInfo) Init() {
	for i := 0; i < len(seqInfo.Seq); i++ {
		for j := 0; j < 4; j++ {
			seqInfo.DistributionNum[j] = append(seqInfo.DistributionNum[j], 0)
			seqInfo.DistributionFreq[j] = append(seqInfo.DistributionFreq[j], 0)
		}
	}

	seqInfo.xlsx = excelize.NewFile()
	seqInfo.Style = make(map[string]int)
	seqInfo.Style["center"] = simpleUtil.HandleError(seqInfo.xlsx.NewStyle(center)).(int)

	seqInfo.rowDeletion = 2
	seqInfo.rowDeletion1 = 2
	seqInfo.rowDeletion2 = 2
	seqInfo.rowDeletionDup = 2
	seqInfo.rowDeletion3 = 2
	seqInfo.rowInsertion = 2
	seqInfo.rowInsertionDeletion = 2
	seqInfo.rowMutation = 2
	seqInfo.rowOther = 2
	for i, sheet := range seqInfo.SheetList {
		if i == 0 {
			simpleUtil.CheckErr(seqInfo.xlsx.SetSheetName("Sheet1", sheet))
		} else {
			simpleUtil.HandleError(seqInfo.xlsx.NewSheet(sheet))
		}
	}

	// 设置列宽
	//simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets[0], "A", "A", 20))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets["Stats"], "M", "R", 12))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets["Stats"], "S", "S", 14))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets["BarCode"], "A", "E", 50))
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets["BarCode"], "B", "B", 50))

	for i := 3; i < len(seqInfo.SheetList)-1; i++ {
		SetRow(seqInfo.xlsx, seqInfo.SheetList[i], 1, 1, []interface{}{"#TargetSeq", "SubMatchSeq", "Count", "AlignResult"})
		simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.SheetList[i], "A", "D", 25))
	}
	SetRow(seqInfo.xlsx, seqInfo.Sheets["Other"], 1, 1, []interface{}{"#TargetSeq", "SubMatchSeq", "Count", "AlignDeletion", "AlignInsertion", "AlignMutation"})
	simpleUtil.CheckErr(seqInfo.xlsx.SetColWidth(seqInfo.Sheets["Other"], "A", "F", 25))
}

func (seqInfo *SeqInfo) Save() {
	log.Printf("seqInfo.xlsx.SaveAs(%s)", seqInfo.Excel)

	simpleUtil.CheckErr(seqInfo.xlsx.SaveAs(seqInfo.Excel))
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4(verbose int) {
	// 1. 统计不同测序结果出现的频数
	seqInfo.WriteSeqResult(".SeqResult.txt", verbose)

	seqInfo.GetHitSeq()

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	seqInfo.WriteSeqResultNum()

	seqInfo.UpdateDistributionStats()

	//seqInfo.PrintStats()
}

func (seqInfo *SeqInfo) WriteSeqResult(path string, verbose int) {
	var (
		tarSeq    = string(seqInfo.Seq)
		indexSeq  = seqInfo.IndexSeq
		tarLength = len(tarSeq) + 10
		//seqHit      = regexp.MustCompile(indexSeq + tarSeq)
		polyA       = regexp.MustCompile(`(.*?)` + indexSeq + `(.*?)AAAAAAAA`)
		regIndexSeq = regexp.MustCompile(indexSeq + `(.*?)$`)
		regTarSeq   = regexp.MustCompile(tarSeq)
		termA       = 0
		termFix     = ""

		outputShort     *os.File
		outputUnmatched *os.File
	)
	if tarSeq == "A" {
		polyA = regexp.MustCompile(`(.*?)` + indexSeq + `(.*?)TTTTTTTT`)
	}
	if verbose > 0 {
		outputShort = osUtil.Create(filepath.Join(*outputDir, seqInfo.Name+path+".short.txt"))
		outputUnmatched = osUtil.Create(filepath.Join(*outputDir, seqInfo.Name+path+".unmatched.txt"))
		fmtUtil.Fprintf(outputUnmatched, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "#Seq", "A", "C", "G", "T", "TargetSeq", "IndexSeq", "PloyA")
	}
	for i := len(tarSeq) - 1; i > -1; i-- {
		if tarSeq[i] == 'A' {
			termA++
			termFix += "A"
		} else {
			break
		}
	}
	log.Printf("[%-10s] Fix:[%s]\n", seqInfo.Name, termFix)

	for _, fastq := range seqInfo.Fastqs {
		log.Printf("load %s", fastq)
		var (
			file    = osUtil.Open(fastq)
			scanner *bufio.Scanner
			i       = -1
		)
		if gz.MatchString(fastq) {
			scanner = bufio.NewScanner(simpleUtil.HandleError(gzip.NewReader(file)).(io.Reader))
		} else {
			scanner = bufio.NewScanner(file)
		}
		for scanner.Scan() {
			var s = scanner.Text()
			i++
			if i%4 != 1 {
				continue
			}
			seqInfo.ReadsLength[len(s)]++

			seqInfo.Stats["AllReadsNum"]++
			if len(s) < 50 {
				seqInfo.Stats["ShortReadsNum"]++
				if verbose > 0 {
					fmtUtil.Fprintf(outputShort, "%s\t%d\n", s, len(s))
				}
				continue
			}
			var (
				tSeq string
				rcS  = ReverseComplement(s)
				// regexp match
				m []string
			)

			if regIndexSeq.MatchString(s) {
				seqInfo.Stats["IndexReadsNum"]++
			} else if *rc && regIndexSeq.MatchString(rcS) {
				seqInfo.Stats["IndexReadsNum"]++
			}

			if regIndexSeq.MatchString(s) {
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
			} else if regIndexSeq.MatchString(rcS) {
				for i2, c := range []byte(rcS) {
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
			}
			if polyA.MatchString(s) || polyA.MatchString(rcS) {
				if polyA.MatchString(s) {
					m = polyA.FindStringSubmatch(s)
				} else if polyA.MatchString(rcS) {
					m = polyA.FindStringSubmatch(rcS)
				}
				//m = polyA.FindStringSubmatch(s)

				tSeq = m[2] //[seqInfo.Offset:]
				if tarSeq != "A" {
					tSeq += termFix
				}

				if len(tSeq) <= seqInfo.Offset {
					tSeq += "X"
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else if tSeq[seqInfo.Offset:] == tarSeq[seqInfo.Offset:] {
					seqInfo.Stats["RightReadsNum"]++
					seqInfo.HitSeqCount[tSeq]++
				} else if !regN.MatchString(tSeq[seqInfo.Offset:]) && len(tSeq) < tarLength {
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else {
					//fmt.Printf("[%s]:[%s]:[%+v]\n", s, tSeq, m)
					seqInfo.Stats["ExcludeReadsNum"]++
				}
			} else if *long && (regIndexSeq.MatchString(s) || regIndexSeq.MatchString(rcS)) {
				//m = regIndexSeq.FindStringSubmatch(s)
				if regIndexSeq.MatchString(s) {
					m = regIndexSeq.FindStringSubmatch(s)
				} else if regIndexSeq.MatchString(rcS) {
					m = regIndexSeq.FindStringSubmatch(rcS)
				}
				tSeq = m[1]
				var cut = len(tSeq)
				for {
					if cut > 0 && tSeq[cut-1] == 'A' {
						cut--
					} else {
						break
					}
				}
				tSeq = tSeq[:cut]
				if tarSeq != "A" {
					tSeq += termFix
				}
				if len(tSeq) <= seqInfo.Offset {
					tSeq += "X"
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else if tSeq[seqInfo.Offset:] == tarSeq[seqInfo.Offset:] {
					seqInfo.Stats["RightReadsNum"]++
					seqInfo.HitSeqCount[tSeq]++
				} else if !regN.MatchString(tSeq[seqInfo.Offset:]) && len(tSeq) < tarLength {
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else {
					//fmt.Printf("[%s]:[%s]:[%+v]\n", s, tSeq, m)
					seqInfo.Stats["ExcludeReadsNum"]++
				}

			} else {
				seqInfo.Stats["UnmatchedReadsNum"]++
				if verbose > 1 {

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
		simpleUtil.CheckErr(file.Close())
	}
	seqInfo.Stats["AnalyzedReadsNum"] = seqInfo.Stats["RightReadsNum"] + seqInfo.Stats["IndexPolyAReadsNum"]

	if verbose > 0 {
		simpleUtil.CheckErr(outputShort.Close())
		simpleUtil.CheckErr(outputUnmatched.Close())
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

func (seqInfo *SeqInfo) WriteSeqResultNum() {
	for i, key := range seqInfo.HitSeq {
		if key[seqInfo.Offset:] == string(seqInfo.Seq[seqInfo.Offset:]) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key]})
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key]})
			seqInfo.rowDeletion++
			continue
		}
		if seqInfo.Align1(key) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align})
			continue
		}

		if seqInfo.Align2(key) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert})
			continue
		}

		if seqInfo.Align3(key) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
			continue
		}
		SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})

		SetRow(seqInfo.xlsx, seqInfo.Sheets["Other"], 1, seqInfo.rowOther, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
		seqInfo.rowOther++
		seqInfo.Stats["ErrorOtherReadsNum"] += seqInfo.HitSeqCount[key]
	}
	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["ErrorDelReadsNum"] + seqInfo.Stats["RightReadsNum"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion1"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["ErrorDel1ReadsNum"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion2"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["ErrorDel2ReadsNum"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDup"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["ErrorDelDupReadsNum"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion3"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["ErrorDel3ReadsNum"]},
	)

	var sheet = seqInfo.Sheets["Deletion"]
	for i := 3; i < seqInfo.rowDeletion; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["ErrorDelReadsNum"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["Deletion1"]
	for i := 2; i < seqInfo.rowDeletion1; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["ErrorDel1ReadsNum"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["Deletion2"]
	for i := 2; i < seqInfo.rowDeletion2; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["ErrorDel2ReadsNum"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["Deletion3"]
	for i := 2; i < seqInfo.rowDeletion3; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["ErrorDel3ReadsNum"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionDup"]
	for i := 2; i < seqInfo.rowDeletionDup; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["ErrorDelDupReadsNum"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
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
		seqInfo.Stats["ErrorDelReadsNum"] += count
		return true
	}

	var k = 0 // match count to Seq
	for i := range a {
		if k < len(b) && (a[i] == b[k] || a[i] == 'N') {
			c = append(c, b[k])
			k++
		} else {
			c = append(c, '-')
			delCount++
		}
	}
	seqInfo.Align = c
	//if k >= len(b) && !minus3.Match(c) { // all match
	if k >= len(b) { // all match
		SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.rowDeletion++
		if delCount == 1 {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion1"], 1, seqInfo.rowDeletion1, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.Stats["ErrorDel1ReadsNum"] += count
			seqInfo.rowDeletion1++
		} else if delCount == 2 {
			if minus2.Match(c) {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDup"], 1, seqInfo.rowDeletionDup, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.Stats["ErrorDelDupReadsNum"] += count
				seqInfo.rowDeletionDup++
			} else {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion2"], 1, seqInfo.rowDeletion2, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.Stats["ErrorDel2ReadsNum"] += count
				seqInfo.rowDeletion2++
			}
		} else if delCount >= 3 {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion3"], 1, seqInfo.rowDeletion3, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.Stats["ErrorDel3ReadsNum"] += count
			seqInfo.rowDeletion3++
		}
		for i, c1 := range c {
			if c1 == '-' {
				seqInfo.DistributionNum[0][i] += count
			}
		}
		seqInfo.Stats["ErrorDelReadsNum"] += count
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
			if i < len(a) && k < len(b) && (a[i] == b[k] || a[i] == 'N') { // match to Seq
				c = append(c, b[k])
				k += 1
			} else if i > 0 && i <= len(a) && k < len(b) && (a[i-1] == b[k] || a[i-1] == 'N') { // match to Seq -1 bp
				c = append(c, '+')
				k += 1
				i--
			} else {
				c = append(c, '-')
			}
		}
	}
	seqInfo.AlignInsert = c
	if k >= len(b)-1 && c[0] != '+' {
		//if !plus3.Match(c) && !minus3.Match(c) && !m2p2.Match(c) && minus1.Match(c) {
		if !plus3.Match(c) {
			if minus1.Match(c) {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["InsertionDeletion"], 1, seqInfo.rowInsertionDeletion, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowInsertionDeletion++
				seqInfo.Stats["ErrorInsDelReadsNum"] += count
			} else {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["Insertion"], 1, seqInfo.rowInsertion, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowInsertion++
				seqInfo.Stats["ErrorInsReadsNum"] += count
			}
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
			if i < len(b) && (s == b[i] || s == 'N') {
				c = append(c, s)
			} else {
				k++
				c = append(c, 'X')
			}
		}
	}
	seqInfo.AlignMut = c
	if k < 2 && len(c) > 0 {
		SetRow(seqInfo.xlsx, seqInfo.Sheets["Mutation"], 1, seqInfo.rowMutation, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.rowMutation++
		seqInfo.Stats["ErrorMutReadsNum"] += count
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
	seqInfo.Stats["ErrorReadsNum"] = seqInfo.Stats["ErrorDelReadsNum"] + seqInfo.Stats["ErrorInsReadsNum"] + seqInfo.Stats["ErrorInsDelReadsNum"] + seqInfo.Stats["ErrorMutReadsNum"] + seqInfo.Stats["ErrorOtherReadsNum"]
	seqInfo.Stats["ExcludeOtherReadsNum"] = seqInfo.Stats["RightReadsNum"] + seqInfo.Stats["ErrorReadsNum"] - seqInfo.Stats["ErrorOtherReadsNum"]
	seqInfo.Stats["AccuReadsNum"] = seqInfo.Stats["ExcludeOtherReadsNum"] * len(seqInfo.Seq)

	for i := range seqInfo.Seq {
		// right reads num
		seqInfo.DistributionNum[3][i] = seqInfo.Stats["ExcludeOtherReadsNum"] - seqInfo.DistributionNum[0][i] - seqInfo.DistributionNum[1][i] - seqInfo.DistributionNum[2][i]
		for j := 0; j < 4; j++ {
			seqInfo.DistributionFreq[j][i] = math2.DivisionInt(seqInfo.DistributionNum[j][i], seqInfo.Stats["ExcludeOtherReadsNum"])
		}

		seqInfo.Stats["AccuRightNum"] += seqInfo.DistributionNum[3][i]
	}
}

func (seqInfo *SeqInfo) PrintStats() {
	var (
		stats = seqInfo.Stats
		out   = osUtil.Create(filepath.Join("result", seqInfo.Name+".stats.txt"))
	)
	defer simpleUtil.DeferClose(out)

	fmtUtil.Fprintf(out,
		"AllReadsNum\t\t= %d\n",
		stats["AllReadsNum"],
	)
	fmtUtil.Fprintf(out,
		"+ShortReadsNum\t\t= %d\t%7.4f%%\n",
		stats["ShortReadsNum"],
		math2.DivisionInt(stats["ShortReadsNum"], stats["AllReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"+UnmatchedReadsNum\t= %d\t%7.4f%%\n",
		stats["UnmatchedReadsNum"],
		math2.DivisionInt(stats["UnmatchedReadsNum"], stats["AllReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"+ExcludeReadsNum\t= %d\t%7.4f%%\n",
		stats["ExcludeReadsNum"],
		math2.DivisionInt(stats["ExcludeReadsNum"], stats["AllReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"+IndexReadsNum\t\t= %d\t%.4f%%\n",
		stats["IndexReadsNum"],
		math2.DivisionInt(stats["IndexReadsNum"], stats["AllReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"+AnalyzedReadsNum\t= %d\t%.4f%%\n",
		stats["AnalyzedReadsNum"],
		math2.DivisionInt(stats["AnalyzedReadsNum"], stats["IndexReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++RightReadsNum\t\t= %d\t%.4f%%\n",
		stats["RightReadsNum"],
		math2.DivisionInt(stats["RightReadsNum"], stats["AnalyzedReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++IndexPolyAReadsNum\t= %d\t%.4f%%\n",
		stats["IndexPolyAReadsNum"],
		math2.DivisionInt(stats["IndexPolyAReadsNum"], stats["AnalyzedReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"+++ErrorReadsNum\t= %d\n",
		stats["ErrorReadsNum"],
	)
	fmtUtil.Fprintf(out,
		"++++ErrorDelReadsNum\t= %d\t%.4f%%\n",
		stats["ErrorDelReadsNum"],
		math2.DivisionInt(stats["ErrorDelReadsNum"], stats["ErrorReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++++ErrorInsReadsNum\t= %d\t%.4f%%\n",
		stats["ErrorInsReadsNum"],
		math2.DivisionInt(stats["ErrorInsReadsNum"], stats["ErrorReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++++ErrorMutReadsNum\t= %d\t%7.4f%%\n",
		stats["ErrorMutReadsNum"],
		math2.DivisionInt(stats["ErrorMutReadsNum"], stats["ErrorReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++++ErrorOtherReadsNum\t= %d\t%.4f%%\n",
		stats["ErrorOtherReadsNum"],
		math2.DivisionInt(stats["ErrorOtherReadsNum"], stats["ErrorReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++AverageBaseAccuracy\t= %7.4f%%\t%d/%d\n",
		math2.DivisionInt(stats["AccuRightNum"], stats["AccuReadsNum"])*100,
		stats["AccuRightNum"], stats["AccuReadsNum"],
	)
}

func (seqInfo *SeqInfo) PlotLineACGT(path string) {
	var (
		line   = charts.NewLine()
		xaxis  [151]int
		yaxis  [151]int
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
		yaxis[i] = seqInfo.A[i] + seqInfo.C[i] + seqInfo.G[i] + seqInfo.T[i]
	}

	line.SetXAxis(xaxis).
		AddSeries("A", generateLineItems(seqInfo.A[:])).
		AddSeries("C", generateLineItems(seqInfo.C[:])).
		AddSeries("G", generateLineItems(seqInfo.G[:])).
		AddSeries("T", generateLineItems(seqInfo.T[:])).
		AddSeries("ALL", generateLineItems(yaxis[:]))
	// SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	simpleUtil.CheckErr(line.Render(output))
}

func (seqInfo *SeqInfo) WriteStatsSheet() {
	var (
		stats = seqInfo.Stats
		xlsx  = seqInfo.xlsx
		sheet = seqInfo.Sheets["Stats"]
		rIdx  = 1

		titleTar     = textUtil.File2Array(path.Join(etcPath, "title.Tar.txt"))
		titleStats   = textUtil.File2Array(path.Join(etcPath, "title.Stats.txt"))
		statsMap     = make(map[string]interface{})
		distribution = seqInfo.DistributionFreq
		readsCount   = stats["AnalyzedReadsNum"]
		title        []interface{}

		out  = osUtil.Create(filepath.Join("result", seqInfo.Name+".steps.txt"))
		osar = osUtil.Create(filepath.Join("result", seqInfo.Name+".one.step.accuracy.rate.txt"))
	)
	defer simpleUtil.DeferClose(out)
	defer simpleUtil.DeferClose(osar)

	SetCellStr(xlsx, sheet, 1, 1, seqInfo.Name)
	MergeCells(xlsx, sheet, 1, rIdx, len(titleTar), rIdx)
	rIdx++

	for _, s := range titleStats {
		var c, ok = stats[s]
		if ok {
			statsMap[s] = c
		}
	}
	statsMap["靶标"] = seqInfo.IndexSeq
	statsMap["合成序列"] = string(seqInfo.Seq)
	statsMap["Accuracy"] = math2.DivisionInt(stats["RightReadsNum"], stats["AnalyzedReadsNum"])
	statsMap["AverageBaseAccuracy"] = math2.DivisionInt(stats["AccuRightNum"], stats["AccuReadsNum"])
	for _, s := range titleStats {
		SetRow(xlsx, sheet, 1, rIdx, []interface{}{s, "", statsMap[s]})
		MergeCells(xlsx, sheet, 1, rIdx, 2, rIdx)
		MergeCells(xlsx, sheet, 3, rIdx, len(titleTar), rIdx)
		rIdx++
	}

	// Tar stats
	for _, s := range titleTar {
		title = append(title, s)
	}

	fmtUtil.FprintStringArray(out, titleTar, "\t")
	SetRow(xlsx, sheet, 1, rIdx, title)
	rIdx++

	var (
		sumDel    = 0
		countDels = make(map[byte]int)
		sequence  = seqInfo.IndexSeq[len(seqInfo.IndexSeq)-4:] + string(seqInfo.Seq)
	)
	for i, b := range seqInfo.Seq {
		var counts = make(map[byte]int)
		for seq, count := range seqInfo.HitSeqCount {
			if len(seq) <= i {
				delete(seqInfo.HitSeqCount, seq)
				continue
			}

			counts[seq[i]] += count

			if b != 'N' && seq[i] != b {
				delete(seqInfo.HitSeqCount, seq)
			}
		}

		var (
			N     = counts['A'] + counts['C'] + counts['G'] + counts['T']
			del   = readsCount - N
			del1  = del
			ratio = make(map[byte]float64)
		)
		counts['N'] = N
		seqInfo.YieldCoefficient = math2.DivisionInt(counts[b], stats["AnalyzedReadsNum"])

		if i < len(seqInfo.Seq)-1 && seqInfo.Seq[i+1] != seqInfo.Seq[i] {
			del1 = counts[seqInfo.Seq[i+1]]
		}
		countDels[b] += del1

		ratio['A'] = math2.DivisionInt(counts['A'], readsCount)
		ratio['T'] = math2.DivisionInt(counts['T'], readsCount)
		ratio['C'] = math2.DivisionInt(counts['C'], readsCount)
		ratio['G'] = math2.DivisionInt(counts['G'], readsCount)
		ratio['N'] = math2.DivisionInt(counts['N'], readsCount)
		seqInfo.OSAR = ratio[b]
		var ratioDel = math2.DivisionInt(del1, readsCount)
		var ratioSort = RankByteFloatMap(ratio)

		seqInfo.AverageYieldAccuracy = math.Pow(seqInfo.YieldCoefficient, 1.0/float64(i+1))

		var rowValue = []interface{}{
			i + 1,
			string(b),
			distribution[0][i],
			distribution[1][i],
			distribution[2][i],
			distribution[3][i],
			readsCount,
			counts['A'],
			counts['T'],
			counts['C'],
			counts['G'],
			del,
			seqInfo.YieldCoefficient, // 收率
			ratio['A'],
			ratio['T'],
			ratio['C'],
			ratio['G'],
			seqInfo.OSAR,                 // 单步 准确率
			seqInfo.AverageYieldAccuracy, // 收率平均准确率
			string(ratioSort[0].Key),
			ratioSort[0].Value,
			string(ratioSort[1].Key),
			ratioSort[1].Value,
			del1,
			ratioDel,
		}

		readsCount = counts[b]

		SetRow(xlsx, sheet, 1, rIdx, rowValue)
		rIdx++

		fmtUtil.Fprintf(
			osar,
			"%s\t%s\t%c\t%d\t%f\n",
			seqInfo.Name,
			sequence[i:i+4],
			sequence[i+4],
			i+1,
			ratio[b],
		)

		fmtUtil.Fprintf(
			out,
			"%d\t%s\t%f\t%f\t%f\t%f\t%d\t%d\t%d\t%d\t%d\t%d\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%s\t%f\t%s\t%f\t%d\t%f\n",
			rowValue[0],
			rowValue[1],
			rowValue[2],
			rowValue[3],
			rowValue[4],
			rowValue[5],
			rowValue[6],
			rowValue[7],
			rowValue[8],
			rowValue[9],
			rowValue[10],
			rowValue[11],
			rowValue[12],
			rowValue[13],
			rowValue[14],
			rowValue[15],
			rowValue[16],
			rowValue[17],
			rowValue[18],
			rowValue[19],
			rowValue[20],
			rowValue[21],
			rowValue[22],
			del1,
			ratioDel,
		)
		sumDel += del1
	}

	log.Printf(
		"Simple Deletion:\t%s\nAll\t%d\t%.0f%%\nA\t%d\t%0.f%%\nT\t%d\t%.0f%%\nC\t%d\t%.0f%%\nG\t%d\t%.0f%%\n",
		seqInfo.Name,
		sumDel, math2.DivisionInt(100*sumDel, seqInfo.Stats["ErrorReadsNum"]),
		countDels['A'], math2.DivisionInt(100*countDels['A'], sumDel),
		countDels['T'], math2.DivisionInt(100*countDels['T'], sumDel),
		countDels['C'], math2.DivisionInt(100*countDels['C'], sumDel),
		countDels['G'], math2.DivisionInt(100*countDels['G'], sumDel),
	)

	simpleUtil.CheckErr(seqInfo.xlsx.SetRowStyle(sheet, 1, rIdx-1, seqInfo.Style["center"]))
}

type ByteFloat struct {
	Key   byte
	Value float64
}

type ByteFloatList []ByteFloat

// Len returns the length of the ByteFloatList.
//
// It does not take any parameters.
// Returns an integer representing the length of the ByteFloatList.
func (l ByteFloatList) Len() int {
	return len(l)

}

// Less returns whether the element at index i is less than the element at index j in the ByteFloatList.
//
// Parameters:
// - i: the index of the first element to compare
// - j: the index of the second element to compare
//
// Returns:
// - true if the element at index i is less than the element at index j, false otherwise.
func (l ByteFloatList) Less(i, j int) bool {
	return l[i].Value < l[j].Value
}

// Swap swaps the elements at index i and j in the ByteFloatList.
//
// Parameters:
//
//	i - the index of the first element to be swapped.
//	j - the index of the second element to be swapped.
func (l ByteFloatList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// RankByteFloatMap generates a ranked list of ByteFloat values based on the provided map.
//
// It takes a map of byte keys and float64 values as input and returns a ByteFloatList.
func RankByteFloatMap(data map[byte]float64) ByteFloatList {
	var (
		l = make(ByteFloatList, len(data))
		i = 0
	)
	for b, f := range data {
		l[i] = ByteFloat{
			Key:   b,
			Value: f,
		}
		i++
	}
	sort.Sort(sort.Reverse(l))
	return l
}
