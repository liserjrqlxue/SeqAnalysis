package main

import "regexp"

// regexp
var (
	plus3    = regexp.MustCompile(`\+\+\+`)
	minus1   = regexp.MustCompile(`-`)
	minus2   = regexp.MustCompile(`--`)
	regPolyA = regexp.MustCompile(`AAAAAAAA`)

	//minus3   = regexp.MustCompile(`---`)
	//m2p2     = regexp.MustCompile(`\+\+--`)

	regN = regexp.MustCompile(`N`)
	regA = regexp.MustCompile(`A`)
	regC = regexp.MustCompile(`C`)
	regG = regexp.MustCompile(`G`)
	regT = regexp.MustCompile(`T`)

	gz = regexp.MustCompile(`gz$`)
)

var (
	Sheets    = make(map[string]string)
	sheetList []string
	chanList  chan bool
)
