package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupDestinationRejectsUntrustedOrigins(t *testing.T) {
	account := strings.Repeat("a", 32)
	for _, jurisdiction := range []string{"", ".eu", ".us", ".fedramp"} {
		if err := validateBackupDestination("https://" + account + jurisdiction + ".r2.cloudflarestorage.com"); err != nil {
			t.Fatal(err)
		}
	}
	for _, endpoint := range []string{"http://" + account + ".r2.cloudflarestorage.com", "https://example.test", "https://r2.cloudflarestorage.com", "https://" + account + ".r2.cloudflarestorage.com.example.test", "https://user@" + account + ".r2.cloudflarestorage.com", "https://" + account + ".r2.cloudflarestorage.com/path", "https://" + account + ".r2.cloudflarestorage.com?token=x", "https://" + account + ".r2.cloudflarestorage.com:444"} {
		if err := validateBackupDestination(endpoint); err == nil {
			t.Errorf("untrusted backup origin accepted: %s", endpoint)
		}
	}
	for _, input := range []string{"-Uroot", "db?host=example.test", "postgres://other", "name;command", "name with space"} {
		if backupDatabaseIdentifier.MatchString(input) {
			t.Errorf("unsafe database identifier: %s", input)
		}
	}
}

func TestBackupRootRejectsSymlinkAndSharedDirectory(t *testing.T) {
	base := t.TempDir()
	directory := filepath.Join(base, "private")
	root, err := openBackupRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = root.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(directory, filepath.Join(base, "link")); err != nil {
		t.Fatal(err)
	}
	if r, err := openBackupRoot(filepath.Join(base, "link")); err == nil {
		_ = r.Close()
		t.Fatal("symlink root accepted")
	}
	if err = os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if r, err := openBackupRoot(directory); err == nil {
		_ = r.Close()
		t.Fatal("world-readable database backup directory accepted")
	}
}

func TestBackupCaptureBindsUploadedDescriptorAndChecksum(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	root, err := openBackupRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	name := "kredit-20260920T120000Z.dump.gz"
	original := []byte("PGDMP synthetic archive data")
	archive, err := captureArchive(root, name, func(w io.Writer) error { _, err := w.Write(original); return err })
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archive.file.Close() }()
	data, err := io.ReadAll(archive.file)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	if archive.checksum != hex.EncodeToString(h[:]) || archive.size != int64(len(data)) {
		t.Fatal("archive metadata does not describe actual bytes")
	}
	z, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, original) {
		t.Fatal("dump data changed")
	}
	sidecar, err := root.ReadFile(name + ".sha256")
	if err != nil || string(sidecar) != archive.sidecar() {
		t.Fatal("checksum companion missing", err)
	}
	if err = root.Rename(name, "old-archive"); err != nil {
		t.Fatal(err)
	}
	if err = root.WriteFile(name, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = archive.file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	bound, err := io.ReadAll(archive.file)
	if err != nil || !bytes.Equal(data, bound) {
		t.Fatal("upload descriptor followed replacement pathname", err)
	}
}

func TestBackupCaptureFailsBeforeProducerOnCollisionAndCleansPartial(t *testing.T) {
	root, err := openBackupRoot(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	name := "kredit-20260920T120000Z.dump.gz"
	if err = root.WriteFile(name+".sha256", []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = captureArchive(root, name, func(io.Writer) error { called = true; return nil })
	if err == nil || called {
		t.Fatal("producer ran despite preexisting companion")
	}
	if _, err = root.Stat(name); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed capture left a purported archive")
	}
	original, err := root.ReadFile(name + ".sha256")
	if err != nil || string(original) != "preserve" {
		t.Fatal("preexisting companion altered")
	}
	if err = root.Remove(name + ".sha256"); err != nil {
		t.Fatal(err)
	}
	for _, fail := range []bool{false, true} {
		_, err = captureArchive(root, name, func(w io.Writer) error {
			if fail {
				_, e := w.Write([]byte("partial"))
				return errors.Join(e, errors.New("synthetic producer failure"))
			}
			return nil
		})
		if err == nil {
			t.Fatal("empty or failed dump accepted")
		}
		for _, path := range []string{name, name + ".sha256"} {
			if _, err = root.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("partial backup retained as complete", path)
			}
		}
	}
}

func TestBackupRetentionOnlyRemovesRecognizedPairs(t *testing.T) {
	root, err := openBackupRoot(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	at := time.Now().UTC()
	old := at.Add(-30 * 24 * time.Hour)
	pair := "kredit-20260801T120000Z.dump.gz"
	orphan := "kredit-20260802T120000Z.dump.gz"
	link := "kredit-20260803T120000Z.dump.gz"
	for _, name := range []string{pair, pair + ".sha256", orphan, "other.dump.gz", "target"} {
		if err = root.WriteFile(name, []byte("synthetic"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = root.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err = root.Symlink("target", link); err != nil {
		t.Fatal(err)
	}
	if err = root.WriteFile(link+".sha256", []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = pruneBackups(root, at.Add(-14*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{pair, pair + ".sha256"} {
		if _, err = root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("recognized expired pair not removed")
		}
	}
	for _, name := range []string{orphan, "other.dump.gz", "target", link, link + ".sha256"} {
		if _, err = root.Lstat(name); err != nil {
			t.Fatal("unrecognized or symlink entry removed", name, err)
		}
	}
}
