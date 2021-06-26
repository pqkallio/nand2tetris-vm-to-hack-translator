package main

var vmMemPtrToAsmMemPtr = map[string]string{
	"local": "LCL",
	"argument": "ARG",
	"this": "THIS",
	"that": "THAT",
}