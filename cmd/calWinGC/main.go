package main

import (
	"flag"
	"log"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// flag
var (
	input1 = flag.String(
		"i1",
		"",
		"input seq1",
	)
	input2 = flag.String(
		"i2",
		"",
		"input seq2",
	)
	window = flag.Int(
		"w",
		10,
		"window",
	)
	name = flag.String(
		"n",
		"GC",
		"name",
	)
)

func main() {
	flag.Parse()
	if *input1 == "" || *input2 == "" {
		flag.PrintDefaults()
		log.Fatal("-1/-2 required!")
	}

	// 创建一个新的图表
	p := plot.New()
	p.Title.Text = "Line Plot with Points"
	p.X.Label.Text = "index"
	p.Y.Label.Text = "GC"

	// 数据点
	points1 := plotter.XYs{}
	points2 := plotter.XYs{}

	var bytes1 = []byte(*input1)
	for i := 0; i < len(bytes1); i++ {
		if i+*window > len(bytes1) {
			break
		}
		var gc = calGC(bytes1[i : i+*window])
		points1 = append(points1, plotter.XY{X: float64(i + 1), Y: gc})
	}

	var bytes2 = []byte(*input2)
	for i := 0; i < len(bytes2); i++ {
		if i+*window > len(bytes2) {
			break
		}
		var gc = calGC(bytes2[i : i+*window])
		points2 = append(points2, plotter.XY{X: float64(i + 1), Y: gc})
	}

	// 创建一个线图和一个散点图
	line1, pointsPlot1, err := plotter.NewLinePoints(points1)
	if err != nil {
		panic(err)
	}

	// 设置线条和点的样式
	line1.Color = plotutil.Color(1)
	line1.Width = vg.Points(6)
	pointsPlot1.Shape = plotutil.Shape(1)

	// 创建一个线图和一个散点图
	line2, pointsPlot2, err := plotter.NewLinePoints(points2)
	if err != nil {
		panic(err)
	}

	// 设置线条和点的样式
	line2.Color = plotutil.Color(2)
	line2.Width = vg.Points(6)
	pointsPlot2.Shape = plotutil.Shape(2)

	// 添加线图和散点图到图表
	p.Add(line1, line2)
	// 添加图例
	// p.Legend.Add("99.5%", line1, pointsPlot1)
	// p.Legend.Add("94.6%", line2, pointsPlot2)
	p.Legend.Add("99.5%", line1)
	p.Legend.Add("94.6%", line2)

	// 保存图表到文件
	if err := p.Save(16*vg.Inch, 9*vg.Inch, *name+".line_with_points.png"); err != nil {
		panic(err)
	}
}

func calGC(byteS []byte) float64 {
	var count = 0
	for _, c := range byteS {
		if c == 'G' || c == 'C' {
			count++
		}
	}
	return float64(count) / float64(len(byteS))
}
