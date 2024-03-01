package main

import "regexp"

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
