package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatalln("Supply only one argument, the name of the Jack VM (.vm) file to be translated.")
	}

	vmFileName := args[0]

	vmFile, err := os.Open(vmFileName)
	if err != nil {
		log.Fatalf("Unable to open %s: %w", vmFileName, err)
	}

	defer vmFile.Close()

	vms := strings.SplitN(Reverse(vmFileName), ".", 2)
	asmFileName := Reverse(vms[len(vms) - 1]) + ".asm"
	vms = strings.SplitN(vms[len(vms) - 1], "/", 2)
	fn := Reverse(vms[0])

	asmFile, err := os.Create(asmFileName)
	if err != nil {
		log.Fatalf("Unable to create %s: %w", asmFileName, err)
	}

	defer asmFile.Close()

	p := NewParser(vmFile)
	t := NewTranslator(p, asmFile, fn)

	t.translate()
}
