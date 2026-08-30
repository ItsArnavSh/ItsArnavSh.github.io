package main

import (
	"bytes"
	"context"
	"editorial/templates"
	"log"

	"github.com/fsnotify/fsnotify"
)

func renderFile(file string) {
	md, err := read_file(file)
	if err != nil {
		log.Println("error reading file:", err)
		return
	}
	log.Printf("read %d bytes from %s", len(md), file)

	html, err := renderMarkdown(md)
	if err != nil {
		log.Println("error rendering markdown:", err)
		return
	}
	log.Printf("rendered markdown -> %d bytes of html", len(html))

	var buf bytes.Buffer
	err = templates.PostPage(file, html).Render(context.Background(), &buf)
	if err != nil {
		log.Println("error rendering template:", err)
		return
	}
	updateHTML(buf.String())
	log.Println("html updated, serving latest version")
}

func watch_file(file string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// initial render before watching for changes
	renderFile(file)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("change detected:", file)
					renderFile(file)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()

	err = watcher.Add(sanitize_path(file))
	if err != nil {
		log.Fatal(err)
	}
	select {} // block forever
}
