package main

import "testing"

func TestExtractUpSQLStopsBeforeDownSection(t *testing.T) {
	input := []byte("-- +goose Up\nCREATE TABLE example (id INT);\n-- +goose Down\nDROP TABLE example;\n")
	got := extractUpSQL(input)
	want := "-- +goose Up\nCREATE TABLE example (id INT);\n"
	if got != want {
		t.Fatalf("extractUpSQL() = %q, want %q", got, want)
	}
}
