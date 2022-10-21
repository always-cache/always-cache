package main

import (
  "fmt"
	"net/http"
	"time"

	"github.com/ericselin/always-cache"
)

func main() {
	acache := cache.New(cache.Config{
		DefaultMaxAge: 5 * time.Minute,
	})

  handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %q", r.URL.Path)
  })

	http.ListenAndServe(":8080", acache.Middleware(handler))
}
