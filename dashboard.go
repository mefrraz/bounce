package main

import (
	"os"
	"runtime"
)

// Set at build time with -ldflags:
//   go build -ldflags "-X main.current_commit=$(git rev-parse --short HEAD)"
var current_commit = "unknown"

func main() {
	_, _ = os.Stderr.Write([]byte("commit=" + current_commit + "\n"))
	_, _ = os.Stderr.Write([]byte("goroutines=" + itoa(runtime.NumGoroutine()) + "\n"))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte(n%10) + '0'
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
