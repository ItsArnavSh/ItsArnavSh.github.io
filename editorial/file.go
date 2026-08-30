package main

import (
	"fmt"
	"log"
	"os"
)

func read_file(s string) ([]byte, error) {
	data, err := os.ReadFile(sanitize_path(s))
	if err != nil {
		log.Fatal(err)
	}
	return data, err
}

func sanitize_path(file string) string {
	return fmt.Sprintf("../draft/%s", file)
}

func verify_file(file string) bool {
	path := sanitize_path(file)

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return true
}
