package core

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func SaveAsJson(filePath string, any interface{}) {
	jsonData, err := JSONMarshal(any)
	if err != nil {
		Throw(err)
	}

	// Open or create a file for writing
	fmt.Printf("Writing %s...\n", filePath)
	file := CreateFile(filePath)
	defer file.Close()

	// Write the indented JSON to the file
	_, err = file.Write(jsonData)
	if err != nil {
		Throw(err)
	}
}

func LoadFromJson(filePath string, any interface{}) {
	fmt.Printf("Reading %s...\n", filePath)
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		Throw(err)
	}
	json.Unmarshal(jsonData, any)
}

type CsvSupported interface {
	ReadFromCsv(r *csv.Reader)
	WriteAsCsv(w *csv.Writer)
}

func GoStrToCsvStr(str string) string {
	return strings.ReplaceAll(str, "\r\n", "<br>")
}

func CsvStrToGoStr(str string) string {
	return strings.ReplaceAll(str, "<br>", "\r\n")
}

func LoadFromCsv(filePath string, obj CsvSupported) {
	// Open or create a file for writing
	fmt.Printf("Reading %s...\n", filePath)
	file := OpenFile(filePath)
	defer file.Close()

	reader := csv.NewReader(file)
	obj.ReadFromCsv(reader)
}

func SaveAsCsv(filePath string, obj CsvSupported) {
	// Open or create a file for writing
	fmt.Printf("Writing %s...\n", filePath)
	file := CreateFile(filePath)
	defer file.Close()

	writer := csv.NewWriter(file)
	obj.WriteAsCsv(writer)
	writer.Flush()
}
