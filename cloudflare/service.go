package cloudflare

import (
	"context"
	"fmt"
	"io"
	"time"
)

type Service struct {
	r2 *R2Service
}

func NewService(cfg *Config) (*Service, error) {
	r2, err := NewR2Service(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{r2: r2}, nil
}

func (s *Service) GetUploadURL(ctx context.Context, key string) (string, error) {
	// e.g. 2 hours
	return s.r2.PresignUploadURL(ctx, key, 2*time.Hour)
}

func (s *Service) GetDownloadURL(ctx context.Context, key string, originalFilename string) (string, error) {
	// e.g. 24 hours
	return s.r2.PresignDownloadURL(ctx, key, 24*time.Hour, originalFilename)
}

// PublicFileURL returns the URL of a file in the storage provider.
func (s *Service) PublicFileURL(key string) string {
	return s.r2.publicFileURL(key)
}

func (s *Service) UploadPublicFile(ctx context.Context, fileID string,
	prefix string, reader io.ReadSeeker) (string, error) {

	key := fmt.Sprintf("%s/%s", prefix, fileID)
	return s.r2.uploadPublicFile(ctx, key, reader)
}

func (s *Service) DeletePublicFile(ctx context.Context, key string) error {
	err := s.r2.deletePublicFile(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}
	return nil
}
