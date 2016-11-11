package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Response from Snappy server"))
	})

	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
