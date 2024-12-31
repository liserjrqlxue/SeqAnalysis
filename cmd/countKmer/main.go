package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/xuri/excelize/v2"
)

var set = make(map[string]string)

func main() {
	flag.Parse()
	// if *input == "" || *output == "" {
	// 	flag.PrintDefaults()
	// 	log.Fatal("-i/-o required!")
	// }

	var xlsx, err = excelize.OpenFile("AB库.xlsx")
	simpleUtil.CheckErr(err)
	rows, err := xlsx.GetRows("Sheet1")
	simpleUtil.CheckErr(err)

	log.Printf("rows:%d\tcols:%d\n", len(rows), len(rows[0]))
	if rows[0][0] != "A库" || rows[0][2] != "B库" {
		log.Fatal("AB库.xlsx 格式错误")
	}
	var count = [2]int{0, 0}
	for i := 1; i < len(rows); i++ {
		seqA := rows[i][0]
		seqB := rows[i][2]
		if seqA != "" {
			set[strings.Replace(seqA, "->", "", 1)] = "A库"
			count[0]++
		}
		if seqB != "" {
			set[strings.Replace(seqB, "->", "", 1)] = "B库"
			count[1]++
		}
	}
	log.Printf("A库:%d\tB库:%d\n", count[0], count[1])
	fmt.Println("请输入序列：,按ctr+c退出")

	var scanner = bufio.NewScanner(os.Stdin)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		countStr(line)
	}
	err = scanner.Err()
	if err != nil && err != io.EOF {
		log.Fatalf("load with error:[%v]", err)
	}
}

func countStr(str string) {
	var c = make(map[string]int)
	n := len(str)
	for i := 5; i <= n; i++ {
		key := str[i-5 : i]
		// log.Printf("%d：[%s]\n", i, key)
		c[set[key]]++
	}
	for k, v := range c {
		fmt.Printf("%s:\t%d\n", k, v)
	}
}
