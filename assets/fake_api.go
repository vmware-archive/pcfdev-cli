package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"Running"}`))
	})

	if err := http.ListenAndServe("127.0.0.1:8090", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
