package core

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sync"
)

type Error struct {
	hasError   bool
	err        error
	backtraces string
}

func (e *Error) Init() {
	e.hasError = false
	e.err = nil
	e.backtraces = ""
}

func (e *Error) HasError() bool {
	return e.hasError
}

func (e *Error) GetError() error {
	return e.err
}

func (e *Error) GetBacktraces() string {
	return e.backtraces
}

func (e *Error) GetErrorWithTraces() string {
	return fmt.Sprintf(
		"%s\nError: %s\n", e.GetBacktraces(), e.GetError())
}

func (e *Error) Throw(any interface{}, traceStart int) {
	if e.hasError {
		return
	}
	e.hasError = true

	switch v := any.(type) {
	case string:
		e.err = errors.New(v)
	case error:
		e.err = v
	default:
		e.err = fmt.Errorf("you can NOT thorw %s", reflect.TypeOf(any).String())
	}

	// Make python-like traceback text
	e.backtraces = ""
	i := traceStart
	for {
		pt, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		funcName := runtime.FuncForPC(pt).Name()
		if funcName == "runtime.goexit" {
			break
		}
		e.backtraces = fmt.Sprintf("\n  File \"%s\", line %d, in %v%s", file, line, funcName, e.backtraces)
		if funcName == "main.main" {
			break
		}
		i += 1
	}
	e.backtraces = "Traceback (most recent call last):" + e.backtraces
	panic(e.err)
}

var globalError Error = Error{}

func Throw(any interface{}) {
	globalError.Init()
	globalError.Throw(any, 2)
}

func ThrowBase(any interface{}, traceStart int) {
	globalError.Init()
	globalError.Throw(any, traceStart)
}

func HasError() bool {
	return globalError.HasError()
}

func GetError() error {
	return globalError.GetError()
}

func GetErrorWithTraces() string {
	return globalError.GetErrorWithTraces()
}

var logMutex sync.Mutex

func ErrorCheck() {
	// catch panic and show backtraces
	if globalError.HasError() {
		logMutex.Lock()
		log.Fatal(globalError.GetErrorWithTraces())
		logMutex.Unlock()
	}
}

func ErrorCheckWithMsg(msg string) {
	// catch panic and show backtraces
	if globalError.HasError() {
		logMutex.Lock()
		str := msg + globalError.GetErrorWithTraces()
		log.Fatal(str)
		logMutex.Unlock()
	}
}
