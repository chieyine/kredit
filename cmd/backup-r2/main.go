package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	var directory string
	var keepDays int
	flag.StringVar(&directory, "local-dir", "/home/kredit/backups", "Private local backup directory (0700)")
	flag.IntVar(&keepDays, "keep-days", 14, "Days to retain local backups after successful replication")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := runBackup(ctx, directory, keepDays)
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
		os.Exit(1)
	}
}

func runBackup(ctx context.Context, directory string, keepDays int) (err error) {
	if keepDays <= 0 || keepDays > 36500 {
		return errors.New("keep-days must be between 1 and 36500")
	}
	endpoint, bucket := os.Getenv("OBJECT_STORAGE_ENDPOINT"), os.Getenv("OBJECT_STORAGE_BUCKET")
	accessKey, secretKey := os.Getenv("OBJECT_STORAGE_ACCESS_KEY"), os.Getenv("OBJECT_STORAGE_SECRET_KEY")
	if err = validateBackupDestination(endpoint); err != nil {
		return err
	}
	for _, value := range []string{bucket, accessKey, secretKey} {
		if strings.TrimSpace(value) == "" || strings.Contains(value, "<") || strings.HasPrefix(value, "your-") {
			return errors.New("complete non-placeholder R2 bucket and credentials are required")
		}
	}
	container := envDefault("BACKUP_POSTGRES_CONTAINER", "kredit-prod-postgres-1")
	user := envDefault("BACKUP_POSTGRES_USER", "kredit")
	database := envDefault("BACKUP_POSTGRES_DB", "kredit")
	if !backupContainer.MatchString(container) || !backupDatabaseIdentifier.MatchString(user) || !backupDatabaseIdentifier.MatchString(database) {
		return errors.New("invalid PostgreSQL container, database or user identifier")
	}
	root, err := openBackupRoot(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	now := time.Now().UTC()
	timestamp := now.Format("20060102T150405Z")
	name := "kredit-" + timestamp + ".dump.gz"
	dumpCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	archive, err := captureArchive(root, name, func(writer io.Writer) error {
		// Fixed executable and argument positions; identifiers are restricted above.
		// No shell, free-form flags or connection strings are accepted.
		cmd := exec.CommandContext(dumpCtx, "docker", "exec", container, "pg_dump", "-U", user, "-d", database, "--format=custom") // #nosec G204 G702 -- validated identifiers; fixed docker/pg_dump argument vector, no shell.
		cmd.WaitDelay = 5 * time.Second
		cmd.Stdout, cmd.Stderr = writer, os.Stderr
		return cmd.Run()
	})
	if err != nil {
		return fmt.Errorf("capture database dump: %w", err)
	}
	defer func() { err = errors.Join(err, archive.file.Close()) }()
	fmt.Printf("[%s] Local backup captured: %s (%d bytes, sha256: %s)\n", timestamp, name, archive.size, archive.checksum)

	uploadCtx, cancelUpload := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelUpload()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	config, err := awsconfig.LoadDefaultConfig(uploadCtx,
		awsconfig.WithHTTPClient(client), awsconfig.WithRegion(envDefault("OBJECT_STORAGE_REGION", "auto")),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		return errors.New("R2 configuration failed; local backup retained")
	}
	store := s3.NewFromConfig(config, func(o *s3.Options) { o.BaseEndpoint = aws.String(endpoint); o.UsePathStyle = true })
	key := "database-backups/" + name
	if _, err = store.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key), Body: archive.file,
		ContentLength: aws.Int64(archive.size), ContentType: aws.String("application/gzip"), IfNoneMatch: aws.String("*"),
		Metadata: map[string]string{"sha256": archive.checksum, "created-at": timestamp, "environment": "production"},
	}); err != nil {
		return errors.New("R2 archive replication failed; local backup retained and retention skipped")
	}
	if _, err = store.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket: aws.String(bucket), Key: aws.String(key + ".sha256"), Body: strings.NewReader(archive.sidecar()),
		ContentType: aws.String("text/plain"), IfNoneMatch: aws.String("*"),
	}); err != nil {
		return errors.New("R2 checksum replication failed; local backup retained and retention skipped")
	}
	if err = pruneBackups(root, now.AddDate(0, 0, -keepDays)); err != nil {
		return err
	}
	fmt.Printf("[%s] Backup replication and retention completed.\n", timestamp)
	return nil
}
