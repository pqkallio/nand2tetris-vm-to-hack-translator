package main

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	file = iota
	dir
)

type fileInfo struct {
	fullPath string
	file     fs.FileInfo
}

type pathData struct {
	name     string
	pathType int
	files    []fileInfo
}

func openDir(filePath string) (pathData, error) {
	split := strings.Split(filePath, "/")
	fullPath := path.Join(filePath, split[len(split)-1])
	data := pathData{name: fullPath, pathType: dir}

	files := []fileInfo{}

	err := filepath.Walk(filePath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && path != filePath {
			return filepath.SkipDir
		}

		if strings.HasSuffix(info.Name(), ".vm") {
			log.Printf("%s selected for translation", info.Name())
			files = append(files, fileInfo{path, info})
		}

		return nil
	})

	if err != nil {
		return data, err
	}

	data.files = files

	return data, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var data pathData

	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatalln("Supply only one argument, the name of the Jack VM (.vm) file to be translated.")
	}

	vmFilePath := args[0]

	stat, err := os.Stat(vmFilePath)
	if err != nil {
		log.Fatalf("Unable to check path %s", vmFilePath)
	}

	switch mode := stat.Mode(); {
	case mode.IsDir():
		data, err = openDir(vmFilePath)
		if err != nil {
			log.Fatalf("Unable to read directory %s", vmFilePath)
		}
	case mode.IsRegular():
		split := strings.Split(vmFilePath, ".vm")
		data.name = split[0]
		data.pathType = file
		data.files = []fileInfo{{vmFilePath, stat}}
	}

	asmFileName := data.name + ".asm"

	asmFile, err := os.Create(asmFileName)
	if err != nil {
		log.Fatalf("Unable to create %s: %w", asmFileName, err)
	}

	defer asmFile.Close()

	t := NewTranslator(asmFile, data)

	err = t.translate()
	if err != nil {
		log.Printf("the translation finished with errors: %w", err)
	}
}
