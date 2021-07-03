package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type translator struct {
	writer *bufio.Writer
	data   pathData
	ctx    *context
}

func NewTranslator(out *os.File, data pathData) *translator {
	writer := bufio.NewWriter(out)

	return &translator{writer, data, nil}
}

func (t *translator) translateFile(f *os.File) error {
	p := NewParser(f)

	row := p.parseNext()

	for row != nil {
		var translated []string

		switch row.opType {
		case PushPop:
			args := row.args.(pushPopArgs)

			switch args.op {
			case "push":
				translated = t.push(&args)
			case "pop":
				translated = t.pop(&args)
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

		row = p.parseNext()
	}

	return nil

}

func (t *translator) write(output []string) error {
	if _, err := t.writer.Write([]byte(strings.Join(output, "\n") + "\n")); err != nil {
		return err
	}

	if err := t.writer.Flush(); err != nil {
		return err
	}

	return nil
}

func (t *translator) writeBootSector() error {
	return t.write(MultiAppend(
		[]string{
			"@256",
			"D=A",
			"@SP",
			"M=D",
		},
		pushAddrToStack("LCL"),
		pushAddrToStack("LCL"),
		pushAddrToStack("ARG"),
		pushAddrToStack("THIS"),
		pushAddrToStack("THAT"),
		[]string{
			"@Sys.init",
			"0;JMP",
		},
	))
}

func (t *translator) translate() error {
	if t.data.pathType == dir {
		if err := t.writeBootSector(); err != nil {
			return err
		}
	}

	for _, f := range t.data.files {
		ff, err := os.Open(f.fullPath)
		if err != nil {
			log.Fatalf("Cannot open file %s", f.fullPath)
		}

		err = t.translateFile(ff)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *translator) pop(args *pushPopArgs) []string {
	switch args.mem.seg {
	case "argument":
		fallthrough
	case "local":
		fallthrough
	case "this":
		fallthrough
	case "that":
		return popToMemSeg(&args.mem)
	case "static":
		return popStatic(args)
	case "temp":
		return popTemp(&args.mem)
	case "pointer":
		return popPointer(&args.mem)
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

func (t *translator) push(args *pushPopArgs) []string {
	switch args.mem.seg {
	case "argument":
		fallthrough
	case "local":
		fallthrough
	case "this":
		fallthrough
	case "that":
		return pushFromMemSeg(&args.mem)
	case "constant":
		return pushConst(&args.mem)
	case "static":
		return pushStatic(args)
	case "temp":
		return pushTemp(&args.mem)
	case "pointer":
		return pushPointer(&args.mem)
	}

	return []string{}
}

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
		[]string{
			"@" + mem.offset,
			"D=A",
		},
		pushToStack...,
	)
}

func pushFromMemSeg(mem *mem) []string {
	return append(fromMem(mem), pushToStack...)
}

func pushStatic(args *pushPopArgs) []string {
	return append(
		[]string{
			"@" + args.fileName + "." + args.mem.offset,
			"D=M",
		},
		pushToStack...,
	)
}

func popStatic(args *pushPopArgs) []string {
	return append(
		popFromStack,
		"@"+args.fileName+"."+args.mem.offset,
		"M=D",
	)
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

	lblPrefix := args.op + "." + strconv.Itoa(row.rowIdx)
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

func (t *translator) Func(row *vmRow) []string {
	args := row.args.(funcCallArgs)
	t.ctx = newContext(args.name)

	loopLabel := args.name + ".initLCL"
	loopEndLabel := loopLabel + ".end"

	return []string{
		"(" + args.name + ")",
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
		"@" + loopLabel,
		"0;JMP",
		"(" + loopEndLabel + ")",
	}
}

func (t *translator) Call(row *vmRow) []string {
	args := row.args.(funcCallArgs)
	retAddr := fmt.Sprintf("%s$ret.%d", t.ctx.funcName, t.ctx.nextIdx())
	funcName := args.name

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

	label := args.name
	if t.ctx != nil {
		label = t.ctx.funcName + "$" + label
	}

	return []string{
		"(" + label + ")",
	}
}

func (t *translator) Goto(row *vmRow) []string {
	args := row.args.(gotoArgs)

	label := args.label
	if t.ctx != nil {
		label = t.ctx.funcName + "$" + label
	}

	return []string{
		"@" + label,
		"0;JMP",
	}
}

func (t *translator) IfGoto(row *vmRow) []string {
	args := row.args.(gotoArgs)

	label := args.label
	if t.ctx != nil {
		label = t.ctx.funcName + "$" + label
	}

	return []string{
		"@SP",
		"M=M-1",
		"A=M",
		"D=M",
		"@R13",
		"M=-1",
		"D=D-M",
		"@" + label,
		"D;JEQ",
	}
}

func pushAddrToStack(addr string) []string {
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

type context struct {
	funcName string
	idx      int
}

func newContext(funcName string) *context {
	return &context{funcName, -1}
}

func (c *context) nextIdx() int {
	c.idx++

	return c.idx
}
