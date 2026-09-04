package documents

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ObjectStore is compatible with AWS S3 and MinIO. Credentials are supplied
// by deployment configuration and never persisted in document metadata.
type S3ObjectStore struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

func NewS3ObjectStore(ctx context.Context, endpoint, region, accessKey, secretKey, bucket string) (*S3ObjectStore, error) {
	if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, errors.New("object storage endpoint, region, credentials, and bucket are required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	return &S3ObjectStore{client: client, presign: s3.NewPresignClient(client), bucket: bucket}, nil
}

func (s *S3ObjectStore) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if s == nil || s.client == nil {
		return errors.New("object storage is not configured")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body, ContentLength: aws.Int64(size), ContentType: aws.String(contentType), ServerSideEncryption: "AES256"})
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
	request, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), IfNoneMatch: aws.String("*"), ContentType: aws.String(contentType), ServerSideEncryption: "AES256"}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", err
	}
	return request.URL, nil
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
