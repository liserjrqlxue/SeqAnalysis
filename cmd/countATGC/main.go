package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"

	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
)

var (
	input = flag.String(
		"i",
		"",
		"input seq.txt",
	)
)

var Count [120][5]float64

func main() {
	flag.Parse()
	if *input == "" {
		flag.PrintDefaults()
		log.Fatal("-i required!")
	}

	var in = osUtil.Open(*input)
	defer simpleUtil.DeferClose(in)
	var scanner = bufio.NewScanner(in)
	for scanner.Scan() {
		bs := scanner.Bytes()
		for i := 0; i < len(bs); i++ {
			switch bs[i] {
			case 'A':
				Count[i][1]++
			case 'C':
				Count[i][2]++
			case 'G':
				Count[i][3]++
			case 'T':
				Count[i][4]++
			}
			Count[i][0]++
		}
	}
	err := scanner.Err()
	if err != nil {
		log.Fatalf("load with error:[%v]", err)
	}
	fmt.Print("POS\tA\tC\tG\tT\tGC\tAT\n")
	for i := 0; i < 120; i++ {
		fmt.Printf(
			"%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\n",
			i+1,
			100*Count[i][1]/Count[i][0],
			100*Count[i][2]/Count[i][0],
			100*Count[i][3]/Count[i][0],
			100*Count[i][4]/Count[i][0],
			100*(Count[i][2]+Count[i][3])/Count[i][0],
			100*(Count[i][1]+Count[i][4])/Count[i][0],
		)
	}
}
