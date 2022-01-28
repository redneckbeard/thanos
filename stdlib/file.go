package stdlib

import (
	"bufio"
	"bytes"
	"os"
)

func MakeSplitFunc(separator string, chomp bool) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, []byte(separator)); i >= 0 {
			upper := i
			if !chomp {
				upper += len(separator)
			}
			return i + len(separator), data[:upper], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}

var OpenModes = map[string]int{
	"r":  os.O_RDONLY,
	"r+": os.O_RDWR,
	"w":  os.O_WRONLY | os.O_CREATE,
	"w+": os.O_RDWR | os.O_CREATE | os.O_TRUNC,
	"a":  os.O_WRONLY | os.O_CREATE | os.O_APPEND,
	"a+": os.O_RDWR | os.O_CREATE | os.O_APPEND,
}
