package main

import (
	"bytes"
	"context"
	"editorial/templates"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fileFlag := flag.String("f", "", "watch and serve a markdown file")
	compileFlag := flag.String("c", "", "compile a markdown file to static html in ../blog/")
	flag.Parse()

	switch {
	case *compileFlag != "":
		if !verify_file(*compileFlag) {
			log.Fatalf("File not valid")
		}
		if err := compileFile(*compileFlag); err != nil {
			log.Fatalf("compile error: %v", err)
		}

	case *fileFlag != "":
		if !verify_file(*fileFlag) {
			log.Fatalf("File not valid")
		}
		go watch_file(*fileFlag)
		serve()

	default:
		fmt.Println("Usage: editorial -f <filename> | -c <filename>")
		os.Exit(1)
	}
}

func compileFile(file string) error {
	md, err := read_file(file)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	html, err := renderMarkdown(md)
	if err != nil {
		return fmt.Errorf("error rendering markdown: %w", err)
	}

	var buf bytes.Buffer
	if err := templates.PostPage(file, html).Render(context.Background(), &buf); err != nil {
		return fmt.Errorf("error rendering template: %w", err)
	}

	outName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) + ".html"
	outPath := filepath.Join("..", "blog", outName)

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("error creating output dir: %w", err)
	}

	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	log.Printf("compiled %s -> %s", file, outPath)
	return nil
}
