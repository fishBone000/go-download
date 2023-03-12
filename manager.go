package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	Abort   = -1
	Success = 0
	Running = 1
	maxFail = 10
)

type Result struct {
	code int
	msg  string
}

type Manager struct {
	MaxThread int
	URL       string
	Res       chan Result

	mux      sync.RWMutex
	list     List[*thread]
	totSize  int
	runCnt   int
	notify   chan *Result
	fileName string
}

func (mng *Manager) Run() {
	if mng.MaxThread == 0 {
		mng.abort("Invalid max thread. ")
		return
	}

	req, rngErr := http.NewRequest("HEAD", mng.URL, nil)
	if rngErr != nil {
		mng.abort("Failed when building a new HTTP request: ", rngErr.Error())
		return
	}

	req.Header.Add("Range", "bytes=0-127")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		mng.abort("Error sending HTTP request: ", err.Error())
		return
	}
	if resp.StatusCode != http.StatusPartialContent {
		switch resp.StatusCode {
		case http.StatusOK:
			mng.abort("Server ignored ranged downloading for this file, in this case, why don't you use some other downloaders?")
			return
		case http.StatusRequestedRangeNotSatisfiable:
			mng.abort("Server doesn't support ranged downloading for this file. ")
			return
		default:
			mng.abort("Unexpected status code: ", resp.Status)
			return
		}
	}

	var szStr string
	var sz int
	_, ok := resp.Header["Content-Range"]
	if ok {
		szStr = resp.Header["Content-Range"][0]
		var found bool
		_, szStr, found = strings.Cut(szStr, "/")
		i, err := strconv.Atoi(szStr)
		if szStr == "*" || !found || err != nil {
			mng.abort("Cannot acquire total size. ")
			return
		}
		sz = i
	} else {
		mng.abort("Server didn't respond with total size. ")
		return
	}

	mng.totSize = sz
	mng.fileName = mng.URL[strings.LastIndex(mng.URL, "/")+1:]

	mng.notify = make(chan *Result, mng.MaxThread)
	mng.initDownload()

	failCnt := 0
	for {
		res := <-mng.notify
		mng.runCnt--
		brk := false
		switch res.code {
		case Abort:
			mng.sitrep("Thread failed: ", res.msg, " Running threads: ", mng.runCnt)

			if failCnt > maxFail {
				if mng.runCnt == 0 {
					mng.abort("We failed man...")
					return
				}
				continue
			}
			mng.allocThread()
		case Success:
			mng.sitrep("1 thread just finished downloading. ")
			mng.allocThread()
			if mng.runCnt == 0 {
				mng.sitrep("All threads finished downloading. ")
				brk = true
			}
		}
		if brk { break }
	}

	// wd, err := os.Getwd()
	// if err != nil {
	// 	mng.abort("Unable to get working directory. Reason: ", err.Error())
	// 	return
	// }
	file, err := os.OpenFile(mng.fileName, os.O_CREATE | os.O_RDWR, 0755)
	if err != nil {
		mng.abort("Error creating file. Reason: ", err)
		return
	}
	for p := mng.list.Front(); p != nil; p = p.Next() {
		th := p.Content()
		_, err = th.file.Seek(0, 0)
		if err != nil {
			mng.abort("Error reseting temp file offset. Reason: ", err.Error())
			return
		}
		file.ReadFrom(th.file)
		th.file.Close()
		os.Remove(th.file.Name())
	}
	file.Close()
	mng.success("Success! Merged all temp files into ", file.Name())
}

// func (mng *Manager) checkCompl() bool {
// 	mng.mux.RLock()
// 	defer mng.mux.RUnlock()
// 	p := mng.list.Front()
// 	th := p.Content()
// 	if th.begin != 0 {
// 		return false
// 	}
// 	for ; p != nil; p = p.Next() {
// 		th = p.Content()
// 		th.mux.RLock()
// 		defer th.mux.RUnlock()

// 		if th.stat.code == Running {
// 			return false
// 		}
// 		if p.Next() == nil {
// 			return p.Content().end == mng.totSize-1
// 		} else {
// 			nth := p.Next().Content()
// 			nth.mux.RLock()
// 			defer nth.mux.RUnlock()
// 			if th.end+1 != nth.begin {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

func (mng *Manager) allocThread() {
	cnt := 0
	mng.mux.Lock()
	defer mng.mux.Unlock()
	if mng.list.Empty() {
		mng.newThread(0, mng.totSize-1)
		cnt++
		return
	}
	p := mng.list.Front()
	if p.Content().begin != 0 {
		mng.newThread(0, p.Content().begin-1)
		cnt++
		return
	}

	for ; p != nil; p = p.Next() {
		th := p.Content()
		switch th.stat.code {
		case Success:
			fallthrough
		case Abort:
			if p.next != nil {
				nextBegin := p.Next().Content().begin
				if th.end+1 != nextBegin {
					mng.newThread(th.end+1, nextBegin-1)
					cnt++
					return
				} else {
					continue
				}
			} else if th.end+1 != mng.totSize {
				mng.newThread(th.end+1, mng.totSize-1)
				cnt++
				return
			} else {
				return
			}
		case Running:
			assigned := th.end - th.begin + 1
			if float64(assigned - th.totWrite)/float64(mng.totSize) > 0.05 {
				begin := th.begin + int(float64(th.totWrite)*1.1)
				th.mux.Lock()
				th.end = begin - 1
				th.mux.Unlock()
				var end int
				if p.next != nil {
					end = p.next.Content().begin
				} else {
					end = mng.totSize - 1
				}
				mng.newThread(begin, end)
				cnt++
			} else {
				continue
			}
		}
	}
}

func (mng *Manager) sitrep(args ...any) {
	if mng.Res != nil {
		msg := arg2s(args)

		mng.Res <- Result{
			code: Running, 
			msg:  msg,
		}
	}
}

func (mng *Manager) abort(args ...any) {
	for p := mng.list.Front(); p != nil; p = p.Next() {
		th := p.Content()
		th.file.Close()
		os.Remove(th.file.Name())
	}

	if mng.Res != nil {
		msg := arg2s(args)

		mng.Res <- Result{
			code: Abort,
			msg:  msg,
		}
	}
}

func (mng *Manager) success(args ...any) {
	if mng.Res != nil {
		msg := arg2s(args)

		mng.Res <- Result{
			code: Success,
			msg:  msg,
		}
	} 
}

func (mng *Manager) initDownload() {
	step := int(mng.totSize / mng.MaxThread) + 1
	if step < 1 {
		step = 1
	}
	begin := 0
	end := begin + step - 1
	for {
		mng.newThread(begin, end)

		begin = end + 1
		if begin >= mng.totSize {
			break
		}
		end = begin + step - 1
		if end >= mng.totSize {
			end = mng.totSize
		}
	}
}

func (mng *Manager) newThread(begin int, end int) *thread {
	th := thread{
		mng:    mng,
		begin:  begin,
		end:    end,
		stat:   Result{code: Running, msg: ""},
		client: &http.Client{},
	}
	mng.mux.Lock()
	if mng.list.Empty() {
		mng.list.Append(&th)
		th.node = mng.list.tail
	} else {
		n := mng.list.Front()
		for n.Next() != nil && n.Content().begin < begin {
			n = n.Next()
		}
		if n.Content().begin >= begin {
			mng.list.InsNode(n, &th)
			th.node = n.prev
		} else {
			mng.list.Append(&th)
			th.node = mng.list.tail
		}
	}
	mng.runCnt++
	mng.mux.Unlock()
	go th.run()
	return &th
}
