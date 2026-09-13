package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	var localDir string
	var keepDays int
	flag.StringVar(&localDir, "local-dir", "/home/kredit/backups", "Local backup storage directory")
	flag.IntVar(&keepDays, "keep-days", 14, "Days to retain local backups")
	flag.Parse()

	endpoint := os.Getenv("OBJECT_STORAGE_ENDPOINT")
	bucket := os.Getenv("OBJECT_STORAGE_BUCKET")
	accessKey := os.Getenv("OBJECT_STORAGE_ACCESS_KEY")
	secretKey := os.Getenv("OBJECT_STORAGE_SECRET_KEY")
	region := os.Getenv("OBJECT_STORAGE_REGION")
	if region == "" {
		region = "auto"
	}

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		fmt.Fprintf(os.Stderr, "Error: OBJECT_STORAGE_ENDPOINT, OBJECT_STORAGE_BUCKET, OBJECT_STORAGE_ACCESS_KEY, and OBJECT_STORAGE_SECRET_KEY are required\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(localDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating local backup dir: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	timestamp := now.Format("20060102T150405Z")
	dumpFile := filepath.Join(localDir, fmt.Sprintf("kredit-%s.dump.gz", timestamp))

	fmt.Printf("[%s] Starting PostgreSQL database backup...\n", timestamp)

	// Execute pg_dump via docker
	cmd := exec.Command("docker", "exec", "kredit-prod-postgres-1", "pg_dump", "-U", "kredit", "-d", "kredit", "--format=custom")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening stdout pipe: %v\n", err)
		os.Exit(1)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting pg_dump: %v\n", err)
		os.Exit(1)
	}

	outFile, err := os.OpenFile(dumpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating dump file: %v\n", err)
		os.Exit(1)
	}

	hasher := sha256.New()
	multiWriter := io.MultiWriter(outFile, hasher)
	gzWriter := gzip.NewWriter(multiWriter)

	copied, err := io.Copy(gzWriter, stdout)
	if err != nil {
		_ = outFile.Close()
		fmt.Fprintf(os.Stderr, "Error copying dump data: %v\n", err)
		os.Exit(1)
	}
	if err := gzWriter.Close(); err != nil {
		_ = outFile.Close()
		fmt.Fprintf(os.Stderr, "Error closing gzip writer: %v\n", err)
		os.Exit(1)
	}
	if err := outFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "pg_dump command failed: %v\n", err)
		os.Exit(1)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	stat, _ := os.Stat(dumpFile)
	sizeBytes := stat.Size()
	fmt.Printf("[%s] Backup complete: %s (uncompressed: %d bytes, compressed: %d bytes, sha256: %s)\n",
		timestamp, dumpFile, copied, sizeBytes, checksum)

	// Write sha256 file
	_ = os.WriteFile(dumpFile+".sha256", []byte(fmt.Sprintf("%s  %s\n", checksum, filepath.Base(dumpFile))), 0600)

	// Upload to Cloudflare R2 if configured
	if strings.Contains(endpoint, "<") || strings.HasPrefix(accessKey, "your-") || endpoint == "https://r2.cloudflarestorage.com" {
		fmt.Printf("[%s] Cloudflare R2 not fully configured yet (needs Account ID and real token). Local backup is saved.\n", timestamp)
	} else {
		fmt.Printf("[%s] Uploading to Cloudflare R2 bucket '%s'...\n", timestamp, bucket)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		httpClient := &http.Client{Transport: tr}

		cfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithHTTPClient(httpClient),
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "AWS config error: %v\n", err)
		} else {
			s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(endpoint)
				o.UsePathStyle = true
			})

			r2Key := fmt.Sprintf("database-backups/kredit-%s.dump.gz", timestamp)
			uploadData, err := os.Open(dumpFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening backup for upload: %v\n", err)
			} else {
				defer uploadData.Close()
				_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
					Bucket:        aws.String(bucket),
					Key:           aws.String(r2Key),
					Body:          uploadData,
					ContentLength: aws.Int64(sizeBytes),
					ContentType:   aws.String("application/gzip"),
					Metadata: map[string]string{
						"sha256":      checksum,
						"created-at":  timestamp,
						"environment": "production",
					},
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "R2 upload warning: %v\n", err)
				} else {
					fmt.Printf("[%s] Successfully uploaded to R2: %s\n", timestamp, r2Key)
					checksumKey := r2Key + ".sha256"
					_, _ = s3Client.PutObject(ctx, &s3.PutObjectInput{
						Bucket:      aws.String(bucket),
						Key:         aws.String(checksumKey),
						Body:        bytes.NewReader([]byte(fmt.Sprintf("%s  %s\n", checksum, filepath.Base(dumpFile)))),
						ContentType: aws.String("text/plain"),
					})
				}
			}
		}
	}

	// Prune local backups older than keepDays
	cutoff := now.AddDate(0, 0, -keepDays)
	entries, _ := os.ReadDir(localDir)
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil && !info.IsDir() && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(localDir, entry.Name()))
		}
	}

	fmt.Printf("[%s] Backup and prune completed successfully.\n", timestamp)
}
