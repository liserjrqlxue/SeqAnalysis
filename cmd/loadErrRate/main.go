package main

import (
	"flag"
	"log"
	math2 "math"
	"sort"

	"github.com/liserjrqlxue/goUtil/fmtUtil"
	"github.com/liserjrqlxue/goUtil/math"
	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/liserjrqlxue/goUtil/stringsUtil"
	"github.com/liserjrqlxue/goUtil/textUtil"
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
	rec = flag.Bool(
		"rec",
		false,
		"recursive",
	)
)

type ErrRate struct {
	ID        string
	Base4     string
	Synthesis string
	Pos       int
	ErrRate   float64
	B4s       string
}

var AllData = make(map[string][]*ErrRate)

func main() {
	flag.Parse()
	if *input == "" || *output == "" {
		flag.PrintDefaults()
		log.Fatal("-i/-o required!")
	}
	var data, _ = textUtil.File2MapArray(*input, "\t", nil)
	var out = osUtil.Create(*output)
	defer simpleUtil.DeferClose(out)

	for _, item := range data {
		key := item["b4s"]
		errRate := &ErrRate{
			ID:        item["id"],
			Base4:     item["base4"],
			Synthesis: item["synthesis"],
			Pos:       stringsUtil.Atoi(item["pos"]),
			ErrRate:   stringsUtil.Atof(item["errRate"]),
			B4s:       key,
		}
		errRates, ok := AllData[key]
		if !ok {
			errRates = make([]*ErrRate, 0)
		}
		errRates = append(errRates, errRate)
		AllData[key] = errRates
	}
	fmtUtil.Fprintf(out, "key\tcount\tmean\tsd\tgMean\tmin\tmax\t")
	fmtUtil.Fprintf(out, "count\tmean\tsd\tgMean\tmin\tmax\tID\tPOS\terrRate\n")
	var sortKeys = make([]string, 0, len(AllData))
	for key := range AllData {
		sortKeys = append(sortKeys, key)
	}
	sort.Strings(sortKeys)
	for _, key := range sortKeys {
		items := AllData[key]
		var errRate []float64
		for _, v := range items {
			errRate = append(errRate, v.ErrRate)
		}
		sort.Float64s(errRate)

		var mean, sd, gMean, min, max = stat(errRate)
		fmtUtil.Fprintf(out, "%s\t%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t", key, len(errRate), mean, sd, gMean, min, max)

		var filterErrRate = filterOutliers(errRate)
		if *rec {
			filterErrRate = filterOutliersREC(filterErrRate)
		}
		mean, sd, gMean, min, max = stat(filterErrRate)
		fmtUtil.Fprintf(out, "%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\n", len(filterErrRate), mean, sd, gMean, min, max)

		for _, v := range items {
			fmtUtil.Fprintf(out, "\t\t\t\t\t\t\t\t\t\t\t\t\t%s\t%d\t%.2f\n", v.ID, v.Pos, v.ErrRate)
		}
	}
}

// 统计期望、标准差、最大值、最小值
func stat(sortedData []float64) (mean, sd, gMean, min, max float64) {
	mean, sd = math.MeanStdDev(sortedData)
	min = sortedData[0]
	max = sortedData[len(sortedData)-1]
	gMean = geometricMean100(sortedData)
	return mean, sd, gMean, min, max
}

// 过滤离群值的函数
func filterOutliers(sortedData []float64) []float64 {
	// 计算数据的四分位数
	q1, q3 := quartiles(sortedData)

	// 计算IQR（四分位距）
	iqr := q3 - q1

	// 确定离群值的范围
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// 过滤数据，排除离群值
	var cleanedData []float64
	for _, value := range sortedData {
		if value >= lowerBound && value <= upperBound {
			cleanedData = append(cleanedData, value)
		}
	}

	return cleanedData
}

// 过滤离群值的函数，递归版本
func filterOutliersREC(sortedData []float64) []float64 {
	// 计算数据的四分位数
	q1, q3 := quartiles(sortedData)

	// 计算IQR（四分位距）
	iqr := q3 - q1

	// 确定离群值的范围
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// 过滤数据，排除离群值
	var cleanedData []float64
	for _, value := range sortedData {
		if value >= lowerBound && value <= upperBound {
			cleanedData = append(cleanedData, value)
		}
	}

	if len(cleanedData) < len(sortedData) {
		return filterOutliersREC(cleanedData)
	}
	return cleanedData
}

// 计算四分位数的函数
// 输入排序后的数据
func quartiles(sortedData []float64) (q1, q3 float64) {
	// 计算Q1和Q3
	n := len(sortedData)
	q1Index := n / 4
	q3Index := 3 * n / 4

	q1 = sortedData[q1Index]
	q3 = sortedData[q3Index]

	return q1, q3
}

// 计算一组数的几何平均数
func geometricMean(numbers []float64) float64 {
	// 初始化乘积
	product := 1.0

	// 计算乘积
	for _, num := range numbers {
		product *= num
	}

	// 计算几何平均数
	geometricMean := math2.Pow(product, 1.0/float64(len(numbers)))

	return geometricMean
}

// 计算 1-x 的几何平均数
func geometricMean100(numbers []float64) float64 {
	// 初始化乘积
	product := 1.0

	// 计算乘积
	for _, num := range numbers {
		product *= (100 - num)
	}

	// 计算几何平均数
	geometricMean := math2.Pow(product, 1.0/float64(len(numbers)))

	return 100 - geometricMean
}
