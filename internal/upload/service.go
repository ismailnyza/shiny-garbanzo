package upload

import (
	"github.com/ismael/qr-restaurant/internal/shared/config"
	"github.com/ismael/qr-restaurant/pkg/storage"
)

type Service struct {
	storageClient *storage.Client
	cfg           *config.Config
}

func NewService(storageClient *storage.Client, cfg *config.Config) *Service {
	return &Service{storageClient: storageClient, cfg: cfg}
}
