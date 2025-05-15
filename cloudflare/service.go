package cloudflare

import (
	"context"
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
