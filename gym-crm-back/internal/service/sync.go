package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/clients"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type SyncService struct {
	terminalRepo     repository.TerminalRepository
	clientRepo       repository.ClientRepository
	clientTariffRepo repository.ClientTariffRepository
	uploadsDir       string
}

func NewSyncService(
	terminalRepo repository.TerminalRepository,
	clientRepo repository.ClientRepository,
	clientTariffRepo repository.ClientTariffRepository,
	uploadsDir string,
) *SyncService {
	return &SyncService{terminalRepo, clientRepo, clientTariffRepo, uploadsDir}
}

// loadAndCompressFace reads the face JPEG from disk and compresses it to ≤200KB.
func (s *SyncService) loadAndCompressFace(clientID int) ([]byte, error) {
	photoFile := filepath.Join(s.uploadsDir, fmt.Sprintf("%d.jpg", clientID))
	raw, err := os.ReadFile(photoFile)
	if err != nil {
		return nil, err
	}
	return compressJPEG(raw, 200*1024)
}

// compressJPEG re-encodes a JPEG (or PNG) to stay under maxBytes.
// It tries quality 85, 70, 55, 40 before giving up.
func compressJPEG(data []byte, maxBytes int) ([]byte, error) {
	if len(data) <= maxBytes {
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	for _, q := range []int{85, 70, 55, 40} {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, fmt.Errorf("encode jpeg: %w", err)
		}
		if buf.Len() <= maxBytes {
			return buf.Bytes(), nil
		}
	}
	// Return lowest quality result anyway
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 40}) //nolint:errcheck
	return buf.Bytes(), nil
}

func newHikvisionClient(t models.Terminal) *clients.HikvisionClient {
	return clients.NewHikvisionClient(t.IP, t.Port, t.Username, t.Password)
}

func (s *SyncService) SyncClientToAllTerminals(ctx context.Context, clientID int) error {
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	// No active tariff → expire on terminal (yesterday = no access)
	endTime := time.Now().AddDate(0, 0, -1)
	if ct, err := s.clientTariffRepo.GetActive(ctx, clientID); err == nil && ct != nil {
		endTime = ct.EndDate
	}

	terminals, err := s.terminalRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list terminals: %w", err)
	}

	// Load face photo if exists
	var faceData []byte
	if jpegData, err := s.loadAndCompressFace(clientID); err == nil {
		faceData = jpegData
	}

	return s.runOnTerminals(terminals, func(t models.Terminal) error {
		hik := newHikvisionClient(t)
		if err := hik.UpsertPerson(clientID, client.FullName, endTime); err != nil {
			return err
		}
		if faceData != nil {
			if err := hik.UploadFace(clientID, faceData); err != nil {
				log.Printf("upload face client %d terminal %s: %v", clientID, t.IP, err)
			}
		}
		return nil
	})
}

// SyncPersonToAllTerminals updates only Valid.endTime on all terminals — no face upload.
// Used by the scheduler to enforce schedule-based access without re-uploading photos.
func (s *SyncService) SyncPersonToAllTerminals(ctx context.Context, clientID int, endTime time.Time) error {
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}
	terminals, err := s.terminalRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list terminals: %w", err)
	}
	return s.runOnTerminals(terminals, func(t models.Terminal) error {
		return newHikvisionClient(t).UpsertPerson(clientID, client.FullName, endTime)
	})
}

func (s *SyncService) SyncFaceToAllTerminals(ctx context.Context, clientID int) error {
	jpegData, err := s.loadAndCompressFace(clientID)
	if err != nil {
		return fmt.Errorf("load face: %w", err)
	}
	terminals, err := s.terminalRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list terminals: %w", err)
	}
	return s.runOnTerminals(terminals, func(t models.Terminal) error {
		return newHikvisionClient(t).UploadFace(clientID, jpegData)
	})
}

func (s *SyncService) RemoveClientFromAllTerminals(ctx context.Context, clientID int) error {
	terminals, err := s.terminalRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list terminals: %w", err)
	}
	return s.runOnTerminals(terminals, func(t models.Terminal) error {
		hik := newHikvisionClient(t)
		return hik.DeletePerson(clientID)
	})
}

func (s *SyncService) SyncAllClientsToTerminal(ctx context.Context, terminalID int) error {
	terminal, err := s.terminalRepo.GetByID(ctx, terminalID)
	if err != nil {
		return fmt.Errorf("get terminal: %w", err)
	}

	clients_list, _, err := s.clientRepo.List(ctx, "", 1, 10000)
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}

	hik := newHikvisionClient(*terminal)

	var wg sync.WaitGroup
	for _, cl := range clients_list {
		if !cl.IsActive {
			continue
		}
		cl := cl
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			endTime := time.Now().AddDate(0, 0, -1)
			if ct, err := s.clientTariffRepo.GetActive(ctx2, cl.ID); err == nil && ct != nil {
				endTime = ct.EndDate
			}

			if err := hik.UpsertPerson(cl.ID, cl.FullName, endTime); err != nil {
				log.Printf("sync client %d to terminal %d: %v", cl.ID, terminalID, err)
				return
			}

			// Upload face if photo exists
			if cl.PhotoPath != nil {
				if jpegData, err := s.loadAndCompressFace(cl.ID); err == nil {
					if err := hik.UploadFace(cl.ID, jpegData); err != nil {
						log.Printf("sync face %d to terminal %d: %v", cl.ID, terminalID, err)
					}
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

func (s *SyncService) runOnTerminals(terminals []models.Terminal, fn func(models.Terminal) error) error {
	var wg sync.WaitGroup
	for _, t := range terminals {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(t); err != nil {
				log.Printf("terminal %d (%s): %v", t.ID, t.IP, err)
			}
		}()
	}
	wg.Wait()
	return nil
}
