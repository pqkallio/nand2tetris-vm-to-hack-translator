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

		switch row.opType {
		case PushPop:
			args := row.args.(pushPopArgs)

			switch args.op {
			case "push":
				translated = t.push(&args.mem)
			case "pop":
				translated = t.pop(&args.mem)
			}
		case ArithmeticLogical:
			translated = t.arithmeticLogical(row)
		case Func:
			translated = t.Func(row)
		case Call:
			translated = t.Call(row)
		case Return:
			translated = t.Return(row)
		case Label:
			translated = t.Label(row)
		case Goto:
			translated = t.Goto(row)
		case IfGoto:
			translated = t.IfGoto(row)
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

func (t *translator) pop(mem *mem) []string {
	switch mem.seg {
	case "argument":
		fallthrough
	case "local":
		fallthrough
	case "this":
		fallthrough
	case "that":
		return popToMemSeg(mem)
	case "static":
		return t.popStatic()(mem)
	case "temp":
		return popTemp(mem)
	case "pointer":
		return popPointer(mem)
	}

	return []string{}
}

func popPointer(mem *mem) []string {
	addr := "@THIS"

	if mem.offset != "0" {
		addr = "@THAT"
	}

	return append(
		popFromStack,
		addr,
		"M=D",
	)
}

func popTemp(mem *mem) []string {
	return MultiAppend(
		[]string{
			"@5",
			"D=A",
			"@" + mem.offset,
			"A=D+A",
			"D=A",
			"@R13",
			"M=D",
		},
		popFromStack,
		setData,
	)
}

func popToMemSeg(mem *mem) []string {
	return MultiAppend(
		popAddr(mem),
		popFromStack,
		setData,
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

func (t *translator) push(mem *mem) []string {
	switch mem.seg {
	case "argument":
		fallthrough
	case "local":
		fallthrough
	case "this":
		fallthrough
	case "that":
		return pushFromMemSeg(mem)
	case "constant":
		return pushConst(mem)
	case "static":
		return t.pushStatic()(mem)
	case "temp":
		return pushTemp(mem)
	case "pointer":
		return pushPointer(mem)
	}

	return []string{}
}

type translatorFunc func(mem *mem) []string

func pushPointer(mem *mem) []string {
	addr := "@THIS"

	if mem.offset != "0" {
		addr = "@THAT"
	}

	return append(
		[]string{
			addr,
			"D=M",
		},
		pushToStack...,
	)
}

func pushTemp(mem *mem) []string {
	return append(
		[]string{
			"@5",
			"D=A",
			"@" + mem.offset,
			"A=D+A",
			"D=M",
		},
		pushToStack...,
	)
}

func pushConst(mem *mem) []string {
	return append(
		[]string {
			"@" + mem.offset,
			"D=A",
		},
		pushToStack...,
	)
}

func pushFromMemSeg(mem *mem) []string {
	return append(fromMem(mem), pushToStack...)
}

func (t *translator) pushStatic() translatorFunc {
	return func(mem *mem) []string {
		return append(
			[]string{
				"@" + t.filename + "." + mem.offset,
				"D=M",
			},
			pushToStack...,
		)
	}
}

func (t *translator) popStatic() translatorFunc {
	return func(mem *mem) []string {
		return append(
			popFromStack,
			"@" + t.filename + "." + mem.offset,
			"M=D",
		)
	}
}

func (t *translator) arithmeticLogical(row *vmRow) []string {
	args := row.args.(arithmeticLogicalArgs)

	switch args.op {
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
	args := row.args.(arithmeticLogicalArgs)

	lblPrefix := t.filename + "." + strconv.Itoa(row.rowIdx)
	trueLbl := lblPrefix + "." + "TRUE"
	endLbl := lblPrefix + "." + "END"

	return []string{
		"D=M-D",
		"@" + trueLbl,
		"D;" + jumps[args.op],
		"@" + endLbl,
		"D=0;JMP",
		"(" + trueLbl + ")",
		"D=-1",
		"(" + endLbl + ")",
		"@SP",
		"A=M",
	}
}

func (t *translator) labelForFunc(args *funcCallArgs) string {
	return t.filename + "." + args.name
}

func (t *translator) Func(row *vmRow) []string {
	args := row.args.(funcCallArgs)

	loopLabel := t.labelForFunc(&args) + ".initLCL"
	loopEndLabel := loopLabel + ".end"

	return []string{
		"(" + t.labelForFunc(&args) + ")",
		"@" + args.nArgs,
		"D=A",
		"@R13",
		"M=D",
		"(" + loopLabel + ")",
		"@R13",
		"D=M",
		"@" + loopEndLabel,
		"D;JEQ",
		"@SP",
		"A=M",
		"M=0",
		"@SP",
		"M=M+1",
		"@R13",
		"M=M-1",
		"@" + t.labelForFunc(&args),
		"0;JMP",
		"(" + loopEndLabel + ")",
	}
}

func (t *translator) Call(row *vmRow) []string {
	args := row.args.(funcCallArgs)
	retAddr := t.filename + "." + strconv.Itoa(row.rowIdx) + "retAddr"
	funcName := t.filename + "." + args.name

	return MultiAppend(
		[]string{
			"@" + retAddr,
			"D=A",
			"@SP",
			"A=M",
			"M=D",
			"@SP",
			"M=M+1",
		},
		pushAddrToStack("LCL"),
		pushAddrToStack("ARG"),
		pushAddrToStack("THIS"),
		pushAddrToStack("THAT"),
		[]string{
			"@SP",
			"D=M",
			"@5",
			"D=D-A",
			"@" + args.nArgs,
			"D=D-A",
			"@ARG",
			"M=D",
			"@SP",
			"D=M",
			"@LCL",
			"M=D",
			"@" + funcName,
			"0;JMP",
			"(" + retAddr + ")",
		},
	)
}

func (t *translator) Return(_ *vmRow) []string {
	return []string{
		"@LCL",
		"D=M",
		"@R13",
		"M=D",
		"@5",
		"A=D-A",
		"D=M",
		"@R14",
		"M=D",
		"@SP",
		"M=M-1",
		"A=M",
		"D=M",
		"@ARG",
		"A=M",
		"M=D",
		"@ARG",
		"A=M",
		"D=A+1",
		"@SP",
		"M=D",
		"@R13",
		"D=M",
		"@1",
		"A=D-A",
		"D=M",
		"@THAT",
		"M=D",
		"@R13",
		"D=M",
		"@2",
		"A=D-A",
		"D=M",
		"@THIS",
		"M=D",
		"@R13",
		"D=M",
		"@3",
		"A=D-A",
		"D=M",
		"@ARG",
		"M=D",
		"@R13",
		"D=M",
		"@4",
		"A=D-A",
		"D=M",
		"@LCL",
		"M=D",
		"@R14",
		"A=M",
		"0;JMP",
	}
}

func (t *translator) Label(row *vmRow) []string {
	args := row.args.(labelArgs)

	return []string{
		"(" + t.filename + "." + args.name + ")",
	}
}

func (t *translator) Goto(row *vmRow) []string {
	args := row.args.(gotoArgs)

	return []string{
		"@" + args.label,
		"0;JMP",
	}
}

func (t *translator) IfGoto(row *vmRow) []string {
	args := row.args.(gotoArgs)

	return []string{
		"@SP",
		"M=M-1",
		"A=M",
		"D=M",
		"@R13",
		"M=-1",
		"D=D-M",
		"@" + args.label,
		"D;JEQ",
	}
}

func pushAddrToStack(addr string) []string{
	return []string{
		"@" + addr,
		"D=M",
		"@SP",
		"A=M",
		"M=D",
		"@SP",
		"M=M+1",
	}
}

func unOp(op ...string) []string {
	return MultiAppend(
		secondOrUnaryOperand,
		op,
		postOper,
	)
}

func binOp(op ...string) []string {
	return MultiAppend(
		[]string{
			"@SP",
			"M=M-1",
			"A=M",
			"D=M",
		},
		secondOrUnaryOperand,
		op,
		postOper,
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