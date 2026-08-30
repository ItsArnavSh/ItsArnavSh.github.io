package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

var currentHTML atomic.Value
var reloadChan = make(chan struct{})

func serve() {
	currentHTML.Store("<html><body>loading...</body></html>")

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := currentHTML.Load().(string)

		// inject a tiny script that listens for reload signals
		script := `
<script>
	const es = new EventSource("/events");
	es.onmessage = () => location.reload();
</script>`

		w.Write([]byte(html + script))
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		for {
			select {
			case <-reloadChan:
				fmt.Fprintf(w, "data: reload\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	fs := http.FileServer(http.Dir("../static/images"))
	mux.Handle("/images/", http.StripPrefix("/images/", fs))

	log.Println("serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func updateHTML(newHTML string) {
	currentHTML.Store(newHTML)
	select {
	case reloadChan <- struct{}{}:
	default:
	}
}
