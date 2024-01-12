package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

const (
	kmerLength = 9
)

type ParallelTest struct {
	ID string

	// 收率
	YieldCoefficient     []float64
	YieldCoefficientMean float64
	YieldCoefficientSD   float64

	// 单步准确率
	// One-step accuracy rate
	AverageYieldAccuracy     []float64
	AverageYieldAccuracyMean float64
	AverageYieldAccuracySD   float64
}

func (p *ParallelTest) Calculater() {
	p.YieldCoefficientMean, p.YieldCoefficientSD = math2.MeanStdDev(p.YieldCoefficient)
	p.AverageYieldAccuracyMean, p.AverageYieldAccuracySD = math2.MeanStdDev(p.AverageYieldAccuracy)
}

type SeqInfo struct {
	Name           string
	ParallelTestID string
	Excel          string

	UseReverseComplement bool
	AssemblerMode        bool
	Reverse              bool

	xlsx      *excelize.File
	Sheets    map[string]string
	SheetList []string
	Style     map[string]int
	del3      *os.File
	del1      *os.File

	// Discrete and continuous
	rowDeletion            int // 所有缺失
	rowDeletionSingle      int // 缺1nt
	rowDeletionDiscrete2   int // 离散缺2nt
	rowDeletionDiscrete3   int // 离散缺3nt
	rowDeletionContinuous2 int // 连续缺2nt
	rowDeletionContinuous3 int // 连续缺3nt

	rowInsertion             int
	rowInsertionDeletion     int
	rowMutation              int
	rowOther                 int
	DeletionContinuous3Index int

	Seq         []byte
	Align       []byte
	AlignInsert []byte
	AlignMut    []byte

	IndexSeq string
	Fastqs   []string

	HitSeq      []string
	HitSeqCount map[string]int
	Stats       map[string]int

	DistributionNum  [4][]int
	DistributionFreq [4][]float64

	// fastq
	ReadsLength map[int]int
	A           [300]int
	C           [300]int
	G           [300]int
	T           [300]int
	DNA         [300]byte
	DNAKmer     [kmerLength][300]map[string]int
	Kmer        map[string]int

	// summary
	// 收率
	YieldCoefficient     float64
	AverageYieldAccuracy float64

	// One-step accuracy rate
	OSAR float64
}

func NewSeqInfo(data map[string]string, long, rev bool) *SeqInfo {
	var seqInfo = new(SeqInfo)
	seqInfo = &SeqInfo{
		Name:                 data["id"],
		ParallelTestID:       data["平行"],
		IndexSeq:             strings.ToUpper(data["index"]),
		Seq:                  []byte(strings.ToUpper(data["seq"])),
		Fastqs:               strings.Split(data["fq"], ","),
		Excel:                filepath.Join(*outputDir, data["id"]+".xlsx"),
		Sheets:               Sheets,
		SheetList:            sheetList,
		Stats:                make(map[string]int),
		HitSeqCount:          make(map[string]int),
		ReadsLength:          make(map[int]int),
		AssemblerMode:        long,
		Reverse:              rev,
		UseReverseComplement: true,
	}
	// support N
	seqInfo.IndexSeq = strings.Replace(seqInfo.IndexSeq, "N", ".", -1)

	if seqInfo.Reverse {
		seqInfo.Seq = Reverse(seqInfo.Seq)
	}
	log.Printf("[%s]:[%s]:[%s]:[%+v]\n", seqInfo.Name, seqInfo.IndexSeq, seqInfo.Seq, seqInfo.Fastqs)
	return seqInfo
}

func (seqInfo *SeqInfo) Init() {
	seqInfo.Kmer = make(map[string]int)

	var refNt = append([]byte(seqInfo.IndexSeq), seqInfo.Seq...)
	for i := 0; i < 300; i++ {
		seqInfo.DNA[i] = byte('A')
		if i < len(refNt) {
			seqInfo.DNA[i] = refNt[i]
		}
		for j := 0; j < kmerLength; j++ {
			seqInfo.DNAKmer[j][i] = make(map[string]int)
		}
	}
	for i := 0; i < len(seqInfo.Seq); i++ {
		for j := 0; j < 4; j++ {
			seqInfo.DistributionNum[j] = append(seqInfo.DistributionNum[j], 0)
			seqInfo.DistributionFreq[j] = append(seqInfo.DistributionFreq[j], 0)
		}
	}

	seqInfo.xlsx = excelize.NewFile()
	seqInfo.Style = make(map[string]int)
	var center = &excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	}
	seqInfo.Style["center"] = simpleUtil.HandleError(seqInfo.xlsx.NewStyle(center)).(int)

	seqInfo.rowDeletion = 2
	seqInfo.rowDeletionSingle = 2
	seqInfo.rowDeletionDiscrete2 = 2
	seqInfo.rowDeletionContinuous2 = 2
	seqInfo.rowDeletionContinuous3 = 2
	seqInfo.rowDeletionDiscrete3 = 2
	seqInfo.rowInsertion = 2
	seqInfo.rowInsertionDeletion = 2
	seqInfo.rowMutation = 2
	seqInfo.rowOther = 2
	seqInfo.DeletionContinuous3Index = len(seqInfo.Seq)
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

func (seqInfo *SeqInfo) SingleRun(resultDir string) {
	seqInfo.Init()
	seqInfo.CountError4(resultDir, *verbose)

	seqInfo.WriteStatsSheet(resultDir)
	seqInfo.Save()
	seqInfo.PrintStats(resultDir)
	seqInfo.PlotLineACGT(filepath.Join(resultDir, seqInfo.Name))
}

func (seqInfo *SeqInfo) Save() {
	log.Printf("seqInfo.xlsx.SaveAs(%s)", seqInfo.Excel)

	simpleUtil.CheckErr(seqInfo.xlsx.SaveAs(seqInfo.Excel))
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4(outputDir string, verbose int) {
	// 1. 统计不同测序结果出现的频数
	seqInfo.WriteSeqResult(".SeqResult.txt", outputDir, verbose)

	seqInfo.GetHitSeq()

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	seqInfo.del3 = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+".del3.txt"))
	seqInfo.del1 = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+".del1.txt"))
	seqInfo.WriteSeqResultNum()

	seqInfo.UpdateDistributionStats()

	//seqInfo.PrintStats()
}

func (seqInfo *SeqInfo) WriteSeqResult(path, outputDir string, verbose int) {
	var (
		tarSeq   = string(seqInfo.Seq)
		indexSeq = seqInfo.IndexSeq
		//seqHit      = regexp.MustCompile(indexSeq + tarSeq)
		polyA       = regexp.MustCompile(`^` + indexSeq + `(.*?)AAAAAAAA`)
		regIndexSeq = regexp.MustCompile(`^` + indexSeq + `(.*?)$`)
		regTarSeq   = regexp.MustCompile(tarSeq)

		output          = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+path))
		outputShort     *os.File
		outputUnmatched *os.File

		// value weight
		histogram = make(map[int]int)
	)
	if indexSeq == "" {
		polyA = regexp.MustCompile(`^(.*?)AAAAAAAA`)
		regIndexSeq = regexp.MustCompile(`^(.*?)AAAAAAAA`)
		seqInfo.UseReverseComplement = false
	}
	if tarSeq == "A" {
		polyA = regexp.MustCompile(`^` + indexSeq + `(.*?)TTTTTTTT`)
		regIndexSeq = regexp.MustCompile(`^` + indexSeq + `(.*?)$`)
	}
	defer simpleUtil.DeferClose(output)

	if seqInfo.Reverse {
		regTarSeq = regexp.MustCompile(string(Reverse(append([]byte{}, seqInfo.Seq...))))
	}

	if verbose > 0 {
		outputShort = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+path+".short.txt"))
		outputUnmatched = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+path+".unmatched.txt"))
		fmtUtil.Fprintf(outputUnmatched, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "#Seq", "A", "C", "G", "T", "TargetSeq", "IndexSeq", "PloyA")
	}

	for _, fastq := range seqInfo.Fastqs {
		if fastq == "" {
			continue
		}
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
			// if len(s) < 50 {
			// 	seqInfo.Stats["ShortReadsNum"]++
			// 	if verbose > 0 {
			// 		fmtUtil.Fprintf(outputShort, "%s\t%d\n", s, len(s))
			// 	}
			// 	continue
			// }
			var (
				tSeq string
				rcS  = ReverseComplement(s)
				// regexp match
				m []string
			)
			if indexSeq == "" {
				rcS = s
			}

			var byteS []byte
			if regIndexSeq.MatchString(s) {
				byteS = []byte(s)
				seqInfo.Stats["IndexReadsNum"]++
			} else if seqInfo.UseReverseComplement && regIndexSeq.MatchString(rcS) {
				byteS = []byte(rcS)
				seqInfo.Stats["IndexReadsNum"]++
			}

			for i2, c := range byteS {
				if i2 < 300 {
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

			var byteSloc = regexp.MustCompile(`AAAAAAAA`).FindIndex(byteS)
			if byteSloc != nil {
				byteS = byteS[:byteSloc[0]]
			}

			var kmer []byte
			for i2, c := range byteS {
				if i2 < 300 {

					kmer = append([]byte{c}, kmer...)
					for j := 0; j < kmerLength; j++ {
						var n = min(j+1, len(kmer))
						var key = string(kmer[:n])
						seqInfo.DNAKmer[j][i2][key]++
						if n == kmerLength {
							seqInfo.Kmer[key]++
						}
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

				tSeq = m[1] //[seqInfo.Offset:]
				fmtUtil.Fprintln(output, tSeq)
				histogram[len(tSeq)]++
				if seqInfo.Reverse {
					tSeq = string(Reverse([]byte(tSeq)))
				}

				if len(tSeq) == 0 {
					tSeq += "X"
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else if tSeq == tarSeq {
					seqInfo.Stats["RightReadsNum"]++
					seqInfo.HitSeqCount[tSeq]++
				} else if !regN.MatchString(tSeq) {
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else {
					//fmt.Printf("[%s]:[%s]:[%+v]\n", s, tSeq, m)
					seqInfo.Stats["ExcludeReadsNum"]++
				}
			} else if seqInfo.AssemblerMode && (regIndexSeq.MatchString(s) || regIndexSeq.MatchString(rcS)) {
				//m = regIndexSeq.FindStringSubmatch(s)
				if regIndexSeq.MatchString(s) {
					m = regIndexSeq.FindStringSubmatch(s)
				} else if regIndexSeq.MatchString(rcS) {
					m = regIndexSeq.FindStringSubmatch(rcS)
				}
				tSeq = m[1]
				fmtUtil.Fprintln(output, tSeq)
				histogram[len(tSeq)]++
				if seqInfo.Reverse {
					tSeq = string(Reverse([]byte(tSeq)))
				}

				if len(tSeq) == 0 {
					tSeq += "X"
					seqInfo.HitSeqCount[tSeq]++
					seqInfo.Stats["IndexPolyAReadsNum"]++
				} else if tSeq == tarSeq {
					seqInfo.Stats["RightReadsNum"]++
					seqInfo.HitSeqCount[tSeq]++
				} else if !regN.MatchString(tSeq) {
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

	// output histgram.txt
	histogramFile := osUtil.Create(filepath.Join(outputDir, seqInfo.Name+".histogram.txt"))
	fmtUtil.Fprintln(histogramFile, "length\tweight")
	var seqLengths []int
	for k := range histogram {
		seqLengths = append(seqLengths, k)
	}
	sort.Ints(seqLengths)
	for _, k := range seqLengths {
		fmtUtil.Fprintf(histogramFile, "%d\t%d\n", k, histogram[k])
	}
	simpleUtil.CheckErr(histogramFile.Close())

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
		if key == string(seqInfo.Seq) {
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
	// free HitSeq
	seqInfo.HitSeq = nil
	// 输出所有连续3缺失的位置，用于统计断点分布
	for i := 0; i <= len(seqInfo.Seq)-2; i++ {
		var (
			end = i
			seq = string(seqInfo.Seq)
		)
		if end < 2 {
			var indexSeq = seqInfo.IndexSeq
			seq = string(indexSeq[len(indexSeq)-2:]) + seq
			end += 2
		}
		fmtUtil.Fprintf(seqInfo.del3, "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", i, 0, seq[end-2:end], seq[end:end+2], "", "", "")
	}
	simpleUtil.CheckErr(seqInfo.del3.Close())
	simpleUtil.CheckErr(seqInfo.del1.Close())

	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["Deletion"] + seqInfo.Stats["RightReadsNum"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionSingle"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["DeletionSingle"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete2"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["DeletionDiscrete2"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["DeletionContinuous2"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete3"], 5, 1,
		[]interface{}{"总数", seqInfo.Stats["DeletionDiscrete3"]},
	)

	var sheet = seqInfo.Sheets["Deletion"]
	for i := 3; i < seqInfo.rowDeletion; i++ {
		// var countStr = GetCellValue(seqInfo.xlsx, sheet, 3, i)
		// var count = 0
		// if countStr != "" {
		// 	count = stringsUtil.Atoi(countStr, fmt.Sprint("from Deletion:", seqInfo.Name, " ", i))
		// }
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), fmt.Sprint("from Deletion:", seqInfo.Name, " ", i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["Deletion"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionSingle"]
	for i := 2; i < seqInfo.rowDeletionSingle; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), fmt.Sprint("from DeletionSingle:", seqInfo.Name, " ", i, " ", seqInfo.rowDeletionSingle))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionSingle"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionDiscrete2"]
	for i := 2; i < seqInfo.rowDeletionDiscrete2; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), fmt.Sprint("from DeletionDiscrete2:", seqInfo.Name, " ", i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionDiscrete2"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionDiscrete3"]
	for i := 2; i < seqInfo.rowDeletionDiscrete3; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), fmt.Sprint("from DeletionDiscrete3:", seqInfo.Name, " ", i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionDiscrete3"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionContinuous2"]
	for i := 2; i < seqInfo.rowDeletionContinuous2; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), fmt.Sprint("from DeletionContinuous2:", seqInfo.Name, " ", i))
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionContinuous2"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
}

// var dash = regexp.MustCompile(`-+`)
var dash3 = regexp.MustCompile(`---+`)
var dashEnd = regexp.MustCompile(`-$`)

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
		seqInfo.Stats["Deletion"] += count
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
		seqInfo.Stats["Deletion"] += count

		SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, count, c})
		seqInfo.rowDeletion++

		if delCount == 1 { // 单个缺失
			seqInfo.Stats["DeletionSingle"] += count

			SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionSingle"], 1, seqInfo.rowDeletionSingle, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.rowDeletionSingle++

			var m = minus1.FindIndex(c)
			if m != nil {
				if m[0] == 0 {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, '^', a[m[1]-1], c[m[1]])
				} else if m[1] == len(c) {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, c[m[0]-1], a[m[1]-1], '$')
				} else {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, c[m[0]-1], a[m[1]-1], c[m[1]])
				}
			}

		} else if delCount == 2 { // 2缺失
			seqInfo.Stats["Deletion2"] += count

			if minus2.Match(c) { // 连续2缺失
				seqInfo.Stats["DeletionContinuous2"] += count

				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 1, seqInfo.rowDeletionContinuous2, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowDeletionContinuous2++
			} else { // 离散2缺失
				seqInfo.Stats["DeletionDiscrete2"] += count

				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete2"], 1, seqInfo.rowDeletionDiscrete2, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowDeletionDiscrete2++
			}
		} else if delCount >= 3 {
			seqInfo.Stats["Deletion3"] += count

			if minus3.Match(c) { // 连续3缺失
				seqInfo.Stats["DeletionContinuous3"] += count

				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous3"], 1, seqInfo.rowDeletionContinuous3, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowDeletionContinuous3++

				var index = minus3.FindIndex(c)
				if index != nil {
					seqInfo.DeletionContinuous3Index = min(seqInfo.DeletionContinuous3Index, index[0])
				}

				// 输出所有连续3缺失的位置，用于统计断点分布
				var m = dash3.FindAllIndex(c, -1)
				if dashEnd.Match(c) {
					for i := range m {
						var end = m[i][0]
						var seq = string(a)
						if end < 2 {
							var indexSeq = seqInfo.IndexSeq
							seq = string(indexSeq[len(indexSeq)-2:]) + seq
							end += 2
						}
						fmtUtil.Fprintf(seqInfo.del3, "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", end, count, seq[end-2:end], seq[end:end+2], b, c, a)
					}
				}

				// 输出连续3缺失的位置，用于画示意图
				// var m = dash.FindAllIndex(c, -1)
				// for _, bin := range m {
				// 	if bin[1]-bin[0] > 2 {
				// 		fmtUtil.Fprintf(seqInfo.del3, "%d\t%d\t%d", bin[0], bin[1], count)
				// 		break
				// 	}
				// }
				// for _, bin := range m {
				// 	fmtUtil.Fprintf(seqInfo.del3, "\t%d\t%d", bin[0], bin[1])
				// }
				// fmtUtil.Fprintln(seqInfo.del3)

				// 输出末尾缺失的位置，用于统计断点分布
				// var m = dash.FindAllIndex(c, -1)
				// if len(m) == 1 && dashEnd.Match(c) {
				// 	var end = m[0][0]
				// 	var seq = string(a)
				// 	if end < 2 {
				// 		var indexSeq = seqInfo.IndexSeq
				// 		seq = string(indexSeq[len(indexSeq)-2:]) + seq
				// 		end += 2
				// 	}
				// 	fmtUtil.Fprintf(seqInfo.del3, "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", end, count, seq[end-2:end], seq[end:end+2], b, c, a)
				// }
			} else if minus2.Match(c) { // 连续2缺失
				seqInfo.Stats["DeletionContinuous2"] += count

				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 1, seqInfo.rowDeletionContinuous2, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.rowDeletionContinuous2++
			} else { // 离散3缺失
				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete3"], 1, seqInfo.rowDeletionDiscrete3, []interface{}{seqInfo.Seq, key, count, c})
				seqInfo.Stats["DeletionDiscrete3"] += count
				seqInfo.rowDeletionDiscrete3++
			}
		}

		for i, c1 := range c {
			if c1 == '-' {
				seqInfo.DistributionNum[0][i] += count
			}
		}

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
	seqInfo.Stats["ErrorReadsNum"] = seqInfo.Stats["Deletion"] + seqInfo.Stats["ErrorInsReadsNum"] + seqInfo.Stats["ErrorInsDelReadsNum"] + seqInfo.Stats["ErrorMutReadsNum"] + seqInfo.Stats["ErrorOtherReadsNum"]
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

func (seqInfo *SeqInfo) PrintStats(resultDir string) {
	var (
		stats = seqInfo.Stats
		out   = osUtil.Create(filepath.Join(resultDir, seqInfo.Name+".stats.txt"))
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
		stats["Deletion"],
		math2.DivisionInt(stats["Deletion"], stats["ErrorReadsNum"])*100,
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

func MaxNt(A, C, G, T int) (N byte, percent float64) {
	var max = A
	N = 'A'
	if C > max {
		max = C
		N = 'C'
	}
	if G > max {
		max = G
		N = 'G'
	}
	if T > max {
		max = T
		N = 'T'
	}
	return N, float64(max*100) / float64(A+C+G+T)
}

func (seqInfo *SeqInfo) PlotLineACGT(prefix string) {
	var (
		line       = charts.NewLine()
		xaxis      [300]int
		yaxis      [300]int
		output     = osUtil.Create(prefix + ".ACGT.html")
		dnaStorge  [kmerLength]*os.File
		kmerOutput = osUtil.Create(prefix + ".kmer.txt")
	)
	for j := 0; j < kmerLength; j++ {
		dnaStorge[j] = osUtil.Create(prefix + ".dna." + strconv.Itoa(j+1) + ".txt")
	}
	defer simpleUtil.DeferClose(output)
	defer simpleUtil.DeferClose(kmerOutput)

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "A C G T Distribution",
			Subtitle: "in SE150",
		}))

	// print header for dnaStorge
	for j := 0; j < kmerLength; j++ {
		fmtUtil.Fprintf(
			dnaStorge[j],
			"pos\tRefNt\tMaxNt\tpercent\tA\tC\tG\tT\n",
		)
	}
	fmtUtil.Fprintf(
		kmerOutput,
		"pos\tRefNt\tMaxNt\tpercent\tA\tC\tG\tT\n",
	)

	var kmer [kmerLength + 1][]byte
	for i := 0; i < 300; i++ {
		xaxis[i] = i + 1
		yaxis[i] = seqInfo.A[i] + seqInfo.C[i] + seqInfo.G[i] + seqInfo.T[i]

		for j := 0; j < kmerLength; j++ {
			var preKmer = string(kmer[j][:min(j, len(kmer[j]))])
			var dnaKmer = seqInfo.DNAKmer[j][i]
			var N, percent = MaxNt(
				dnaKmer["A"+preKmer],
				dnaKmer["C"+preKmer],
				dnaKmer["G"+preKmer],
				dnaKmer["T"+preKmer],
			)
			fmtUtil.Fprintf(
				dnaStorge[j],
				"%d\t%c\t%c\t%f\t%d\t%d\t%d\t%d\n",
				i+1, seqInfo.DNA[i], N, percent, dnaKmer["A"+preKmer], dnaKmer["C"+preKmer], dnaKmer["G"+preKmer], dnaKmer["T"+preKmer],
			)
			kmer[j] = append([]byte{N}, kmer[j]...)
			if j == kmerLength-1 {
				if i > kmerLength-2 {
					preKmer = string(kmer[j+1][:kmerLength-1])
					dnaKmer = seqInfo.Kmer
					N, percent = MaxNt(
						dnaKmer["A"+preKmer],
						dnaKmer["C"+preKmer],
						dnaKmer["G"+preKmer],
						dnaKmer["T"+preKmer],
					)
				}
				kmer[j+1] = append([]byte{N}, kmer[j+1]...)
				fmtUtil.Fprintf(
					kmerOutput,
					"%d\t%c\t%c\t%f\t%d\t%d\t%d\t%d\n",
					i+1, seqInfo.DNA[i], N, percent, dnaKmer["A"+preKmer], dnaKmer["C"+preKmer], dnaKmer["G"+preKmer], dnaKmer["T"+preKmer],
				)
				dnaKmer[string(N)+preKmer] = 0
			}
		}
	}

	for k, v := range seqInfo.Kmer {
		fmtUtil.Fprintf(kmerOutput, "%s\t%d\n", k, v)
	}

	// close dnaStorge
	for j := 0; j < kmerLength; j++ {
		simpleUtil.CheckErr(dnaStorge[j].Close())
	}

	line.SetXAxis(xaxis).
		AddSeries("A", GenerateLineItems(seqInfo.A[:])).
		AddSeries("C", GenerateLineItems(seqInfo.C[:])).
		AddSeries("G", GenerateLineItems(seqInfo.G[:])).
		AddSeries("T", GenerateLineItems(seqInfo.T[:])).
		AddSeries("ALL", GenerateLineItems(yaxis[:]))
	// SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	simpleUtil.CheckErr(line.Render(output))
	simpleUtil.CheckErr(line.Render(kmerOutput))
}

// WriteStatsSheet writes the statistics sheet for SeqInfo.
//
// No parameters.
// No return values.
func (seqInfo *SeqInfo) WriteStatsSheet(resultDir string) {
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

		out  = osUtil.Create(filepath.Join(resultDir, seqInfo.Name+".steps.txt"))
		oser = osUtil.Create(filepath.Join(resultDir, seqInfo.Name+".one.step.error.rate.txt"))
	)
	defer simpleUtil.DeferClose(out)
	defer simpleUtil.DeferClose(oser)

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
		sequence  string
	)
	if seqInfo.Reverse {
		sequence = "AAAA" + string(seqInfo.Seq)

	} else {
		sequence = seqInfo.IndexSeq[len(seqInfo.IndexSeq)-4:] + string(seqInfo.Seq)
	}
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
			oser,
			"%s\t%s\t%c\t%d\t%f\n",
			seqInfo.Name,
			sequence[i:i+4],
			sequence[i+4],
			i+1,
			(1-ratio[b])*100,
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
	// free seqInfo.HitSeqCount
	seqInfo.HitSeqCount = nil

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

// WriteStatsTxt writes the statistics of SeqInfo to a text file.
//
// Parameters:
// - file: a pointer to an os.File object representing the file to write the statistics to.
//
// Return type: none.
func (info *SeqInfo) WriteStatsTxt(file *os.File) {
	// Get the statistics from SeqInfo
	stats := info.Stats

	// Format the statistics into a string
	statsString := fmt.Sprintf(
		"%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n",
		info.Name, info.IndexSeq, info.Seq, len(info.Seq),
		stats["AllReadsNum"], stats["IndexReadsNum"], stats["AnalyzedReadsNum"], stats["RightReadsNum"],
		info.YieldCoefficient, info.AverageYieldAccuracy,
		math2.DivisionInt(stats["ErrorReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["Deletion"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionSingle"], stats["AnalyzedReadsNum"]),

		math2.DivisionInt(stats["DeletionDiscrete2"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionContinuous2"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionDiscrete3"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorInsReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorInsDelReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorMutReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorOtherReadsNum"], stats["AnalyzedReadsNum"]),
	)

	// Write the statistics string to the file
	fmtUtil.Fprintf(file, statsString)
}

func (info *SeqInfo) SummaryRow() []interface{} {
	var stats = info.Stats
	return []interface{}{
		info.Name, info.IndexSeq, info.Seq, len(info.Seq),
		stats["AllReadsNum"], stats["IndexReadsNum"], stats["AnalyzedReadsNum"], stats["RightReadsNum"],
		info.YieldCoefficient, info.AverageYieldAccuracy,
		math2.DivisionInt(stats["ErrorReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["Deletion"], stats["AnalyzedReadsNum"]),

		math2.DivisionInt(stats["DeletionSingle"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionContinuous2"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionContinuous3"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionDiscrete2"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["DeletionDiscrete3"], stats["AnalyzedReadsNum"]),
		info.DeletionContinuous3Index + 1,
		math2.DivisionInt(stats["ErrorInsReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorInsDelReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorMutReadsNum"], stats["AnalyzedReadsNum"]),
		math2.DivisionInt(stats["ErrorOtherReadsNum"], stats["AnalyzedReadsNum"]),
	}

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
