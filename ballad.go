package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	concurrent = flag.Int("j", runtime.NumCPU(), "number of concurrent")
	timeout    = flag.Int64("timeout", 10, "timeout second for HTTP request")
)

var (
	errRedirectNotFollowed = errors.New("redirection not followed")
)

var customTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	Dial: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial,
	TLSHandshakeTimeout: 10 * time.Second,
	MaxIdleConnsPerHost: 3 * http.DefaultMaxIdleConnsPerHost,
}

var httpClient = &http.Client{
	Transport: customTransport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return errRedirectNotFollowed
	},
	Timeout: time.Duration(*timeout * int64(time.Second)),
}

type responseSet struct {
	res    *http.Response
	urlStr string
}

func main() {
	flag.Parse()

	var stock []*responseSet
	var m sync.Mutex
	printq := make(chan string, *concurrent)
	stockq := make(chan bool, *concurrent)
	waitingq := make(chan bool, *concurrent)
	quitq := make(chan bool)
	eof := false

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urlStr := strings.TrimRight(scanner.Text(), "\n")
			printq <- urlStr

			go func() {
				waitingq <- true
				res, _ := fetch(urlStr)
				m.Lock()
				stock = append(stock, &responseSet{res: res, urlStr: urlStr})
				m.Unlock()
				<-waitingq
				stockq <- true
			}()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
		eof = true
	}()

	go func() {
		for {
			urlStr := <-printq
		L:
			for {
				for i, set := range stock {
					if set.urlStr == urlStr {
						fmt.Fprintf(os.Stdout, "%s\t%s\n", edit(set.res), set.urlStr)
						m.Lock()
						stock = append(stock[:i], stock[i+1:]...)
						m.Unlock()
						if eof && len(waitingq) == 0 && len(stock) == 0 {
							quitq <- true
						}
						break L
					}
				}
				<-stockq
			}
		}
	}()

	<-quitq
}

func fetch(urlStr string) (*http.Response, error) {
	res, err := httpClient.Head(urlStr)
	if err != nil {
		ue, ok := err.(*url.Error)
		if ok {
			if ue.Err == errRedirectNotFollowed {
				// ignore
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return res, nil
}

func edit(res *http.Response) string {
	if res == nil {
		return "???"
	}
	// TODO Implement other style
	return strconv.Itoa(res.StatusCode)
}
