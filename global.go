package main

import "regexp"

// regexp
var (
	plus3    = regexp.MustCompile(`\+\+\+`)
	minus1   = regexp.MustCompile(`-`)
	minus2   = regexp.MustCompile(`--`)
	regPolyA = regexp.MustCompile(`AAAAAAAA`)
	regN     = regexp.MustCompile(`N`)
	regA     = regexp.MustCompile(`A`)
	regC     = regexp.MustCompile(`C`)
	regG     = regexp.MustCompile(`G`)
	regT     = regexp.MustCompile(`T`)
)
