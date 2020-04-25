package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	numberofstring = flag.Bool("n", false, "print file ")
)

type ScanResult struct {
	file       string
	lineNumber int
	line       string
}

func scanFile(fpath, pattern string) ([]ScanResult, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var result []ScanResult
	counter := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			linewithpattern := ScanResult{file: fpath, lineNumber: counter, line: line}
			result = append(result, linewithpattern)
		}
		counter++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func exit(format string, val ...interface{}) {
	if len(val) == 0 {
		fmt.Println(format)
	} else {
		fmt.Printf(format, val)
		fmt.Println()
	}
	os.Exit(1)
}

func processFile_gorut(fpath string, pattern string, wg *sync.WaitGroup) {

	defer wg.Done()

	res, err := scanFile(fpath, pattern)
	if err != nil {
		exit("Error scanning %s: %s", fpath, err.Error())
	}
	if *numberofstring {
		for _, line := range res {
			fmt.Println(line.file, ":", line.lineNumber, ":", line.line)
		}
	} else {
		for _, line := range res {
			fmt.Println(line.file, ":", line.line)
		}
	}
}

func processFile_single(fpath string, pattern string) {

	res, err := scanFile(fpath, pattern)
	if err != nil {
		exit("Error scanning %s: %s", fpath, err.Error())
	}
	if *numberofstring {
		for _, line := range res {
			fmt.Println(line.file, ":", line.lineNumber, ":", line.line)
		}
	} else {
		for _, line := range res {
			fmt.Println(line.file, ":", line.line)
		}
	}
}

func processDirectory(dir string, pattern string) {
	var wg sync.WaitGroup
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			panic(err)
		}
		if !info.IsDir() {
			wg.Add(1)
			go processFile_gorut(path, pattern, &wg)
		}

	}
	wg.Wait()
}

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		exit("usage: go-search <path> <pattern> to search")
	}

	path := flag.Arg(0)
	pattern := flag.Arg(1)

	info, err := os.Stat(path)
	if err != nil {
		panic(err)
	}

	if info.IsDir() {
		processDirectory(path, pattern)
	} else {
		processFile_single(path, pattern)
	}
}
