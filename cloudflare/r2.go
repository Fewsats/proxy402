package cloudflare

import (
	"context" // Keep context for method signature consistency if desired
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type R2Service struct {
	r2           *s3.S3
	bucket       string
	publicBucket string
}

func NewR2Service(cfg *Config) (*R2Service, error) {
	cred := credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretAccessKey, "")

	r2Config := &aws.Config{
		Credentials:      cred,
		Endpoint:         aws.String(cfg.Endpoint),
		Region:           aws.String("auto"),
		S3ForcePathStyle: aws.Bool(true),
	}

	sess, err := session.NewSession(r2Config)
	if err != nil {
		return nil, err
	}

	r2 := s3.New(sess)

	return &R2Service{
		r2:           r2,
		bucket:       cfg.BucketName,
		publicBucket: cfg.PublicBucketName,
	}, nil
}

func (r *R2Service) PresignUploadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, _ := r.r2.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	// Add WithContext if you want to respect cancellation, though Presign itself is synchronous
	// req.SetContext(ctx)
	return req.Presign(expires)
}

func (r *R2Service) PresignDownloadURL(ctx context.Context, key string, expires time.Duration, originalFilename string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}
	// setting the disposition is used to set the filename
	// in the browser download dialog
	if originalFilename != "" {
		disposition := fmt.Sprintf("attachment; filename=\"%s\"", originalFilename)
		input.ResponseContentDisposition = aws.String(disposition)
	}

	req, _ := r.r2.GetObjectRequest(input)
	return req.Presign(expires)
}

func (r *R2Service) uploadPublicFile(ctx context.Context, key string, reader io.ReadSeeker) (string, error) {
	_, err := r.r2.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.publicBucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return "", err
	}
	return r.publicFileURL(key), nil
}

func (r *R2Service) deletePublicFile(ctx context.Context, key string) error {
	_, err := r.r2.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.publicBucket),
		Key:    aws.String(key),
	})
	return err
}

func (r *R2Service) publicFileURL(key string) string {
	// TODO(pol) this is a dev access to staging bucket hardcoded
	return fmt.Sprintf("https://pub-28923997f1a14d3a836f2f0cdfc5a4a3.r2.dev/%s", key)
}
