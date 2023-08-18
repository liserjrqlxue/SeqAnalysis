package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/liserjrqlxue/goUtil/osUtil"
	"github.com/liserjrqlxue/goUtil/simpleUtil"
	"github.com/xuri/excelize/v2"
)

func ExcelToSlice(filename, sheetName string) ([][]string, error) {
	// Open the Excel file
	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer simpleUtil.DeferClose(file)

	// Get all the rows from the specified sheet
	return file.GetRows(sheetName)
}

func slice2MapArray(s [][]string) (data []map[string]string) {
	var key = s[0]
	for i := 1; i < len(s); i++ {
		var item = make(map[string]string)
		for j := 0; j < len(s[i]); j++ {
			item[key[j]] = s[i][j]
		}
		data = append(data, item)
	}
	return
}

func RunPrimerDesigner(workdir, name, seq string) error {
	workdir = filepath.Join(workdir, name)
	seqFile := filepath.Join(workdir, "seq.txt")
	prefix := filepath.Join(workdir, name)

	if err := os.MkdirAll(workdir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(seqFile, []byte(seq), 0644); err != nil {
		return err
	}

	cmd := exec.Command("util/primerDesigner/primerDesigner.exe", "-i", seqFile, "-o", prefix, "-n", name)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Println(cmd)

	return cmd.Run()
}

func writeBatch(prefix string, list [][3]string) error {
	var count = 0
	var panel = 0
	var fList []*os.File
	var TailChangingPrimers = osUtil.Create(prefix + "换尾引物.txt")
	defer simpleUtil.DeferClose(TailChangingPrimers)

	_, err := fmt.Fprintf(TailChangingPrimers, "%s\t%s\t%s\n", "引物名称", "引物序列5-3", "基因名")
	if err != nil {
		return err
	}

	excel, err := excelize.OpenFile(prefix + "J-" + list[0][0] + ".xlsx")
	if err != nil {
		return err
	}

	for _, seq := range list {
		id := seq[0]
		lines, err := ReadFileToLineArray(seq[1])
		if err != nil {
			return err
		}

		for _, line := range lines {
			if count%96 == 0 {
				f, err := os.Create(prefix + string(rune('A'+panel)) + "-" + id + ".seq")
				if err != nil {
					return err
				}
				panel++
				fList = append(fList, f)
			}
			_, err = fList[panel-1].WriteString(line + "\n")
			if err != nil {
				return err
			}
			var cells = strings.Split(line, ",")
			// Set the value in the specified cell
			if cells[0] != "covering" {
				err = excel.SetCellStr("引物订购单", "D"+strconv.Itoa(17+count), cells[0])
				if err != nil {
					return err
				}
				err = excel.SetCellStr("引物订购单", "E"+strconv.Itoa(17+count), cells[1])
				if err != nil {
					return err
				}
			}
			count++
		}

		lines, err = ReadFileToLineArray(seq[2])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(TailChangingPrimers, strings.Join(lines, "\n"))
		if err != nil {
			return err
		}
	}
	for _, f := range fList {
		err := f.Close()
		if err != nil {
			return err
		}
	}

	err = excel.UpdateLinkedValue()
	if err != nil {
		return err
	}

	// Save the changes back to the file
	err = excel.Save()
	if err != nil {
		return err
	}
	return nil
}

func ReadFileToLineArray(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func CopyFile(source string, destination string) error {
	// Open the source file
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Flush any buffered data to ensure the file is fully written
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
