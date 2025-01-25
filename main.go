package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	flag "github.com/spf13/pflag"

	core "ff7r-text-tool/core"
)

var TOOL_VERSION string = "0.1.0"

type options struct {
	files      []string
	mode       string // export or import
	outdir     string
	format     string // csv or json
	numWorkers int
	verbose    bool
}

var MODE_LIST = []string{
	"export",
	"import",
}

var FORMAT_LIST = []string{
	"csv",
	"json",
}

// Parse arguments
func argparse() *options {
	args := &options{}
	flag.StringVarP(&args.mode, "mode", "m", "export", "export or import is available")
	flag.StringVarP(&args.format, "format", "f", "csv", "csv or json")
	flag.StringVarP(&args.outdir, "outdir", "o", "out", "path to output directory")
	flag.BoolVarP(&args.verbose, "verbose", "v", false, "shows more information")
	flag.IntVarP(&args.numWorkers, "num_workers", "n", 0, "number of worker processes. 0 means the number of CPUs")
	flag.Parse()

	// Check string options
	if !slices.Contains(MODE_LIST, args.mode) {
		core.Throw(fmt.Errorf("unknown mode detected (%s)", args.mode))
	}
	if !slices.Contains(FORMAT_LIST, args.format) {
		core.Throw(fmt.Errorf("unknown format detected (%s)", args.format))
	}

	// Convert paths to absolute paths
	rawFiles := flag.Args()
	if len(rawFiles) == 0 {
		core.Throw("you should specify a file path.")
	}
	if args.mode == "import" && len(rawFiles) == 1 {
		core.Throw("asset path is missing for import mode.")
	}
	args.files = make([]string, 0, len(args.files))
	for _, file := range rawFiles {
		args.files = append(args.files, core.GetFullPath(file))
	}

	args.outdir = core.MakeDir(args.outdir)

	// Get num workers
	if args.numWorkers <= 0 {
		args.numWorkers = runtime.NumCPU()
	}

	fmt.Printf("mode: %s\n", args.mode)
	fmt.Printf("outdir: %s\n", args.outdir)
	fmt.Printf("num_workers: %d\n", args.numWorkers)
	return args
}

func processFile(filePath string, rootdir string, assetRoot string, args *options) {
	parentDir, baseName, _ := core.SplitFilePath(filePath)
	relPath, err := filepath.Rel(rootdir, filePath)
	if err != nil {
		core.Throw(err)
	}
	relPath = filepath.Dir(relPath)
	outdir := core.MakeDir(filepath.Join(args.outdir, relPath))

	if args.mode == "export" {
		// Read .uasset
		uasset := core.Uasset{}
		uassetPath := filepath.Join(parentDir, baseName+".uasset")
		uasset.ReadFromFile(uassetPath)

		outPath := filepath.Join(outdir, baseName+"."+args.format)
		if args.format == "csv" {
			// Save as .csv
			core.SaveAsCsv(outPath, uasset.Uexp)
		} else {
			// Save as .json
			core.SaveAsJson(outPath, uasset.Uexp)
		}
	} else {
		// Read .uasset
		uasset := core.Uasset{}
		uassetPath := filepath.Join(assetRoot, relPath, baseName+".uasset")
		uasset.ReadFromFile(uassetPath)

		newDataPath := filepath.Join(parentDir, baseName+"."+args.format)
		if args.format == "csv" {
			// Read .csv
			core.LoadFromCsv(newDataPath, uasset.Uexp)
		} else {
			// Read .json
			newUexp := &core.Uexp{}
			core.LoadFromJson(newDataPath, newUexp)
			uasset.Uexp.UpdateWithNewUexp(newUexp)
		}

		// Save .uasset and .uexp
		newUassetPath := filepath.Join(outdir, baseName+".uasset")
		uasset.WriteToFile(newUassetPath)
	}
}

func multiProcessFiles(filePath string, assetRoot string, targetExt string, args *options) int {
	fileCount := 0
	fileChan := make(chan string, 128)
	var wg sync.WaitGroup
	var countMutex sync.Mutex
	rootdir, _ := core.SplitPath(filePath)

	// Start worker goroutines
	for i := 0; i < args.numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer core.ErrorCheck()
			defer wg.Done()
			for file := range fileChan {
				processFile(file, rootdir, assetRoot, args)
				countMutex.Lock()
				fileCount++
				countMutex.Unlock()
			}
		}()
	}

	// Search files and send queues
	err := filepath.WalkDir(filePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %v", path, err)
		}
		if !d.IsDir() {
			ext := filepath.Ext(path)
			if ext == targetExt {
				fileChan <- path // Send file path to the channel
			}
		}
		return nil
	})
	if err != nil {
		core.Throw(err)
	}

	close(fileChan)
	wg.Wait()
	return fileCount
}

func main() {
	start := time.Now()

	fmt.Printf("ff7r-text-tool v%s by Matyalatte\n", TOOL_VERSION)

	// Remove time info from log
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Catch panic and show backtraces
	defer core.ErrorCheck()

	args := argparse()
	filePath := args.files[0]
	assetRoot := ""
	if args.mode == "import" {
		assetRoot, _ = core.SplitPath(args.files[1])
	}

	targetExt := ".uasset"
	if args.mode == "import" {
		targetExt = "." + args.format // .csv or .json
	}

	fileCount := 0

	if core.PathIsDir(filePath) {
		fileCount = multiProcessFiles(filePath, assetRoot, targetExt, args)
	} else {
		parentDir, _, ext := core.SplitFilePath(filePath)
		if ext != targetExt {
			core.Throw(fmt.Errorf("not %s. (%s)", targetExt, filePath))
		}
		processFile(filePath, parentDir, assetRoot, args)
		fileCount++
	}

	// Print result
	duration := time.Since(start)
	if fileCount == 0 {
		fmt.Printf("No files processed...\n")
	} else if fileCount == 1 {
		fmt.Printf("Done! processed 1 file in %v\n", duration)
	} else {
		fmt.Printf("Done! processed %d files in %v\n", fileCount, duration)
	}
}
