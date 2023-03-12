package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type thread struct {
	mux      sync.RWMutex
	mng      *Manager
	begin    int
	end      int
	stat     Result
	client   *http.Client
	file     *os.File
	totWrite int
	node     *Node[*thread]
}

func (th *thread) run() {
	req, err := http.NewRequest("GET", th.mng.URL, nil)
	if err != nil {
		th.abort(err.Error())
		return
	}

	r := fmt.Sprintf("bytes=%d-%d", th.begin, th.end)
	req.Header.Add("Range", r)
	
	resp, err := th.client.Do(req)
	if err != nil {
		th.abort(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		th.abort("Thread init download failed: server doesn't support partial download? " + resp.Status)
		return
	}
	unit, begin, end, size, ok := getContentRange(&resp.Header)
	if !ok {
		th.abort("No begin/end info or syntax error in response header. ")
		return
	}
	if unit != "bytes" {
		th.abort("Unsupported unit type: " + unit)
		return
	}

	if begin == -1 && end == -1 {
		if size != th.end-th.begin+1 {
			th.abort("Responded content size doesn't match required content size. ")
			return
		}
	} else if begin != th.begin || end != th.end {
		th.abort("Responded begin/end doesn't match required begin/end. ")
		return
	}

	th.file, err = os.CreateTemp("", th.mng.fileName+".*.godown")
	if err != nil {
		th.abort("Failed to create temp file. Reason: " + err.Error())
	}

	buffer := make([]byte, 5*1024*1024)
	for {
		nr, rErr := resp.Body.Read(buffer)
		if rErr != nil && rErr != io.EOF {
			th.abort("Error downloading. Reason: " + rErr.Error())
			return
		}

		n, wErr := th.file.Write(buffer[0:nr])
		th.mux.Lock()
		th.totWrite += n
		th.mux.Unlock()
		if wErr != nil {
			th.abort("Error writing to temp file. Reason: " + wErr.Error())
			return
		}

		th.mux.RLock()
		if rErr == io.EOF || th.totWrite >= th.end-th.begin+1 {
			th.mux.RUnlock()
			th.success("")
			return
		}
		th.mux.RUnlock()
	}
}

func (th *thread) success(msg string) {
	th.mux.Lock()
	th.stat = Result{
		code: Success,
		msg:  msg,
	}
	th.mux.Unlock()
	th.mng.notify <- &th.stat
}

func (th *thread) abort(msg string) {
	th.mux.Lock()
	th.stat = Result{
		code: Abort,
		msg:  msg,
	}
	th.mux.Unlock()
	if th.totWrite == 0 {
		th.mng.mux.Lock()
		th.mng.list.DelNode(th.node)
		th.mng.mux.Unlock()

		if th.file != nil {
			th.file.Close()
			os.Remove(th.file.Name())
		}
	}

	th.mng.notify <- &th.stat
}
