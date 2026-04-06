package service

import (
	"context"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type TariffService struct {
	tariffRepo repository.TariffRepository
}

func NewTariffService(tariffRepo repository.TariffRepository) *TariffService {
	return &TariffService{tariffRepo}
}

func (s *TariffService) List(ctx context.Context) ([]models.Tariff, error) {
	return s.tariffRepo.List(ctx)
}

func (s *TariffService) Create(ctx context.Context, input models.CreateTariffInput) (*models.Tariff, error) {
	return s.tariffRepo.Create(ctx, input)
}

func (s *TariffService) Update(ctx context.Context, id int, input models.CreateTariffInput) (*models.Tariff, error) {
	return s.tariffRepo.Update(ctx, id, input)
}

func (s *TariffService) Delete(ctx context.Context, id int) error {
	return s.tariffRepo.Delete(ctx, id)
}

func (s *TariffService) ToggleActive(ctx context.Context, id int) (*models.Tariff, error) {
	return s.tariffRepo.ToggleActive(ctx, id)
}
