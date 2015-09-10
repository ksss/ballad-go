package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	concurrent = flag.Int("j", runtime.NumCPU(), "number of concurrent")
	timeout    = flag.Int64("timeout", 10, "timeout second for HTTP request")
)

var (
	errRedirectNotFollowed = errors.New("redirection not followed")
)

var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return errRedirectNotFollowed
	},
	Timeout: time.Duration(*timeout * int64(time.Second)),
}

type responseSet struct {
	res    *http.Response
	urlStr string
}

// Run command for ballad
// goroutine design
// | input | -> | pool | -> |        | -> print
//     +                    | result |
//     +------------------> |        |
func main() {
	flag.Parse()
	in2pool := make(chan string, *concurrent)
	in2result := make(chan string, *concurrent)
	pool2result := make(chan *responseSet)
	deadPool := make(chan bool)
	quit := make(chan bool)
	var inCount int32
	var outCount int32
	eof := false

	for i := 0; i < *concurrent; i++ {
		go func() {
		L:
			for {
				select {
				case <-deadPool:
					break L
				case urlStr := <-in2pool:
					res, _ := fetch(urlStr)
					if res == nil {
						// request failed
					} else {
						// request failed
						defer res.Body.Close()
					}
					pool2result <- &responseSet{
						res:    res,
						urlStr: urlStr,
					}
				}
			}
		}()
	}

	go func() {
		var stock []*responseSet
	L:
		for {
			input := <-in2result
			found := false
			for i, set := range stock {
				if set.urlStr == input {
					found = true
					fmt.Fprintf(os.Stdout, "%s\t%s\n", edit(set.res), set.urlStr)
					atomic.AddInt32(&outCount, 1)
					stock = append(stock[:i], stock[i+1:]...)
					if eof && inCount == outCount {
						// broadcast
						close(deadPool)
						quit <- true
						break L
					}
				}
			}

			if found {
				continue
			}
			if eof && inCount == outCount {
				// broadcast
				close(deadPool)
				quit <- true
				break L
			}

			for {
				set := <-pool2result
				if set.urlStr == input {
					fmt.Fprintf(os.Stdout, "%s\t%s\n", edit(set.res), set.urlStr)
					atomic.AddInt32(&outCount, 1)
					if eof && inCount == outCount {
						// broadcast
						close(deadPool)
						quit <- true
						break L
					}
					break
				} else {
					stock = append(stock, set)
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			atomic.AddInt32(&inCount, 1)
			urlStr := strings.TrimRight(scanner.Text(), "\n")
			in2pool <- urlStr
			in2result <- urlStr
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
		eof = true
	}()

	<-quit
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
