package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, "{ \"hello\": \"world\" }")
	})
	http.ListenAndServe(":80", nil)
}
