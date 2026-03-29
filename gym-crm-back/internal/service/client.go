package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type ClientService struct {
	clientRepo       repository.ClientRepository
	clientTariffRepo repository.ClientTariffRepository
	tariffRepo       repository.TariffRepository
	transactionRepo  repository.TransactionRepository
	syncSvc          *SyncService
	uploadsDir       string
}

func NewClientService(
	clientRepo repository.ClientRepository,
	clientTariffRepo repository.ClientTariffRepository,
	tariffRepo repository.TariffRepository,
	transactionRepo repository.TransactionRepository,
	syncSvc *SyncService,
	uploadsDir string,
) *ClientService {
	return &ClientService{clientRepo, clientTariffRepo, tariffRepo, transactionRepo, syncSvc, uploadsDir}
}

func (s *ClientService) List(ctx context.Context, search string, page, limit int) ([]models.ClientWithTariff, int, error) {
	return s.clientRepo.List(ctx, search, page, limit)
}

func (s *ClientService) GetByID(ctx context.Context, id int) (*models.Client, error) {
	return s.clientRepo.GetByID(ctx, id)
}

func (s *ClientService) Create(ctx context.Context, input models.CreateClientInput) (*models.Client, error) {
	client, err := s.clientRepo.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	go s.syncSvc.SyncClientToAllTerminals(context.Background(), client.ID)
	return client, nil
}

func (s *ClientService) Update(ctx context.Context, id int, input models.UpdateClientInput) (*models.Client, error) {
	return s.clientRepo.Update(ctx, id, input)
}

func (s *ClientService) UploadPhoto(ctx context.Context, id int, r io.Reader) error {
	if err := os.MkdirAll(s.uploadsDir, 0755); err != nil {
		return fmt.Errorf("create uploads dir: %w", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read photo: %w", err)
	}

	// Validate JPEG magic bytes
	if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 || data[2] != 0xFF {
		return fmt.Errorf("invalid JPEG file")
	}

	filename := fmt.Sprintf("%d.jpg", id)
	filePath := filepath.Join(s.uploadsDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write photo: %w", err)
	}

	// Store path with forward slashes so the frontend URL builder works on all OS
	dbPath := strings.ReplaceAll(filePath, "\\", "/")
	if err := s.clientRepo.UpdatePhoto(ctx, id, dbPath); err != nil {
		return fmt.Errorf("update photo path: %w", err)
	}

	go s.syncSvc.SyncFaceToAllTerminals(context.Background(), id)
	return nil
}

func (s *ClientService) Block(ctx context.Context, id int) error {
	if err := s.clientRepo.SetActive(ctx, id, false); err != nil {
		return err
	}
	go s.syncSvc.RemoveClientFromAllTerminals(context.Background(), id)
	return nil
}

func (s *ClientService) Unblock(ctx context.Context, id int) error {
	if err := s.clientRepo.SetActive(ctx, id, true); err != nil {
		return err
	}
	go s.syncSvc.SyncClientToAllTerminals(context.Background(), id)
	return nil
}

func (s *ClientService) Deposit(ctx context.Context, clientID int, input models.DepositInput) (*models.Transaction, error) {
	return s.transactionRepo.Deposit(ctx, clientID, input.Amount, input.Description)
}

func (s *ClientService) GetTransactions(ctx context.Context, clientID int) ([]models.Transaction, error) {
	return s.transactionRepo.ListByClient(ctx, clientID)
}

func (s *ClientService) AssignTariff(ctx context.Context, clientID int, input models.AssignTariffInput) (*models.ClientTariff, error) {
	tariff, err := s.tariffRepo.GetByID(ctx, input.TariffID)
	if err != nil {
		return nil, fmt.Errorf("tariff not found: %w", err)
	}

	// Pre-check balance before creating any records
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}
	if client.Balance < tariff.Price {
		return nil, fmt.Errorf("недостаточно средств: баланс %.2f, необходимо %.2f", client.Balance, tariff.Price)
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}

	endDate := startDate.AddDate(0, 0, tariff.DurationDays)
	ct, err := s.clientTariffRepo.Assign(ctx, clientID, input, endDate, tariff.Price)
	if err != nil {
		return nil, err
	}

	// Deduct tariff price from balance
	desc := fmt.Sprintf("Тариф: %s", tariff.Name)
	if _, err := s.transactionRepo.Payment(ctx, clientID, tariff.Price, desc, ct.ID); err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	go s.syncSvc.SyncClientToAllTerminals(context.Background(), clientID)
	return ct, nil
}

func (s *ClientService) RevokeTariff(ctx context.Context, clientID, tariffRecordID int) error {
	if err := s.clientTariffRepo.Delete(ctx, clientID, tariffRecordID); err != nil {
		return err
	}
	go s.syncSvc.SyncClientToAllTerminals(context.Background(), clientID)
	return nil
}

func (s *ClientService) GetActiveTariff(ctx context.Context, clientID int) (*models.ClientTariffDetail, error) {
	return s.clientTariffRepo.GetActive(ctx, clientID)
}

func (s *ClientService) GetPayments(ctx context.Context, clientID int) ([]models.ClientTariffDetail, error) {
	return s.clientTariffRepo.ListByClient(ctx, clientID)
}

func (s *ClientService) Delete(ctx context.Context, id int) error {
	// Remove from all terminals first (best-effort, don't fail if terminal unreachable)
	go s.syncSvc.RemoveClientFromAllTerminals(context.Background(), id)

	// Delete photo file
	photoFile := filepath.Join(s.uploadsDir, fmt.Sprintf("%d.jpg", id))
	os.Remove(photoFile)

	return s.clientRepo.Delete(ctx, id)
}
