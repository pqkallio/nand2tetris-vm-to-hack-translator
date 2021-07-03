package main

var vmMemPtrToAsmMemPtr = map[string]string{
	"local":    "LCL",
	"argument": "ARG",
	"this":     "THIS",
	"that":     "THAT",
}

var jumps = map[string]string{
	"eq": "JEQ",
	"lt": "JLT",
	"gt": "JGT",
}
