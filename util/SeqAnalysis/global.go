package main

import (
	"regexp"

	"github.com/liserjrqlxue/goUtil/osUtil"
)

// regexp
var (
	plus3  = regexp.MustCompile(`\+\+\+`)
	minus1 = regexp.MustCompile(`-`)
	minus2 = regexp.MustCompile(`--`)
	minus3 = regexp.MustCompile(`---`)

	//minus3   = regexp.MustCompile(`---`)
	//m2p2     = regexp.MustCompile(`\+\+--`)

	regN = regexp.MustCompile(`N`)
	regA = regexp.MustCompile(`A`)
	regC = regexp.MustCompile(`C`)
	regG = regexp.MustCompile(`G`)
	regT = regexp.MustCompile(`T`)

	gz     = regexp.MustCompile(`\.gz$`)
	isXlsx = regexp.MustCompile(`\.xlsx$`)
)

var (
	Sheets           = make(map[string]string)
	sheetList        []string
	chanList         chan bool
	SeqInfoMap       = make(map[string]*SeqInfo)
	ParallelStatsMap = make(map[string]*ParallelTest)
)

var (
	TitleTar            = osUtil.FS2Array(osUtil.OpenFS("etc/title.Tar.txt", exPath, etcEMFS))
	TitleStats          = osUtil.FS2Array(osUtil.OpenFS("etc/title.Stats.txt", exPath, etcEMFS))
	TitleSummary        = osUtil.FS2Array(osUtil.OpenFS("etc/title.Summary.txt", exPath, etcEMFS))
	StatisticalField, _ = osUtil.FS2MapArray(osUtil.OpenFS("etc/统计字段.txt", exPath, etcEMFS), "\t", nil)
)
