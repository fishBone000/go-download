package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func getContentRange(header *http.Header) (string, int, int, int, bool) {
	s, ok := (*header)["Content-Range"]
	var unit string
	var begin int
	var end int
	var size int
	if !ok || len(s) != 1 {
		return "", 0, 0, 0, false
	}

	var sRnSz string
	fmt.Sscanf(s[0], "%s %s", &unit, &sRnSz)
	if unit != "bytes" {
		return "", 0, 0, 0, false
	}
	sRn, sSz, found := strings.Cut(sRnSz, "/")
	if !found {
		return "", 0, 0, 0, false
	}

	if sSz == "*" {
		size = -1
	} else {
		var err error
		size, err = strconv.Atoi(sSz)
		if err != nil {
			return "", 0, 0, 0, false
		}
	}

	if sRn == "*" {
		begin = -1
		end = -1
	} else {
		sBegin, sEnd, found := strings.Cut(sRn, "-")
		if !found {
			return "", 0, 0, 0, false
		}
		var err1 error
		var err2 error
		begin, err1 = strconv.Atoi(sBegin)
		end, err2 = strconv.Atoi(sEnd)
		if err1 != nil || err2 != nil {
			return "", 0, 0, 0, false
		}
		if begin > end {
			return "", 0, 0, 0, false
		}
	}

	return unit, begin, end, size, true
}

func arg2s(args ...any) string {
	msg := ""
	for _, arg := range args {
		msg += fmt.Sprintf("%v", arg)
	}
	return msg
}
