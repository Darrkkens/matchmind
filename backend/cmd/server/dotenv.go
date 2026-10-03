package main

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv reads KEY=VALUE lines from the first .env found in the working
// directory or its parent (the repository root when running from backend/).
// Variables already set in the environment win, so `FOO=x go run` still works.
func loadDotEnv() string {
	for _, path := range []string{".env", "../.env"} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
			key = strings.TrimSpace(key)
			if !ok || key == "" || strings.ContainsAny(key, " \t") {
				continue
			}
			value = strings.TrimSpace(value)
			if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
				value = value[1 : len(value)-1]
			}
			if _, set := os.LookupEnv(key); !set {
				_ = os.Setenv(key, value)
			}
		}
		return path
	}
	return ""
}
