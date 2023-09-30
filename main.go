package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

var (
	bufsize  = int64(*bufsizeFlag * 1024 * 1024)
	bbufsize = *browserBufsizeFlag * 1024 * 1024
	buf      = NewBuf(bufsize)
)

type Buf struct {
	buffer []byte
	n      int
	mu     sync.Mutex
	eof    int // eof offset
}

func NewBuf(size int64) *Buf {
	return &Buf{
		buffer: make([]byte, size),
		eof:    -1,
	}
}

func (b *Buf) startCaptureStdin() {
	for {
		b.mu.Lock()
		n, err := os.Stdin.Read(b.buffer[b.n:])
		if err != nil && err != io.EOF {
			L.Error(err, "read from stdin error")
			os.Exit(1)
		}
		b.n += n
		b.mu.Unlock()

		if err == io.EOF {
			b.eof = b.n
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (b *Buf) eofOffset() int {
	return b.eof
}

func (b *Buf) newReader() io.Reader {
	return bytes.NewReader(b.buffer[:b.n])
}

func (b *Buf) bytes(off, end int) []byte {
	realEnd := min(end, b.n)
	if off >= realEnd {
		return nil
	}
	res := make([]byte, realEnd-off)
	copy(res, b.buffer[off:])
	return res
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func main() {
	pflag.Parse()

	if *helpFlag {
		pflag.Usage()
		return
	}

	go buf.startCaptureStdin()

	bup := NewBup()

	// random port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		L.Error(err, "listen to random port failed")
		os.Exit(2)
	}

	L.Info("listening", "addr", listener.Addr().String())

	// open url in default browser
	openBrowser(fmt.Sprintf("http://%s", listener.Addr().String()))

	http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bup.ServeHTTP(w, r)
	}))
}
