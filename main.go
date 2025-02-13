package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	flag "github.com/spf13/pflag"

	core "ff7r-text-tool/core"
)

var TOOL_VERSION string = "0.1.0"

type options struct {
	files            []string
	mode             string // export or import
	outdir           string
	format           string // csv or json
	numWorkers       int
	verbose          bool
	ignoreEmpty      bool
	subtitleBoxWidth int
	subttleBoxHeight int
}

var MODE_LIST = []string{
	"export",
	"import",
	"dualsub",
	"resize",
	"test",
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
	flag.BoolVarP(&args.ignoreEmpty, "ignore_empty", "i", false, "ignores empty assets")
	flag.IntVarP(&args.numWorkers, "num_workers", "n", 0, "number of worker processes. 0 means the number of CPUs")
	flag.IntVar(&args.subtitleBoxWidth, "width", 930, "width of subtitle widget. the original width is 930")
	flag.IntVar(&args.subttleBoxHeight, "height", 210, "height of subtitle widget. the original height is 210")
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
	if (args.mode == "import" || args.mode == "dualsub") && len(rawFiles) == 1 {
		core.Throw(fmt.Errorf("asset path is missing for this mode. (%s)", args.mode))
	}
	args.files = make([]string, 0, len(args.files))
	for _, file := range rawFiles {
		args.files = append(args.files, core.GetFullPath(file))
	}

	if args.mode == "resize" && !strings.HasSuffix(args.files[0], "Subtitle00.uasset") {
		core.Throw(fmt.Errorf("you should specify Subtitle00.uasset for this mode. (%s)", args.files[0]))
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

func Export(uassetPath string, outPath string, args *options) int {
	// Read .uasset
	uasset := core.Uasset{}
	uasset.ReadFromFile(uassetPath)

	if args.ignoreEmpty && len(uasset.Uexp.Entries) == 0 {
		return 0 // Do not export empty assets
	}

	if args.format == "csv" {
		// Save as .csv
		core.SaveAsCsv(outPath, uasset.Uexp)
	} else {
		// Save as .json
		core.SaveAsJson(outPath, uasset.Uexp)
	}

	return 1
}

func Import(uassetPath string, newDataPath string, outPath string, args *options) int {
	// Read .uasset
	uasset := core.Uasset{}
	uasset.ReadFromFile(uassetPath)

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
	uasset.WriteToFile(outPath)

	return 1
}

func Dualsub(firstPath string, secondPath string, outPath string, args *options) int {
	// Read .uasset
	uasset1 := core.Uasset{}
	uasset1.ReadFromFile(firstPath)

	if args.ignoreEmpty && len(uasset1.Uexp.Entries) == 0 {
		return 0 // Do not export empty assets
	}

	uasset2 := core.Uasset{}
	uasset2.ReadFromFile(secondPath)

	mergedCount := core.MakeDualsub(uasset1.Uexp, uasset2.Uexp)

	if mergedCount == 0 {
		return 0
	}
	// Save .uasset and .uexp
	uasset1.WriteToFile(outPath)
	return 1
}

func filesAreEqual(file1Path, file2Path string) (bool, error) {
	fmt.Printf("Comparing %s and %s...\n", file1Path, file2Path)
	// Open the first file
	file1, err := os.Open(file1Path)
	if err != nil {
		return false, err
	}
	defer file1.Close()

	// Open the second file
	file2, err := os.Open(file2Path)
	if err != nil {
		return false, err
	}
	defer file2.Close()

	// Get file sizes
	file1Info, err := file1.Stat()
	if err != nil {
		return false, err
	}
	file2Info, err := file2.Stat()
	if err != nil {
		return false, err
	}

	// Compare file sizes
	if file1Info.Size() != file2Info.Size() {
		return false, nil
	}

	// Compare file contents
	buf1 := make([]byte, 4096)
	buf2 := make([]byte, 4096)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}

		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("error reading files: %v, %v", err1, err2)
		}
	}

	return true, nil
}

func processFile(filePath string, rootDir string, assetDir string, args *options) int {
	parentDir, baseName, _ := core.SplitFilePath(filePath)
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		core.Throw(err)
	}
	relPath = filepath.Dir(relPath)

	var outdir string
	var secondPath string
	if assetDir == "" || (core.PathExists(assetDir) && core.PathIsDir(assetDir)) {
		_, rootBase := core.SplitPath(rootDir)
		outdir = core.MakeDir(filepath.Join(args.outdir, rootBase, relPath))
		secondPath = filepath.Join(assetDir, relPath, baseName+".uasset")
	} else {
		outdir = core.MakeDir(filepath.Join(args.outdir, relPath))
		secondPath = assetDir
	}

	processed := 0

	if args.mode == "export" {
		uassetPath := filepath.Join(parentDir, baseName+".uasset")
		outPath := filepath.Join(outdir, baseName+"."+args.format)
		processed = Export(uassetPath, outPath, args)
	} else if args.mode == "import" {
		newDataPath := filepath.Join(parentDir, baseName+"."+args.format)
		outPath := filepath.Join(outdir, baseName+".uasset")
		processed = Import(secondPath, newDataPath, outPath, args)
	} else if args.mode == "dualsub" {
		firstPath := filepath.Join(parentDir, baseName+".uasset")
		outPath := filepath.Join(outdir, baseName+".uasset")
		processed = Dualsub(firstPath, secondPath, outPath, args)
	} else if args.mode == "resize" {
		firstPath := filepath.Join(parentDir, baseName+".uasset")
		outPath := filepath.Join(outdir, baseName+".uasset")
		core.ResizeSubtitleWidget(firstPath, outPath, args.subtitleBoxWidth, args.subttleBoxHeight)
		processed = 1
	} else if args.mode == "test" {
		uassetPath := filepath.Join(parentDir, baseName+".uasset")
		newDataPath := filepath.Join(outdir, baseName+"."+args.format)
		Export(uassetPath, newDataPath, args)
		newUassetPath := filepath.Join(outdir, baseName+".uasset")
		Import(uassetPath, newDataPath, newUassetPath, args)
		eq, err := filesAreEqual(uassetPath, newUassetPath)
		if err != nil {
			core.Throw(err)
		}
		if !eq {
			core.Throw(fmt.Errorf("failed to reconstruct asset file. (%s)", uassetPath))
		}
		processed = 1
	}
	return processed
}

func multiProcessFiles(filePath string, assetPath string, targetExt string, args *options) int {
	fileCount := 0
	fileChan := make(chan string, 128)
	var wg sync.WaitGroup
	var countMutex sync.Mutex

	// Start worker goroutines
	for i := 0; i < args.numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer core.ErrorCheck()
			defer wg.Done()
			for file := range fileChan {
				processed := processFile(file, filePath, assetPath, args)
				countMutex.Lock()
				fileCount += processed
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
	assetPath := ""
	if args.mode == "import" || args.mode == "dualsub" {
		assetPath = args.files[1]
	}

	targetExt := ".uasset"
	if args.mode == "import" {
		targetExt = "." + args.format // .csv or .json
	}

	fileCount := 0

	if core.PathIsDir(filePath) {
		fileCount = multiProcessFiles(filePath, assetPath, targetExt, args)
	} else {
		parentDir, _, ext := core.SplitFilePath(filePath)
		if ext != targetExt {
			core.Throw(fmt.Errorf("not %s. (%s)", targetExt, filePath))
		}
		processed := processFile(filePath, parentDir, assetPath, args)
		fileCount += processed
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
