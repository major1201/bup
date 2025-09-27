package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

var (
	bufsize  int64
	bbufsize int
	bufMap   = make(map[string]*Buf)
)

type Buf struct {
	buffer []byte
	n      int
	mu     sync.Mutex
	eof    int // eof offset
	_stop  bool

	ctx    context.Context
	cancel context.CancelFunc

	createTime time.Time
}

func NewBuf(size int64) *Buf {
	res := &Buf{
		buffer:     make([]byte, size),
		eof:        -1,
		createTime: time.Now(),
	}
	res.ctx, res.cancel = context.WithCancel(context.Background())
	return res
}

func (b *Buf) startCaptureReader(reader io.Reader) {
	for {
		if b._stop {
			return
		}

		b.mu.Lock()
		n, err := reader.Read(b.buffer[b.n:])
		if err != nil && err != io.EOF {
			L.Error(err, "read from stdin error")
			os.Exit(1)
		}
		b.n += n
		b.mu.Unlock()

		if err == io.EOF {
			b.eof = b.n
			break
		}
	}
}

func (b *Buf) startCaptureStdin() {
	b.startCaptureReader(os.Stdin)
}

func (b *Buf) startCaptureCommand(srcCommand string) {
	sh := getShell()
	cmd := exec.CommandContext(b.ctx, sh[0], append(sh[1:], srcCommand)...)
	reader, err := cmd.StdoutPipe()
	if err != nil {
		L.Error(err, "get stdout pipe failed")
		b.buffer = []byte(err.Error())
		return
	}
	cmd.Start()
	defer reader.Close()

	b.startCaptureReader(reader)
}

func (b *Buf) stop() {
	b._stop = true
	b.cancel()
}

func (b *Buf) eofOffset() int {
	return b.eof
}

func (b *Buf) newReader() io.Reader {
	if b.eof > 0 {
		return bytes.NewReader(b.buffer[:b.n])
	}

	return &BufReader{
		buf: b,
	}
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

type BufReader struct {
	buf *Buf
	n   int
}

func (r *BufReader) Read(b []byte) (n int, err error) {
	if r.buf.eof > 0 && r.buf.eof == r.n {
		return 0, io.EOF
	}

	n = copy(b, r.buf.buffer[r.n:r.buf.n])
	r.n += n
	return
}

func openBrowser(browser, url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
		switch strings.ToLower(browser) {
		case "": // default browser
		case "chrome":
			args = []string{"-a", "/Applications/Google Chrome.app"}
		case "safari":
			args = []string{"-a", "/Applications/Safari.app"}
		case "firefox":
			args = []string{"-a", "/Applications/Firefox.app"}
		default:
			args = []string{"-a", browser}
		}
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

	bufsize = int64(*bufsizeFlag * 1024 * 1024)
	bbufsize = *browserBufsizeFlag * 1024 * 1024

	bup := NewBup()

	var listener net.Listener
	var err error
	if *daemonModeFlag {
		listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", *portFlag))
	} else {
		bufMap[""] = NewBuf(bufsize)
		go bufMap[""].startCaptureStdin()

		// random port
		listener, err = net.Listen("tcp", "localhost:0")
	}
	if err != nil {
		L.Error(err, "listen to random port failed")
		os.Exit(2)
	}

	L.Info("listening", "addr", listener.Addr().String())

	// open url in default browser
	url := fmt.Sprintf("http://%s", listener.Addr().String())
	L.Info("open following link to get access", "url", url)
	if !*daemonModeFlag {
		// wait for server to start
		openBrowser(*browserFlag, url)
	} else {
		// start session GC in daemon mode
		go startGC()
	}

	http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bup.ServeHTTP(w, r)
	}))
}
