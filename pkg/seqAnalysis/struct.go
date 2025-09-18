package seqAnalysis

import (
	"fmt"
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/liserjrqlxue/goUtil/fmtUtil"
	math2 "github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/stringsUtil"
	"github.com/xuri/excelize/v2"
)

const (
	kmerLength = 9
)

var (
	Short = 0
)

// regexp
var (
	plus3  = regexp.MustCompile(`\+\+\+`)
	minus1 = regexp.MustCompile(`-`)
	minus2 = regexp.MustCompile(`--`)
	minus3 = regexp.MustCompile(`---`)

	regN = regexp.MustCompile(`N`)
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
	if len(p.YieldCoefficient) == 1 {
		p.YieldCoefficientSD = 0
	}
	p.AverageYieldAccuracyMean, p.AverageYieldAccuracySD = math2.MeanStdDev(p.AverageYieldAccuracy)
	if len(p.AverageYieldAccuracy) == 1 {
		p.AverageYieldAccuracySD = 0
	}
}

type SeqInfo struct {
	Name           string
	ParallelTestID string
	Excel          string

	UseReverseComplement bool
	AssemblerMode        bool
	Reverse              bool

	lineLimit int
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

	Seq         []byte // Target Synthesis Seq
	Align       []byte
	AlignInsert []byte
	AlignMut    []byte

	IndexSeq  string
	PostSeq   string
	Fastqs    []string
	SeqChan   chan string
	SeqChanWG sync.WaitGroup

	// SeqResultTxt *os.File
	RegPolyA    *regexp.Regexp
	RegIndexSeq *regexp.Regexp

	LessMem            bool
	HitSeq             []string
	HitSeqCount        map[string]int
	Stats              map[string]int
	AllReadsNum        int
	IndexReadsNum      int
	IndexPolyAReadsNum int
	RightReadsNum      int
	ExcludeReadsNum    int

	DistributionNum  [4][]int
	DistributionFreq [4][]float64

	// fastq
	// ReadsLength map[int]int
	A   [300]int
	C   [300]int
	G   [300]int
	T   [300]int
	DNA [300]byte
	// value:weight -> length:count
	Histogram map[int]int

	UseKmer bool
	DNAKmer [kmerLength][300]map[string]int
	Kmer    map[string]int

	// summary
	// 收率
	YieldCoefficient     float64
	AverageYieldAccuracy float64

	// One-step accuracy rate
	OSAR float64

	// 最高频序列
	HighFreqSeq   string
	HighFreqCount int
}

func NewSeqInfo(data, Sheets map[string]string, sheetList []string, outputDir string, lineLimit int, long, rev, useRC, useKmer, lessMem bool) *SeqInfo {
	var seqInfo = new(SeqInfo)
	seqInfo = &SeqInfo{
		Name:           data["id"],
		ParallelTestID: data["平行"],
		IndexSeq:       strings.ToUpper(data["index"]),
		PostSeq:        strings.ToUpper(data["postBase"]),
		Seq:            []byte(strings.ToUpper(data["seq"])),
		Fastqs:         strings.Split(data["fq"], ","),
		SeqChan:        make(chan string, 102400),

		Excel:     filepath.Join(outputDir, data["id"]+".xlsx"),
		Sheets:    Sheets,
		SheetList: sheetList,
		lineLimit: lineLimit,

		Stats:       make(map[string]int),
		HitSeqCount: make(map[string]int),
		Histogram:   make(map[int]int),
		// ReadsLength:          make(map[int]int),
		AssemblerMode:        long,
		Reverse:              rev,
		UseReverseComplement: useRC,
		UseKmer:              useKmer,
		LessMem:              lessMem,
	}

	seqInfo.SeqChanWG.Add(len(seqInfo.Fastqs))
	// close seqInfo.SeqChan after seqInfo.SeqChanWG
	go func() {
		seqInfo.SeqChanWG.Wait()
		slog.Info("close seqInfo.SeqChan", "name", seqInfo.Name)
		close(seqInfo.SeqChan)
	}()

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
	seqInfo.Style["center"] = simpleUtil.HandleError(seqInfo.xlsx.NewStyle(center))

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

func (seqInfo *SeqInfo) SingleRun(resultDir string, TitleTar, TitleStats []string) {
	slog.Debug("SingleRun Init", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.Init()
	slog.Debug("SingleRun CountError", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.CountError4(resultDir)

	slog.Debug("SingleRun WriteStatsSheet", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.WriteStatsSheet(resultDir, TitleTar, TitleStats)
	slog.Debug("SingleRun Save", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.Save()
	slog.Debug("SingleRun PrintStats", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.PrintStats(resultDir)

	slog.Debug("SingleRun PlotLineACGT", slog.Group("seqInfo", "name", seqInfo.Name))
	prefix := filepath.Join(resultDir, seqInfo.Name)
	seqInfo.PlotLineACGT(prefix)
	if seqInfo.UseKmer {
		slog.Info("SingleRun WriteKmer", slog.Group("seqInfo", "name", seqInfo.Name))
		seqInfo.WriteKmer(prefix)
	}
}

func (seqInfo *SeqInfo) Save() {
	slog.Info("save xlsx", slog.Group("seqInfo", "name", seqInfo.Name, "path", seqInfo.Excel))
	simpleUtil.CheckErr(seqInfo.xlsx.SaveAs(seqInfo.Excel))
	slog.Info("free xlsx", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.xlsx = nil
}

// CountError4 count seq error
func (seqInfo *SeqInfo) CountError4(outputDir string) {
	defer func() {
		slog.Debug("CountError4 Done or Error", slog.Group("seqInfo", "name", seqInfo.Name))
		if r := recover(); r != nil {
			slog.Error("CountError4", slog.Group("seqInfo", "name", seqInfo.Name, "error", r))
		}
	}()
	// 1. 统计不同测序结果出现的频数
	slog.Debug("CountError4 WriteSeqResult", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.WriteSeqResult(".SeqResult.txt", outputDir)

	slog.Debug("CountError4 GetHitSeq", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.GetHitSeq()

	// 2. 与正确合成序列进行比对,统计不同合成结果出现的频数
	seqInfo.del3 = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+".del3.txt"))
	seqInfo.del1 = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+".del1.txt"))
	if seqInfo.LessMem {
		slog.Debug("CountError4 WriteHitSeqLessMem", slog.Group("seqInfo", "name", seqInfo.Name))
		seqInfo.WriteHitSeqLessMem()
	} else {
		slog.Debug("CountError4 WriteHitSeq", slog.Group("seqInfo", "name", seqInfo.Name))
		seqInfo.WriteHitSeq()
	}
	slog.Debug("CountError4 WriteSeqResultNum", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.WriteSeqResultNum()

	slog.Debug("CountError4 UpdateDistributionStats", slog.Group("seqInfo", "name", seqInfo.Name))
	seqInfo.UpdateDistributionStats()

	//seqInfo.PrintStats()
}

func (seqInfo *SeqInfo) WriteSeqResult(path, outputDir string) {
	defer slog.Debug("WriteSeqResult Done or Error", slog.Group("seqInfo", "name", seqInfo.Name))
	var (
		tarSeq   = string(seqInfo.Seq)
		indexSeq = seqInfo.IndexSeq
		postSeq  = seqInfo.PostSeq
	)
	if postSeq == "" {
		postSeq = "AAAAAAAA"
	}
	var regPost = regexp.MustCompile(postSeq)

	// seqInfo.RegPolyA = regexp.MustCompile(`^` + indexSeq + `(.*?)` + postSeq)
	// seqInfo.RegIndexSeq = regexp.MustCompile(`^` + indexSeq + `(.*?)$`)

	seqInfo.RegPolyA = regexp.MustCompile(indexSeq + `(.*?)` + postSeq)
	seqInfo.RegIndexSeq = regexp.MustCompile(indexSeq + `(.*?)$`)

	// seqInfo.SeqResultTxt = osUtil.Create(filepath.Join(outputDir, seqInfo.Name+path))
	// defer simpleUtil.DeferClose(seqInfo.SeqResultTxt)

	if indexSeq == "" {
		seqInfo.RegPolyA = regexp.MustCompile(`^(.*?)` + postSeq)
		seqInfo.RegIndexSeq = regexp.MustCompile(`^(.*?)` + postSeq)
		seqInfo.UseReverseComplement = false
	}
	if tarSeq == "A" || tarSeq == "AAAAAAAAAAAAAAAAAAAA" {
		seqInfo.RegPolyA = regexp.MustCompile(`^` + indexSeq + `(.*?)TTTTTTTT`)
		seqInfo.RegIndexSeq = regexp.MustCompile(`^` + indexSeq + `(.*?)$`)
	}
	slog.Debug("RegPolyA", slog.Group("seqInfo", "name", seqInfo.Name, "reg", seqInfo.RegPolyA.String()))
	slog.Debug("RegIndexSeq", slog.Group("seqInfo", "name", seqInfo.Name, "reg", seqInfo.RegIndexSeq.String()))

	slog.Debug("WriteSeqResult Write1SeqResult", slog.Group("seqInfo", "name", seqInfo.Name))
	for s := range seqInfo.SeqChan {
		seqInfo.Write1SeqResult(s, regPost)
	}

	// update Stats
	seqInfo.Stats["IndexReadsNum"] = seqInfo.IndexReadsNum
	seqInfo.Stats["AllReadsNum"] = seqInfo.AllReadsNum
	seqInfo.Stats["RightReadsNum"] = seqInfo.RightReadsNum
	seqInfo.Stats["AnalyzedReadsNum"] = seqInfo.RightReadsNum + seqInfo.IndexPolyAReadsNum

	slog.Debug("WriteSeqResult WriteHistogram", slog.Group("seqInfo", "name", seqInfo.Name))
	// output histgram.txt
	WriteHistogram(filepath.Join(outputDir, seqInfo.Name+".histogram.txt"), seqInfo.Histogram)

	slog.Debug("WriteSeqResult Done", slog.Group("seqInfo", "name", seqInfo.Name))
}

func (seqInfo *SeqInfo) Write1SeqResult(s string, reg *regexp.Regexp) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Write1SeqResult", slog.Group("seqInfo", "name", seqInfo.Name, "error", r, "reg", reg))
		}
	}()
	seqInfo.AllReadsNum++
	submatch, byteS, indexSeqMatch := MatchSeq(s, seqInfo.RegPolyA, seqInfo.RegIndexSeq, seqInfo.UseReverseComplement, seqInfo.AssemblerMode)

	if indexSeqMatch {
		seqInfo.IndexReadsNum++
	}

	seqInfo.UpdateACGT(byteS)
	// trim byteS from polyA
	var byteSloc = reg.FindIndex(byteS)
	if byteSloc != nil {
		byteS = byteS[:byteSloc[0]]
	}

	if seqInfo.UseKmer {
		seqInfo.UpdateKmer(byteS)
	}

	if submatch != nil {
		tSeq := submatch[1] //[seqInfo.Offset:]
		// fmtUtil.Fprintln(seqInfo.SeqResultTxt, tSeq)

		// 过滤 len(seq)<=Short
		if Short > 0 && len(tSeq) <= Short {
			seqInfo.ExcludeReadsNum++
			return
		}

		seqInfo.Histogram[len(tSeq)]++

		seqInfo.UpdateHitSeqCount(string(seqInfo.Seq), tSeq)
	}
}

func (seqInfo *SeqInfo) UpdateKmer(byteS []byte) {
	var kmer []byte
	for i, c := range byteS {
		if i < 300 {
			kmer = append([]byte{c}, kmer...)
			for j := 0; j < kmerLength; j++ {
				var n = min(j+1, len(kmer))
				var key = string(kmer[:n])
				seqInfo.DNAKmer[j][i][key]++
				if n == kmerLength {
					seqInfo.Kmer[key]++
				}
			}
		}
	}
}

func (seqInfo *SeqInfo) UpdateHitSeqCount(tarSeq, seq string) {
	if seqInfo.Reverse {
		seq = string(Reverse([]byte(seq)))
	}

	if len(seq) == 0 {
		seq += "X"
		seqInfo.HitSeqCount[seq]++
		seqInfo.IndexPolyAReadsNum++
	} else if seq == tarSeq {
		seqInfo.RightReadsNum++
		seqInfo.HitSeqCount[seq]++
	} else if !regN.MatchString(seq) {
		seqInfo.HitSeqCount[seq]++
		seqInfo.IndexPolyAReadsNum++
	} else {
		//fmt.Printf("[%s]:[%s]:[%+v]\n", s, tSeq, m)
		seqInfo.ExcludeReadsNum++
	}
}

func (seqInfo *SeqInfo) UpdateACGT(seq []byte) {
	for i, c := range seq {
		if i < 300 {
			switch c {
			case 'A':
				seqInfo.A[i]++
			case 'C':
				seqInfo.C[i]++
			case 'G':
				seqInfo.G[i]++
			case 'T':
				seqInfo.T[i]++
			}
		}
	}
}

func (seqInfo *SeqInfo) GetHitSeq() {
	defer slog.Debug("WriteSeqResult GetHitSeq Done or Error", slog.Group("seqInfo", "name", seqInfo.Name))
	slog.Debug("GetHitSeq append", slog.Group("seqInfo", "name", seqInfo.Name, "len", len(seqInfo.HitSeqCount)))
	for k := range seqInfo.HitSeqCount {
		seqInfo.HitSeq = append(seqInfo.HitSeq, k)
	}
	slog.Debug("GetHitSeq sort", slog.Group("seqInfo", "name", seqInfo.Name, "len", len(seqInfo.HitSeqCount)))
	sort.Slice(seqInfo.HitSeq, func(i, j int) bool {
		return seqInfo.HitSeqCount[seqInfo.HitSeq[i]] > seqInfo.HitSeqCount[seqInfo.HitSeq[j]]
	})
	slog.Debug("GetHitSeq Done", slog.Group("seqInfo", "name", seqInfo.Name, "len", len(seqInfo.HitSeqCount)))
}

func (seqInfo *SeqInfo) WriteHitSeqLessMem() {
	defer func() {
		slog.Debug("WriteSeqResult WriteHitSeqLessMem Done or Error", slog.Group("seqInfo", "name", seqInfo.Name))
		if r := recover(); r != nil {
			slog.Error("WriteSeqResult WriteHitSeqLessMem", slog.Group("seqInfo", "name", seqInfo.Name, "error", r))
		}
	}()
	for i, key := range seqInfo.HitSeq {
		var keep = true
		// if i == 0 {
		// seqInfo.HighFreqSeq = key
		// seqInfo.HighFreqCount = seqInfo.HitSeqCount[key]
		// slog.Info("高频序列", "Name", seqInfo.Name, "HighFreqSeq", seqInfo.HighFreqSeq, "HighFreqCount", seqInfo.HighFreqCount)
		// }
		if i > seqInfo.lineLimit+2 {
			keep = false
		}
		if key == string(seqInfo.Seq) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key]})
			seqInfo.rowDeletion++
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key]})
			continue
		}
		if seqInfo.Align1(key, keep) {
			if keep {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align})
			}
			continue
		}

		if seqInfo.Align2(key, keep) {
			if keep {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert})
			}
			continue
		}

		if seqInfo.Align3(key, keep) {
			if keep {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
			}
			continue
		}
		if keep {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Other"], 1, seqInfo.rowOther, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert, seqInfo.AlignMut})
			seqInfo.rowOther++
		}
		seqInfo.Stats["ErrorOtherReadsNum"] += seqInfo.HitSeqCount[key]
	}
	// seqInfo.HighFreqSeq = seqInfo.HitSeq[0]
	// seqInfo.HighFreqCount = seqInfo.HitSeqCount[seqInfo.HighFreqSeq]
	// slog.Info("高频序列", "Name", seqInfo.Name, "HighFreqSeq", seqInfo.HighFreqSeq, "HighFreqCount", seqInfo.HighFreqCount)
	// free HitSeq
	seqInfo.HitSeq = nil
}

func (seqInfo *SeqInfo) WriteHitSeq() {
	var keep = true
	for i, key := range seqInfo.HitSeq {
		if key == string(seqInfo.Seq) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key]})
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []interface{}{seqInfo.Seq, key, seqInfo.HitSeqCount[key]})
			seqInfo.rowDeletion++
			continue
		}
		if seqInfo.Align1(key, keep) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align})
			continue
		}

		if seqInfo.Align2(key, keep) {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["BarCode"], 1, i+1, []interface{}{key, seqInfo.HitSeqCount[key], seqInfo.Align, seqInfo.AlignInsert})
			continue
		}

		if seqInfo.Align3(key, keep) {
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
}

func (seqInfo *SeqInfo) WriteSeqResultNum() {
	WriteUpperDownNIL(seqInfo.del3, seqInfo.IndexSeq, string(seqInfo.Seq), 3)
	simpleUtil.CheckErr(seqInfo.del3.Close())
	simpleUtil.CheckErr(seqInfo.del1.Close())

	SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 5, 1,
		[]any{"总数", seqInfo.Stats["Deletion"] + seqInfo.RightReadsNum},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionSingle"], 5, 1,
		[]any{"总数", seqInfo.Stats["DeletionSingle"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete2"], 5, 1,
		[]any{"总数", seqInfo.Stats["DeletionDiscrete2"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 5, 1,
		[]any{"总数", seqInfo.Stats["DeletionContinuous2"]},
	)
	SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete3"], 5, 1,
		[]any{"总数", seqInfo.Stats["DeletionDiscrete3"]},
	)

	var sheet = seqInfo.Sheets["Deletion"]
	for i := 3; i < seqInfo.rowDeletion; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), "from Deletion: "+seqInfo.Name)
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["Deletion"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionSingle"]
	for i := 2; i < seqInfo.rowDeletionSingle; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), "from DeletionSingle: "+seqInfo.Name)
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionSingle"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionDiscrete2"]
	for i := 2; i < seqInfo.rowDeletionDiscrete2; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), "from DeletionDiscrete2: "+seqInfo.Name)
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionDiscrete2"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionDiscrete3"]
	for i := 2; i < seqInfo.rowDeletionDiscrete3; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), "from DeletionDiscrete3: ", seqInfo.Name)
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionDiscrete3"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}
	sheet = seqInfo.Sheets["DeletionContinuous2"]
	for i := 2; i < seqInfo.rowDeletionContinuous2; i++ {
		var count = stringsUtil.Atoi(GetCellValue(seqInfo.xlsx, sheet, 3, i), "from DeletionContinuous2: ", seqInfo.Name)
		SetCellValue(seqInfo.xlsx, sheet, 5, i, math2.DivisionInt(count, seqInfo.Stats["DeletionContinuous2"]))
		SetCellValue(seqInfo.xlsx, sheet, 6, i, math2.DivisionInt(count, seqInfo.Stats["AnalyzedReadsNum"]))
	}

}

// var dash = regexp.MustCompile(`-+`)
var dash3 = regexp.MustCompile(`---+`)
var dashEnd = regexp.MustCompile(`-$`)

func (seqInfo *SeqInfo) Align1(sequencingSeqStr string, keep bool) bool {
	var (
		targetSynthesisSeq  = seqInfo.Seq
		sequencingSeq       = []byte(sequencingSeqStr)
		sequencingAlignment []byte

		count    = seqInfo.HitSeqCount[sequencingSeqStr]
		delCount = 0
	)

	if len(targetSynthesisSeq) == 1 && len(sequencingSeq) == 1 && sequencingSeq[0] == 'X' {
		sequencingAlignment = append(sequencingAlignment, '-')
		seqInfo.Align = sequencingAlignment
		seqInfo.DistributionNum[0][0] += count
		seqInfo.Stats["Deletion"] += count
		return true
	}

	var k = 0 // match count to Seq
	for i := range targetSynthesisSeq {
		if k < len(sequencingSeq) && (targetSynthesisSeq[i] == sequencingSeq[k] || targetSynthesisSeq[i] == 'N') {
			sequencingAlignment = append(sequencingAlignment, sequencingSeq[k])
			k++
		} else {
			sequencingAlignment = append(sequencingAlignment, '-')
			delCount++
		}
	}
	seqInfo.Align = sequencingAlignment
	//if k >= len(b) && !minus3.Match(c) { // all match
	if k >= len(sequencingSeq) { // all match
		seqInfo.Stats["Deletion"] += count

		if keep {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Deletion"], 1, seqInfo.rowDeletion, []any{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
			seqInfo.rowDeletion++
		}

		if delCount == 1 { // 单个缺失
			seqInfo.Stats["DeletionSingle"] += count

			if keep {
				SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionSingle"], 1, seqInfo.rowDeletionSingle, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
				seqInfo.rowDeletionSingle++

			}

			var m = minus1.FindIndex(sequencingAlignment)
			if m != nil {
				if m[0] == 0 {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, '^', targetSynthesisSeq[m[1]-1], sequencingAlignment[m[1]])
				} else if m[1] == len(sequencingAlignment) {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, sequencingAlignment[m[0]-1], targetSynthesisSeq[m[1]-1], '$')
				} else {
					fmtUtil.Fprintf(seqInfo.del1, "%d\t%d\t%d\t%c\t%c\t%c\n", m[0], m[1], count, sequencingAlignment[m[0]-1], targetSynthesisSeq[m[1]-1], sequencingAlignment[m[1]])
				}
			}

		} else if delCount == 2 { // 2缺失
			seqInfo.Stats["Deletion2"] += count

			if minus2.Match(sequencingAlignment) { // 连续2缺失
				seqInfo.Stats["DeletionContinuous2"] += count

				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 1, seqInfo.rowDeletionContinuous2, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
					seqInfo.rowDeletionContinuous2++
				}
			} else { // 离散2缺失
				seqInfo.Stats["DeletionDiscrete2"] += count

				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete2"], 1, seqInfo.rowDeletionDiscrete2, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
					seqInfo.rowDeletionDiscrete2++
				}
			}
		} else if delCount >= 3 {
			seqInfo.Stats["Deletion3"] += count

			if minus3.Match(sequencingAlignment) { // 连续3缺失
				seqInfo.Stats["DeletionContinuous3"] += count

				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous3"], 1, seqInfo.rowDeletionContinuous3, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
					seqInfo.rowDeletionContinuous3++
				}

				var index = minus3.FindIndex(sequencingAlignment)
				if index != nil {
					seqInfo.DeletionContinuous3Index = min(seqInfo.DeletionContinuous3Index, index[0])
				}

				// 输出所有连续3缺失的位置，用于统计断点分布
				var m = dash3.FindAllIndex(sequencingAlignment, -1)
				if dashEnd.Match(sequencingAlignment) {
					WriteUpperDown(seqInfo.del3, seqInfo.IndexSeq, string(targetSynthesisSeq), 3, count, m)
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
			} else if minus2.Match(sequencingAlignment) { // 连续2缺失
				seqInfo.Stats["DeletionContinuous2"] += count

				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionContinuous2"], 1, seqInfo.rowDeletionContinuous2, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
					seqInfo.rowDeletionContinuous2++
				}
			} else { // 离散3缺失
				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["DeletionDiscrete3"], 1, seqInfo.rowDeletionDiscrete3, []interface{}{seqInfo.Seq, sequencingSeqStr, count, sequencingAlignment})
					seqInfo.rowDeletionDiscrete3++
				}
				seqInfo.Stats["DeletionDiscrete3"] += count
			}
		}

		for i, c1 := range sequencingAlignment {
			if c1 == '-' {
				seqInfo.DistributionNum[0][i] += count
			}
		}

		return true
	}
	return false
}

// Align2 aligns insertions with the key.
//
// Parameters:
// - key: a string representing the key to align with.
// - keep: a boolean indicating whether to keep the alignment result.
//
// Returns:
// - a boolean indicating whether the alignment was successful.
func (seqInfo *SeqInfo) Align2(key string, keep bool) bool {

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
				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["InsertionDeletion"], 1, seqInfo.rowInsertionDeletion, []interface{}{seqInfo.Seq, key, count, c})
				}
				seqInfo.rowInsertionDeletion++
				seqInfo.Stats["ErrorInsDelReadsNum"] += count
			} else {
				if keep {
					SetRow(seqInfo.xlsx, seqInfo.Sheets["Insertion"], 1, seqInfo.rowInsertion, []interface{}{seqInfo.Seq, key, count, c})
					seqInfo.rowInsertion++
				}
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

func (seqInfo *SeqInfo) Align3(key string, keep bool) bool {
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
		if keep {
			SetRow(seqInfo.xlsx, seqInfo.Sheets["Mutation"], 1, seqInfo.rowMutation, []interface{}{seqInfo.Seq, key, count, c})
			seqInfo.rowMutation++
		}
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
	seqInfo.Stats["ExcludeOtherReadsNum"] = seqInfo.RightReadsNum + seqInfo.Stats["ErrorReadsNum"] - seqInfo.Stats["ErrorOtherReadsNum"]
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
		seqInfo.AllReadsNum,
	)
	fmtUtil.Fprintf(out,
		"+ShortReadsNum\t\t= %d\t%7.4f%%\n",
		stats["ShortReadsNum"],
		math2.DivisionInt(stats["ShortReadsNum"], seqInfo.AllReadsNum)*100,
	)
	// fmtUtil.Fprintf(out,
	// 	"+UnmatchedReadsNum\t= %d\t%7.4f%%\n",
	// 	stats["UnmatchedReadsNum"],
	// 	math2.DivisionInt(stats["UnmatchedReadsNum"], seqInfo.AllReadsNum)*100,
	// )
	fmtUtil.Fprintf(out,
		"+ExcludeReadsNum\t= %d\t%7.4f%%\n",
		seqInfo.ExcludeReadsNum,
		math2.DivisionInt(seqInfo.ExcludeReadsNum, seqInfo.AllReadsNum)*100,
	)
	fmtUtil.Fprintf(out,
		"+IndexReadsNum\t\t= %d\t%.4f%%\n",
		seqInfo.IndexReadsNum,
		math2.DivisionInt(seqInfo.IndexReadsNum, seqInfo.AllReadsNum)*100,
	)
	fmtUtil.Fprintf(out,
		"+AnalyzedReadsNum\t= %d\t%.4f%%\n",
		stats["AnalyzedReadsNum"],
		math2.DivisionInt(stats["AnalyzedReadsNum"], seqInfo.IndexReadsNum)*100,
	)
	fmtUtil.Fprintf(out,
		"++RightReadsNum\t\t= %d\t%.4f%%\n",
		seqInfo.RightReadsNum,
		math2.DivisionInt(seqInfo.RightReadsNum, stats["AnalyzedReadsNum"])*100,
	)
	fmtUtil.Fprintf(out,
		"++IndexPolyAReadsNum\t= %d\t%.4f%%\n",
		seqInfo.IndexPolyAReadsNum,
		math2.DivisionInt(seqInfo.IndexPolyAReadsNum, stats["AnalyzedReadsNum"])*100,
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
		line   = charts.NewLine()
		xaxis  [300]int
		yaxis  [300]int
		output = osUtil.Create(prefix + ".ACGT.html")
	)
	defer simpleUtil.DeferClose(output)

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    "A C G T Distribution",
			Subtitle: "in SE150",
		}))

	for i := 0; i < 300; i++ {
		xaxis[i] = i + 1
		yaxis[i] = seqInfo.A[i] + seqInfo.C[i] + seqInfo.G[i] + seqInfo.T[i]
	}

	line.SetXAxis(xaxis).
		AddSeries("A", GenerateLineItems(seqInfo.A[:])).
		AddSeries("C", GenerateLineItems(seqInfo.C[:])).
		AddSeries("G", GenerateLineItems(seqInfo.G[:])).
		AddSeries("T", GenerateLineItems(seqInfo.T[:])).
		AddSeries("ALL", GenerateLineItems(yaxis[:]))
	// SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))
	simpleUtil.CheckErr(line.Render(output))
}

func (seqInfo *SeqInfo) WriteKmer(prefix string) {
	var (
		dnaStorge  [kmerLength]*os.File
		kmerOutput = osUtil.Create(prefix + ".kmer.txt")
	)
	for j := 0; j < kmerLength; j++ {
		dnaStorge[j] = osUtil.Create(prefix + ".dna." + strconv.Itoa(j+1) + ".txt")
	}
	defer simpleUtil.DeferClose(kmerOutput)

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
}

// WriteStatsSheet writes the statistics sheet for SeqInfo.
//
// No parameters.
// No return values.
func (seqInfo *SeqInfo) WriteStatsSheet(resultDir string, TitleTar, TitleStats []string) {
	var (
		stats = seqInfo.Stats
		xlsx  = seqInfo.xlsx
		sheet = seqInfo.Sheets["Stats"]
		rIdx  = 1

		titleTar     = TitleTar
		titleStats   = TitleStats
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
	statsMap["Accuracy"] = math2.DivisionInt(seqInfo.RightReadsNum, stats["AnalyzedReadsNum"])
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

		fmtUtil.Fprintf(
			oser,
			"%s\t%s\t%c\t%d\t%f\t%d\t%d\n",
			seqInfo.Name,
			sequence[i:i+4],
			sequence[i+4],
			i+1,
			(1-ratio[b])*100,
			readsCount, counts[b],
		)

		readsCount = counts[b]

		SetRow(xlsx, sheet, 1, rIdx, rowValue)
		rIdx++

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
	seqInfo.HitSeqCount = make(map[string]int)
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
		"%s\t%s\t%st\t%s\t%d\t%d\t%d\t%d\t%d\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n",
		info.Name, info.IndexSeq, info.Seq, info.PostSeq, len(info.Seq),
		info.AllReadsNum, info.IndexReadsNum, stats["AnalyzedReadsNum"], info.RightReadsNum,
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

func (info *SeqInfo) SummaryRow() []any {
	var stats = info.Stats
	return []any{
		info.Name, info.IndexSeq, info.Seq, len(info.Seq),
		info.AllReadsNum, info.IndexReadsNum, stats["AnalyzedReadsNum"], info.RightReadsNum,
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
