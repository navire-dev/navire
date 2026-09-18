package metrics

import (
	"context"
	"fmt"

	coremetrics "github.com/navire-dev/navire/core/internal/metrics"
)

type Service struct {
	exporter coremetrics.Exporter
}

func NewService(exporter coremetrics.Exporter) *Service {
	return &Service{exporter: exporter}
}

func (s *Service) Scrape(ctx context.Context) (ScrapeResponse, error) {
	if s.exporter == nil {
		return ScrapeResponse{}, fmt.Errorf("metrics exporter is not configured")
	}
	payload, err := s.exporter.Scrape(ctx)
	if err != nil {
		return ScrapeResponse{}, err
	}
	return ScrapeResponse{Body: payload.Body, ContentType: payload.ContentType}, nil
}
