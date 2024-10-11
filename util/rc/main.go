package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	// "compress/gzip"
	gzip "github.com/klauspost/pgzip"

	util "SeqAnalysis/pkg/seqAnalysis"

	"github.com/liserjrqlxue/goUtil/osUtil"
)

type File struct {
	path string
	file *os.File
	gzr  *gzip.Reader
}

var isGz = regexp.MustCompile(`\.gz$`)

func main() {
	var (
		fileList   []string
		stringList []string
		files      []*File
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
			var file = &File{
				path: v,
				file: osUtil.Open(v),
			}
			if isGz.MatchString(v) {
				file.gzr, _ = gzip.NewReader(file.file)
			}
			files = append(files, file)
		}
	} else if len(stringList) != 0 {
		for _, v := range stringList {
			fmt.Println(util.ReverseComplement(v))
		}
		return
	}
	if len(files) == 0 {
		fileList = append(fileList, "STDIN")
		files = append(files, &File{path: "STDIN", file: os.Stdin})
	}
	for i, file := range files {
		var v io.ReadCloser
		v = file.file
		if file.gzr != nil {
			v = file.gzr
		}
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
		if file.gzr != nil {
			err = file.gzr.Close()
			if err != nil {
				log.Fatalf("file:[%s] gz close failed:[%v]", fileList[i], err)
			}
		}
		err = file.file.Close()
		if err != nil {
			log.Fatalf("file:[%s] close failed:[%v]", fileList[i], err)
		}
	}
}
