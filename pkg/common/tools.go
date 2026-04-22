package common

import (
	"fmt"
	"runtime"
)

func WrapErr(err error) error {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return err
	}
	fn := runtime.FuncForPC(pc)
	return fmt.Errorf("%s:%d %s: %w", file, line, fn.Name(), err)
}
