package utils

import (
	"fmt"
	"runtime"

	"github.com/jwalton/gchalk"
)

type ErrNoInputValue struct {
	PortName string
}

func (m *ErrNoInputValue) Error() string {
	return fmt.Sprintf("no value for input '%v'", m.PortName)
}

type ErrUnknownPort struct {
	PortName string
}

func (m *ErrUnknownPort) Error() string {
	return fmt.Sprintf("unknown port %v", m.PortName)
}

// This logs the function name as well.
func Throw(err error) error {
	if err != nil {
		// Notice that we're using 1, so it will actually log where
		// the error happened, 0 = this function, we don't want that.
		pc, filename, line, _ := runtime.Caller(1)

		loc := fmt.Sprintf("%s:%d", filename, line)
		funcname := gchalk.WithBold().Magenta(runtime.FuncForPC(pc).Name())

		err = fmt.Errorf("Error in [%s]\n  %s(..)\n%w", gchalk.Cyan(loc), funcname, err)
	}
	return err
}
