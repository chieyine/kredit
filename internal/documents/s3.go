package documents

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3ObjectStore is compatible with AWS S3 and MinIO. Credentials are supplied
// by deployment configuration and never persisted in document metadata.
type S3ObjectStore struct {
	client     *s3.Client
	presign    *s3.PresignClient
	bucket     string
	encryption types.ServerSideEncryption
}

func NewS3ObjectStore(ctx context.Context, endpoint, region, accessKey, secretKey, bucket string) (*S3ObjectStore, error) {
	if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, errors.New("object storage endpoint, region, credentials, and bucket are required")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("object storage endpoint must be an HTTP or HTTPS URL")
	}
	encryption := types.ServerSideEncryptionAes256
	// R2 encrypts objects at rest but does not accept S3's SSE-S3 request header.
	if strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".r2.cloudflarestorage.com") {
		encryption = ""
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	return &S3ObjectStore{client: client, presign: s3.NewPresignClient(client), bucket: bucket, encryption: encryption}, nil
}

func (s *S3ObjectStore) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if s == nil || s.client == nil {
		return errors.New("object storage is not configured")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body, IfNoneMatch: aws.String("*"), ContentLength: aws.Int64(size), ContentType: aws.String(contentType), ServerSideEncryption: s.encryption})
	return err
}

func (s *S3ObjectStore) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s == nil || s.presign == nil {
		return "", errors.New("object storage is not configured")
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return "", errors.New("signed URL TTL must be between 1 second and 24 hours")
	}
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (s *S3ObjectStore) SignedUploadURL(ctx context.Context, key string, ttl time.Duration, contentType string) (string, error) {
	if s == nil || s.presign == nil {
		return "", errors.New("object storage is not configured")
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return "", errors.New("signed URL TTL must be between 1 second and 24 hours")
	}
	request, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), IfNoneMatch: aws.String("*"), ContentType: aws.String(contentType), ServerSideEncryption: s.encryption}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (s *S3ObjectStore) UploadHeaders(contentType string) map[string]string {
	headers := map[string]string{"If-None-Match": "*", "Content-Type": contentType}
	if s.encryption != "" {
		headers["x-amz-server-side-encryption"] = string(s.encryption)
	}
	return headers
}

func (s *S3ObjectStore) Head(ctx context.Context, key string) (int64, string, error) {
	if s == nil || s.client == nil {
		return 0, "", errors.New("object storage is not configured")
	}
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return 0, "", err
	}
	if result.ContentLength == nil || result.ContentType == nil {
		return 0, "", errors.New("object storage returned incomplete metadata")
	}
	return *result.ContentLength, *result.ContentType, nil
}

// Open reads private object bytes directly so completion records a checksum of
// the stored content rather than trusting a client-provided digest or ETag.
func (s *S3ObjectStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("object storage is not configured")
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (s *S3ObjectStore) ListObjects(ctx context.Context, cursor string) ([]ObjectCandidate, string, error) {
	if s == nil || s.client == nil {
		return nil, "", errors.New("object storage is not configured")
	}
	input := &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), MaxKeys: aws.Int32(100)}
	if cursor != "" {
		input.ContinuationToken = aws.String(cursor)
	}
	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, "", err
	}
	objects := []ObjectCandidate{}
	for _, item := range result.Contents {
		if item.Key != nil && item.LastModified != nil {
			objects = append(objects, ObjectCandidate{Key: *item.Key, ModifiedAt: *item.LastModified})
		}
	}
	return objects, aws.ToString(result.NextContinuationToken), nil
}
func (s *S3ObjectStore) DeleteObject(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return errors.New("object storage is not configured")
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}
