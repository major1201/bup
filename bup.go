package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"time"
)

//go:embed bup.html
var bupHTML string

const QueryParamOp = "_ws_o"

type bup struct {
	tpl *template.Template
	cmd *exec.Cmd
	sh  []string
}

func NewBup() *bup {
	tpl := template.New("index")
	_, _ = tpl.Parse(bupHTML)
	shell := getShell()
	L.Info("found shell", "exec", shell)
	return &bup{
		tpl: tpl,
		sh:  shell,
	}
}

func (l *bup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch q.Get(QueryParamOp) {
	case "ws":
		l.handleWebsocket(w, r)
		return
	}

	q.Set(QueryParamOp, "ws")
	wsURI := fmt.Sprintf("%s%s?%s", r.Host, r.URL.Path, q.Encode())

	vars := map[string]string{
		"WEBSOCKET_URI": wsURI,
	}

	if err := l.tpl.Execute(w, vars); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errMsg := fmt.Sprintf("generate webpage failed : %v", err)
		_, _ = w.Write([]byte(errMsg))
	}
}

func (l *bup) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	command := q.Get("command")
	L.Info("new request", "command", command)

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conn := &wsConnection{conn: wsConn}
	closeFunc := func() {
		_ = conn.Close()
		_ = wsConn.Close()
	}

	var closed bool
	wsConn.SetCloseHandler(func(_ int, _ string) error {
		closed = true
		return nil
	})

	go func() {
		defer closeFunc()

		go func() {
			// read connection or you'll never perceive close event from websocket client
			// https://github.com/gorilla/websocket/issues/414
			_, _, _ = wsConn.NextReader()
		}()

		if l.cmd != nil {
			l.cmd.Process.Kill()
		}

		if command == "" {
			// keep following if not EOFed
			off := 0
			const interval = 100 * time.Millisecond
			for buf.eofOffset() != off {
				// break if ws closed
				if closed {
					break
				}

				b := buf.bytes(off, bbufsize)
				if len(b) == 0 {
					time.Sleep(interval)
					continue
				}
				conn.Write(b)
				if off+len(b) >= bbufsize {
					break
				}
				off += len(b)
				time.Sleep(interval)
			}
		} else {
			l.cmd = exec.CommandContext(context.Background(), l.sh[0], append(l.sh[1:], command)...)
			l.cmd.Stdin = buf.newReader()
			l.cmd.Stdout = conn
			l.cmd.Stderr = conn
			if err := l.cmd.Run(); err != nil {
				conn.Write([]byte(err.Error()))
			}
		}
	}()
}

func getShell() (res []string) {
	res = *shellFlag
	if len(res) > 0 {
		return
	}

	res = []string{"", "-c"}
	sh := os.Getenv("SHELL")
	if sh != "" {
		res[0] = sh
		return
	}
	sh, _ = exec.LookPath("bash")
	if sh != "" {
		res[0] = sh
		return
	}
	sh, _ = exec.LookPath("sh")
	if sh != "" {
		res[0] = sh
		return
	}
	panic("cannot find shell: no -e flag, $SHELL is empty, neither bash nor sh are in $PATH")
}
