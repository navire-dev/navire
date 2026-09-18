package register_dispatcher

import (
	"context"

	"github.com/navire-dev/navire/core/internal/api/deps"
	"github.com/navire-dev/navire/core/internal/dispatchers/registration"
)

type Service struct {
	registration registration.Service
}

func NewService(d deps.Deps) *Service {
	return &Service{registration: registration.Service{Redis: d.Redis, Secret: d.RegistrationSecret}}
}

func (s *Service) Execute(ctx context.Context, request Request) (Response, error) {
	return s.registration.Register(ctx, request)
}
