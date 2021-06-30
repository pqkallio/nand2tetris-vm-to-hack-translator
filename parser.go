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

type funcCallArgs struct {
	name string
	nArgs string
}

type pushPopArgs struct {
	op string
	mem mem
}

type labelArgs struct {
	name string
}

type arithmeticLogicalArgs struct {
	op string
}

type gotoArgs struct {
	label string
}

func NewParser(in *os.File) *parser {
	scanner := bufio.NewScanner(in)

	return &parser{scanner, -1}
}

const (
	ArithmeticLogical = iota
	PushPop
	Func
	Call
	Return
	Label
	Goto
	IfGoto
)

type mem struct {
	seg string
	offset string
}

type vmRow struct {
	command string
	opType int
	rowIdx int
	args interface{}
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
			switch split[0] {
			case "return":
				row.opType = Return
			default:
				row.opType = ArithmeticLogical
				row.args = arithmeticLogicalArgs{op: split[0]}
			}
		case 2:
			switch split[0] {
			case "label":
				row.opType = Label
				row.args = labelArgs{name: split[1]}
			case "goto":
				row.opType = Goto
				row.args = gotoArgs{label: split[1]}
			case "if-goto":
				row.opType = IfGoto
				row.args = gotoArgs{label: split[1]}
			}
		case 3:
			switch split[0] {
			case "function":
				row.opType = Func
				row.args = funcCallArgs{
					name:       split[1],
					nArgs: split[2],
				}
			case "call":
				row.opType = Call
				row.args = funcCallArgs{
					name: split[1],
					nArgs:    split[2],
				}
			case "push":
				fallthrough
			case "pop":
				row.opType = PushPop
				row.args = pushPopArgs{
					op:  split[0],
					mem: mem{
						seg:    split[1],
						offset: split[2],
					},
				}
			}
		}

		return &row
	}

	return nil
}