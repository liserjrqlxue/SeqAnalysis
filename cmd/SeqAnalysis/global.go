package main

import (
	"regexp"

	util "SeqAnalysis/pkg/seqAnalysis"
)

// regexp
var (
	isXlsx = regexp.MustCompile(`\.xlsx$`)
)

var (
	Sheets           = make(map[string]string)
	sheetList        []string
	chanList         chan bool
	SeqInfoMap       = make(map[string]*util.SeqInfo)
	ParallelStatsMap = make(map[string]*util.ParallelTest)
)

var (
	TitleTar         []string
	TitleStats       []string
	TitleSummary     []string
	StatisticalField []map[string]string
)
