package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type translator struct {
	parser *parser
	writer *bufio.Writer
	filename string
}

func NewTranslator(p *parser, out *os.File, filename string) *translator {
	writer := bufio.NewWriter(out)

	return &translator{p, writer, filename}
}

func (t *translator) translate() error {
	row := t.parser.parseNext()

	for row != nil {
		var translated []string

		switch row.op {
		case "push":
			translated = t.push(row)
		case "pop":
			translated = t.pop(row)
		default:
			translated = t.arithmeticLogical(row)
		}

		output := append(
			[]string{
				"// " + row.command,
			},
			translated...,
		)

		if _, err := t.writer.Write([]byte(strings.Join(output, "\n") + "\n")); err != nil {
			return err
		}

		if err := t.writer.Flush(); err != nil {
			return err
		}

		row = t.parser.parseNext()
	}

	return nil
}

func (t *translator) pop(row *vmRow) []string {
	switch row.mem.seg {
	case "argument":
	case "local":
	case "this":
	case "that":
		return popToMemSeg(row)
	case "static":
		return t.popStatic()(row)
	case "temp":
		return popTemp(row)
	case "pointer":
		return popPointer(row)
	}

	return []string{}
}

func popPointer(row *vmRow) []string {
	addr := "@THIS"

	if row.mem.offset != "0" {
		addr = "@THAT"
	}

	return append(
		popFromStack,
		addr,
		"M=D",
	)
}

func popTemp(row *vmRow) []string {
	return append(
		append(
			[]string{
				"@5",
				"D=A",
				"@" + row.mem.offset,
				"A=D+A",
				"D=A",
				"@R13",
				"M=D",
			},
			popFromStack...,
		),
		setData...,
	)
}

func popToMemSeg(row *vmRow) []string {
	return append(
		append(
			popAddr(&row.mem),
			popFromStack...,
		),
		setData...,
	)
}

func popAddr(m *mem) []string {
	return []string{
		"@" + m.offset,
		"D=A",
		"@" + vmMemPtrToAsmMemPtr[m.seg],
		"A=M",
		"A=D+A",
		"D=A",
		"@R13",
		"M=D",
	}
}

func (t *translator) push(row *vmRow) []string {
	switch row.mem.seg {
	case "argument":
	case "local":
	case "this":
	case "that":
		return pushFromMemSeg(row)
	case "constant":
		return pushConst(row)
	case "static":
		return t.pushStatic()(row)
	case "temp":
		return pushTemp(row)
	case "pointer":
		return pushPointer(row)
	}

	return []string{}
}

type translatorFunc func(row *vmRow) []string

func pushPointer(row *vmRow) []string {
	addr := "@THIS"

	if row.mem.offset != "0" {
		addr = "@THAT"
	}

	return append(
		[]string{
			addr,
			"D=A",
		},
		pushToStack...,
	)
}

func pushTemp(row *vmRow) []string {
	return append(
		[]string{
			"@5",
			"D=A",
			"@" + row.mem.offset,
			"A=D+A",
			"D=M",
		},
		pushToStack...,
	)
}

func pushConst(row *vmRow) []string {
	return append(
		[]string {
			"@" + row.mem.offset,
			"D=A",
		},
		pushToStack...,
	)
}

func pushFromMemSeg(row *vmRow) []string {
	return append(fromMem(&row.mem), pushToStack...)
}

func (t *translator) pushStatic() translatorFunc {
	return func(row *vmRow) []string {
		return append(
			[]string{
				"@" + t.filename + "." + row.mem.offset,
				"D=M",
			},
			pushToStack...,
		)
	}
}

func (t *translator) popStatic() translatorFunc {
	return func(row *vmRow) []string {
		return append(
			popFromStack,
			"@" + t.filename + "." + row.mem.offset,
			"M=D",
		)
	}
}

func (t *translator) arithmeticLogical(row *vmRow) []string {
	switch row.op {
	case "add":
		return binOp("D=D+M")
	case "sub":
		return binOp("D=M-D")
	case "eq":
		fallthrough
	case "gt":
		fallthrough
	case "lt":
		return binOp(t.logical(row)...)
	case "neg":
		return unOp("D=-M")
	case "and":
		return binOp("D=D&M")
	case "or":
		return binOp("D=D|M")
	case "not":
		return unOp("D=!M")

	}

	return []string{}
}

func (t *translator) logical(row *vmRow) []string {
	lblPrefix := t.filename + "." + strconv.Itoa(row.rowIdx)
	trueLbl := lblPrefix + "." + "TRUE"
	endLbl := lblPrefix + "." + "END"

	return []string{
		"D=M-D",
		"@" + trueLbl,
		"D;" + jumps[row.op],
		"@" + endLbl,
		"D=0;JMP",
		"(" + trueLbl + ")",
		"D=-1",
		"(" + endLbl + ")",
		"@SP",
		"A=M",
	}
}

func unOp(op ...string) []string {
	return append(
		append(
			secondOrUnaryOperand,
			op...,
		),
		postOper...,
	)
}

func binOp(op ...string) []string {
	return append(
		append(
			append(
				[]string{
					"@SP",
					"M=M-1",
					"A=M",
					"D=M",
				},
				secondOrUnaryOperand...,
			),
			op...,
		),
		postOper...,
	)
}

var postOper = []string{
	"M=D",
	"@SP",
	"M=M+1",
}

var secondOrUnaryOperand = []string{
	"@SP",
	"M=M-1",
	"A=M",
}

var pushToStack = []string{
	"@SP",
	"A=M",
	"M=D",
	"@SP",
	"M=M+1",
}

var popFromStack = []string{
	"@SP",
	"M=M-1",
	"A=M",
	"D=M",
}

var setData = []string{
	"@R13",
	"A=M",
	"M=D",
}

func fromMem(mem *mem) []string {
	return []string{
		"@" + mem.offset,
		"D=A",
		"@" + vmMemPtrToAsmMemPtr[mem.seg],
		"A=M",
		"A=D+A",
		"D=M",
	}
}