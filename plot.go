package main

import "github.com/go-echarts/go-echarts/v2/opts"

func generateLineItems(vs []int) []opts.LineData {
	var items = make([]opts.LineData, 0)
	for _, v := range vs {
		items = append(items, opts.LineData{Value: v})
	}
	return items
}
