package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func exit(format string, val ...interface{}) {
	if len(val) == 0 {
		fmt.Println(format)
	} else {
		fmt.Printf(format, val)
		fmt.Println()
	}
	os.Exit(1)
}

type ScanResult struct {
	file       string
	lineNumber int
	line       string
	err        error
}

func scanFile(fpath, pattern string) (r ScanResult) {
	f, err := os.Open(fpath)
	if err != nil {
		r.err = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	counter := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			r.file = fpath
			r.lineNumber = counter
			r.line = line
			r.err = nil
		}
		counter++
	}
	if err := scanner.Err(); err != nil {
		return
	}
	return
}

func walkDirectory(done chan bool, root string) (chan string, chan error) {
	files := make(chan string)
	// Канал в котором можно оставить 1 элемент(1 ошибку)
	errc := make(chan error, 1)
	go func() {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			select {
			case <-done:
				return errors.New("walk canceled by user")
			default: // Nothing.
			}
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files <- path
			}
			return nil
		})
		errc <- err
		close(files)
	}()
	return files, errc
}

func md5FilesParallel(done chan bool, files chan string, n int, pattern string) chan ScanResult {
	res := make(chan ScanResult)
	if n == 0 {
		fmt.Println("KEK")
		go func() {
			var wg sync.WaitGroup
			for f := range files {
				wg.Add(1)
				go func() {
					select {
					case <-done: //Canceled
					case res <- scanFile(f, pattern): // OK
					}
					wg.Done()
				}()
			}
			wg.Wait()
			close(res)
		}()
		return res
	}
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			for f := range files {
				select {
				case <-done: // Canceled
				case res <- scanFile(f, pattern): // OK
				}
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(res)
	}()
	return res
}

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		exit("usage: go-search <path> <pattern> to search")
	}
	path := flag.Arg(0)
	pattern := flag.Arg(1)
	done := make(chan bool)
	start := time.Now()
	fmt.Println(path, pattern)
	files, errc := walkDirectory(done, path)
	// Синтетика.
	//go func() {
	//    time.Sleep(1 * time.Second)
	//    close(done)
	//}()
	//results := md5Files(files)
	results := md5FilesParallel(done, files, 20, pattern)
	for res := range results {
		if res.file != "" {
			fmt.Println(res.file, res.lineNumber, res.line, res.err)
		}
	}
	fmt.Println(<-errc)
	fmt.Println("Elapsed: ", time.Now().Sub(start))
}
