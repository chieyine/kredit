package main

import "testing"

func TestDatabaseURLFromEnv(t *testing.T) {
	t.Run("requires a configured URL", func(t *testing.T) {
		if _, err := databaseURLFromEnv(func(string) string { return "  " }); err == nil {
			t.Fatal("expected an error for a missing DATABASE_URL")
		}
	})

	t.Run("returns a trimmed worker URL", func(t *testing.T) {
		const want = "postgres://kredit@localhost/kredit"
		got, err := databaseURLFromEnv(func(key string) string {
			if key != "RIVER_DATABASE_URL" {
				t.Fatalf("unexpected environment key %q", key)
			}
			return "  " + want + "  "
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("databaseURLFromEnv() = %q, want %q", got, want)
		}
	})
}
