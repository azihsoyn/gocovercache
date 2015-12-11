package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	outDir       string
	coverprofile string
	verbose      bool
	parallel     int
)

func init() {
	flag.StringVar(&outDir, "outdir", ".cache", "cache directory")
	flag.StringVar(&coverprofile, "coverprofile", "profile.cov", "coverage report file")
	flag.IntVar(&parallel, "parallel", runtime.NumCPU(), "parallel number")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("create %s failed. %s", outDir, err)
	}
}

func getDirMD5(h *hash.Hash) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		(*h).Write(buf)

		return nil
	}
}

func calcCheckSum(dirpath string) string {
	var checksum string
	h := md5.New()
	err := filepath.Walk(dirpath, getDirMD5(&h))
	if err != nil {
		fmt.Println("calcCheckSum failed.", err)
		os.Exit(1)
	}
	checksum = hex.EncodeToString(h.Sum(nil))
	return checksum
}

func runTest(pkg, checksum string) {
	name := strings.Replace(pkg, "/", ".", -1)

	filefmt := `%s.profile.%s`
	filename := fmt.Sprintf(filefmt, name, checksum)
	path := fmt.Sprintf("%s/%s", outDir, filename)

	// skip test if same file exists
	if _, err := os.Stat(path); err == nil {
		if verbose {
			fmt.Printf("pkg %s not changed. profile already exists.\n", pkg)
		}
		return
	}
	if verbose {
		fmt.Printf("pkg %s changed. do tests.\n", pkg)
	}
	out, err := exec.Command("go", "test", "-covermode=count", fmt.Sprintf("-coverprofile=%s", path), pkg).Output()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Print(string(out))
}

func getPackageList() []string {
	var pkgs = []string{}

	cmd := exec.Command("go", "list", "./...")
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		pkgs = append(pkgs, scanner.Text())
	}
	cmd.Wait()

	return pkgs
}

func removeOldReport(pkg, checksum string) {
	name := strings.Replace(pkg, "/", ".", -1)
	err := filepath.Walk(outDir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			// old report
			if strings.Contains(path, name) && !strings.Contains(path, checksum) {
				if err := os.Remove(path); err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		fmt.Println("removeOldReport failed.", err)
		os.Exit(1)
	}
}

func getAbsolutePackageDir(pkg string) string {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println("couldn't get current dir", err)
		os.Exit(1)
	}
	// this is workaround...
	return fmt.Sprintf("%s/src/%s", strings.Split(pwd, "/src")[0], pkg)
}

func uniteReports() {
	f, err := os.Create(coverprofile)
	if err != nil {
		fmt.Printf("create %s failed. %s", coverprofile, err)
		os.Exit(1)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err := w.WriteString("mode: count\n"); err != nil {
		fmt.Printf("write failed. %s", err)
		os.Exit(1)
	}

	err = filepath.Walk(outDir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				fmt.Printf("file %s could not read: %v\n", path, err)
				return err
			}
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "mode:") {
					continue
				}

				_, err := w.WriteString(scanner.Text() + "\n")
				if err != nil {
					return err
				}
			}

			return nil
		})
	if err := w.Flush(); err != nil {
		fmt.Println("flush failed.", err)
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("uniteReports failed.", err)
		os.Exit(1)
	}
}

func main() {
	pkgs := getPackageList()

	var wg sync.WaitGroup
	// see http://deeeet.com/writing/2014/07/30/golang-parallel-by-cpu/
	semaphore := make(chan int, parallel)
	for _, pkg := range pkgs {
		wg.Add(1)
		go func(pkg string) {
			defer wg.Done()
			semaphore <- 1
			pkgDir := getAbsolutePackageDir(pkg)
			checksum := calcCheckSum(pkgDir)
			runTest(pkg, checksum)
			removeOldReport(pkg, checksum)
			<-semaphore
		}(pkg)
	}
	wg.Wait()

	uniteReports()
}
