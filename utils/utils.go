package utils

import (
	"fmt"
	"runtime"
)

func ErrorWithTrace(e error) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%s:%d\n\t%v", file, line, e)
}
