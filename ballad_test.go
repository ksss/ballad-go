package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var robotsTxtHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Last-Modified", "sometime")
	fmt.Fprintf(w, "User-agent: go\nDisallow: /something/")
})

func BenchmarkSingleFetch(b *testing.B) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()
	for i := 0; i < b.N; i++ {
		fetch(ts.URL)
	}
}
func BenchmarkParallelFetch(b *testing.B) {
	ts := httptest.NewServer(robotsTxtHandler)
	defer ts.Close()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fetch(ts.URL)
		}
	})
}
