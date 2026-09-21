package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var r2Host = regexp.MustCompile(`^[a-f0-9]{32}(\.(eu|us|fedramp))?\.r2\.cloudflarestorage\.com$`)
var backupContainer = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
var backupDatabaseIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,62}$`)
var backupName = regexp.MustCompile(`^kredit-[0-9]{8}T[0-9]{6}Z\.dump\.gz$`)

func validateBackupDestination(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") || u.Port() != "" || !r2Host.MatchString(u.Host) {
		return errors.New("backup endpoint must be the account-specific HTTPS Cloudflare R2 origin")
	}
	return nil
}

// The directory must be private before any database bytes are produced. Root
// pins subsequent operations to that directory even if its path is renamed.
func openBackupRoot(directory string) (*os.Root, error) {
	if !filepath.IsAbs(directory) {
		return nil, errors.New("backup directory must be absolute")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	before, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode().Perm()&0077 != 0 {
		return nil, errors.New("backup directory must be a real, owner-only directory (0700)")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	after, err := root.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		_ = root.Close()
		return nil, errors.New("backup directory changed while opening")
	}
	return root, nil
}

type backupArchive struct {
	file     *os.File
	name     string
	checksum string
	size     int64
}

func (a *backupArchive) sidecar() string { return a.checksum + "  " + a.name + "\n" }

// Both paths are exclusively created before starting the producer. The same
// archive descriptor is subsequently uploaded, not a replaceable pathname.
func captureArchive(root *os.Root, name string, produce func(io.Writer) error) (archive *backupArchive, err error) {
	if root == nil || !backupName.MatchString(name) || produce == nil {
		return nil, errors.New("invalid backup capture")
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	complete, sidecarCreated := false, false
	defer func() {
		if !complete {
			err = errors.Join(err, file.Close(), root.Remove(name))
			if sidecarCreated {
				err = errors.Join(err, root.Remove(name+".sha256"))
			}
		}
	}()
	companion, err := root.OpenFile(name+".sha256", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	sidecarCreated = true
	companionClosed := false
	defer func() {
		if !companionClosed {
			err = errors.Join(err, companion.Close())
		}
	}()
	hash := sha256.New()
	compressed := gzip.NewWriter(io.MultiWriter(file, hash))
	counter := &countWriter{writer: compressed}
	produceErr := produce(counter)
	if err = errors.Join(produceErr, compressed.Close()); err != nil {
		return nil, err
	}
	if counter.bytes == 0 {
		return nil, errors.New("database dump was empty")
	}
	if err = file.Sync(); err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	result := &backupArchive{file, name, hex.EncodeToString(hash.Sum(nil)), info.Size()}
	if _, err = io.WriteString(companion, result.sidecar()); err != nil {
		return nil, err
	}
	if err = companion.Sync(); err != nil {
		return nil, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	err = companion.Close()
	companionClosed = true
	if err != nil {
		return nil, err
	}
	directory, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	if err = errors.Join(directory.Sync(), directory.Close()); err != nil {
		return nil, err
	}
	complete = true
	return result, nil
}

type countWriter struct {
	writer io.Writer
	bytes  int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.bytes += int64(n)
	return n, err
}

// A completed offsite upload is a precondition imposed by runBackup. Retention
// only removes recognized regular archives with regular checksum companions.
func pruneBackups(root *os.Root, cutoff time.Time) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !backupName.MatchString(name) {
			continue
		}
		archive, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if !archive.Mode().IsRegular() || !archive.ModTime().Before(cutoff) {
			continue
		}
		companion, err := root.Lstat(name + ".sha256")
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !companion.Mode().IsRegular() {
			continue
		}
		for _, path := range []string{name, name + ".sha256"} {
			if err = root.Remove(path); err != nil {
				return fmt.Errorf("prune recognized backup: %w", err)
			}
		}
	}
	return nil
}

func envDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
