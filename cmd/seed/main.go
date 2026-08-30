package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://kredit:kredit@localhost:5432/kredit?sslmode=disable"
	}
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	seed := filepath.Join(root, "db", "seeds", "001_demo.sql")
	if _, err := os.Stat(seed); err != nil {
		panic(err)
	}
	fmt.Printf("loading %s\n", seed)
	command := exec.Command("psql", databaseURL, "-v", "ON_ERROR_STOP=1", "-f", seed)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		panic(err)
	}
}
