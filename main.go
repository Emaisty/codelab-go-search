package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type HashResult struct {
	file string
	hash string
	err  error
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

func md5File(path string) (r HashResult) {
	r.file = path
	f, err := os.Open(path)
	if err != nil {
		r.err = err
		return
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		r.err = err
		return
	}
	hashBytes := h.Sum(nil)
	r.err = nil
	r.hash = fmt.Sprintf("%x", hashBytes)
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

func md5Files(files chan string) chan HashResult {
	res := make(chan HashResult)
	go func() {
		for f := range files {
			res <- md5File(f)
		}
		close(res)
	}()
	return res
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
	done := make(chan bool)
	start := time.Now()
	files, errc := walkDirectory(done, ".")
	pattern := "kek"
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
