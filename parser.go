package main

import (
	"bufio"
	"os"
	"strings"
)

type parser struct {
	scanner *bufio.Scanner
	rowIdx int
}

func NewParser(in *os.File) *parser {
	scanner := bufio.NewScanner(in)

	return &parser{scanner, -1}
}

const (
	ArithmeticLogical = iota
	PushPop
)

type mem struct {
	seg string
	offset string
}

type vmRow struct {
	command string
	opType int
	op string
	mem mem
	rowIdx int
}

func (p *parser) nextRowIdx() int {
	p.rowIdx++

	return p.rowIdx
}

func (p *parser) parseNext() *vmRow {
	for p.scanner.Scan() {
		var row vmRow
		command := strings.TrimSpace(p.scanner.Text())

		if len(command) == 0 || strings.HasPrefix(command, "//") {
			continue
		}

		row.command = command
		row.rowIdx = p.nextRowIdx()

		split := strings.Split(command, " ")

		switch len(split) {
		case 1:
			row.opType = ArithmeticLogical
			row.op = split[0]
		case 3:
			row.opType = PushPop
			row.op = split[0]
			row.mem = mem{
				seg:    split[1],
				offset: split[2],
			}

		}

		return &row
	}

	return nil
}