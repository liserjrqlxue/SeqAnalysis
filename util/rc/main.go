package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"SeqAnalysis/util"

	"github.com/liserjrqlxue/goUtil/osUtil"
)

func main() {
	var (
		fileList   []string
		stringList []string
		files      []*os.File
	)
	for i, v := range os.Args {
		if i == 0 {
			continue
		}
		log.Printf("args:[%s]", v)
		if osUtil.FileExists(v) {
			fileList = append(fileList, v)
		} else {
			stringList = append(stringList, v)
		}
	}
	if len(fileList) != 0 {
		for _, v := range stringList {
			log.Printf("file:[%s] not exists!", v)
		}
		for _, v := range fileList {
			files = append(files, osUtil.Create(v))
		}
	} else if len(stringList) != 0 {
		for _, v := range stringList {
			fmt.Println(util.ReverseComplement(v))
		}
		return
	}
	if len(files) == 0 {
		fileList = append(fileList, "STDIN")
		files = append(files, os.Stdin)
	}
	for i, v := range files {
		var (
			reader = bufio.NewReader(v)
			line   string
			err    error
		)
		for {
			line, err = reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Println(strings.TrimPrefix(util.ReverseComplement(line), "\n"))
		}
		if err != io.EOF {
			log.Fatalf("file:[%s] load with error:[%v]", fileList[i], err)
		}
		err = v.Close()
		if err != nil {
			log.Fatalf("file:[%s] close failed:[%v]", fileList[i], err)
		}
	}
}
