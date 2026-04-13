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

// terminalOpTimeout is the per-terminal deadline for a single ISAPI call
// (UpsertPerson, UploadFace, DeletePerson, …). The http.Client already has a
// 10-second timeout, but wrapping each terminal operation in its own
// context.WithTimeout lets callers cancel the whole batch cleanly and ensures
// a slow terminal does not block runOnTerminals indefinitely when the http
// keep-alive machinery delays a connection reset.
const terminalOpTimeout = 10 * time.Second

// maxSyncParallel caps the number of concurrent client→terminal sync
// goroutines in SyncAllClientsToTerminal. Without this, syncing 500 clients
// would fire 500 simultaneous HTTPS requests at one terminal, saturating its
// limited connection-accept queue and causing flapping.
const maxSyncParallel = 10

type SyncService struct {
	terminalRepo     repository.TerminalRepository
	clientRepo       repository.ClientRepository
	clientTariffRepo repository.ClientTariffRepository
	uploadsDir       string

	// hikClients caches one HikvisionClient per terminal so the underlying
	// http.Transport (and its connection pool) is reused across calls.
	// Invalidate an entry via InvalidateTerminalClient when terminal credentials
	// or IP change (Update/Delete).
	hikClients map[int]*clients.HikvisionClient
	hikMu      sync.RWMutex
}

func NewSyncService(
	terminalRepo repository.TerminalRepository,
	clientRepo repository.ClientRepository,
	clientTariffRepo repository.ClientTariffRepository,
	uploadsDir string,
) *SyncService {
	return &SyncService{
		terminalRepo:     terminalRepo,
		clientRepo:       clientRepo,
		clientTariffRepo: clientTariffRepo,
		uploadsDir:       uploadsDir,
		hikClients:       make(map[int]*clients.HikvisionClient),
	}
}

// getHikvisionClient returns the cached HikvisionClient for the given terminal,
// creating one on first use. The client is safe for concurrent use.
func (s *SyncService) getHikvisionClient(t models.Terminal) *clients.HikvisionClient {
	s.hikMu.RLock()
	c, ok := s.hikClients[t.ID]
	s.hikMu.RUnlock()
	if ok {
		return c
	}

	s.hikMu.Lock()
	defer s.hikMu.Unlock()
	// Double-check after acquiring write lock.
	if c, ok = s.hikClients[t.ID]; ok {
		return c
	}
	c = clients.NewHikvisionClient(t.IP, t.Port, t.Username, t.Password)
	s.hikClients[t.ID] = c
	return c
}

// ClientForTerminal returns the shared HikvisionClient for the given terminal.
// It is safe for concurrent use from multiple goroutines and is intended for
// callers outside SyncService (e.g. TerminalController.GetStatus) so that all
// ISAPI traffic to a terminal flows through a single http.Transport, keeping
// the connection pool intact.
func (s *SyncService) ClientForTerminal(t models.Terminal) *clients.HikvisionClient {
	return s.getHikvisionClient(t)
}

// InvalidateTerminalClient removes the cached HikvisionClient for a terminal.
// Call this after updating or deleting a terminal so the next request picks up
// the new credentials/IP.
func (s *SyncService) InvalidateTerminalClient(terminalID int) {
	s.hikMu.Lock()
	defer s.hikMu.Unlock()
	delete(s.hikClients, terminalID)
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

func (s *SyncService) SyncClientToAllTerminals(ctx context.Context, clientID int) error {
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	// No active tariff → expire on terminal (yesterday = no access)
	endTime := time.Now().AddDate(0, 0, -1)
	if ct, err := s.clientTariffRepo.GetActive(ctx, clientID); err == nil && ct != nil {
		// Set to end of day so the terminal allows access the entire last day
		d := ct.EndDate
		endTime = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
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

	return s.runOnTerminals(ctx, terminals, func(tCtx context.Context, t models.Terminal) error {
		hik := s.getHikvisionClient(t)
		if err := hik.UpsertPerson(tCtx, clientID, client.FullName, endTime); err != nil {
			return err
		}
		if faceData != nil {
			if err := hik.UploadFace(tCtx, clientID, faceData); err != nil {
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
	return s.runOnTerminals(ctx, terminals, func(tCtx context.Context, t models.Terminal) error {
		return s.getHikvisionClient(t).UpsertPerson(tCtx, clientID, client.FullName, endTime)
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
	return s.runOnTerminals(ctx, terminals, func(tCtx context.Context, t models.Terminal) error {
		return s.getHikvisionClient(t).UploadFace(tCtx, clientID, jpegData)
	})
}

func (s *SyncService) RemoveClientFromAllTerminals(ctx context.Context, clientID int) error {
	terminals, err := s.terminalRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list terminals: %w", err)
	}
	return s.runOnTerminals(ctx, terminals, func(tCtx context.Context, t models.Terminal) error {
		return s.getHikvisionClient(t).DeletePerson(tCtx, clientID)
	})
}

// SyncAllClientsToTerminal pushes every active client to a single terminal.
// A semaphore (maxSyncParallel) limits concurrency so the terminal is not
// flooded with hundreds of simultaneous HTTPS connections.
func (s *SyncService) SyncAllClientsToTerminal(ctx context.Context, terminalID int) error {
	terminal, err := s.terminalRepo.GetByID(ctx, terminalID)
	if err != nil {
		return fmt.Errorf("get terminal: %w", err)
	}

	clientsList, _, err := s.clientRepo.List(ctx, "", 1, 10000)
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}

	hik := s.getHikvisionClient(*terminal)

	sem := make(chan struct{}, maxSyncParallel)
	var wg sync.WaitGroup

	for _, cl := range clientsList {
		if !cl.IsActive {
			continue
		}
		cl := cl
		wg.Add(1)
		sem <- struct{}{} // acquire slot — blocks when maxSyncParallel reached
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Each client gets its own timeout so a single slow response
			// cannot stall the whole batch.
			tCtx, cancel := context.WithTimeout(ctx, terminalOpTimeout)
			defer cancel()

			endTime := time.Now().AddDate(0, 0, -1)
			if ct, err := s.clientTariffRepo.GetActive(tCtx, cl.ID); err == nil && ct != nil {
				d := ct.EndDate
				endTime = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
			}

			if err := hik.UpsertPerson(tCtx, cl.ID, cl.FullName, endTime); err != nil {
				log.Printf("sync client %d to terminal %d: %v", cl.ID, terminalID, err)
				return
			}

			if cl.PhotoPath != nil {
				if jpegData, err := s.loadAndCompressFace(cl.ID); err == nil {
					if err := hik.UploadFace(tCtx, cl.ID, jpegData); err != nil {
						log.Printf("sync face %d to terminal %d: %v", cl.ID, terminalID, err)
					}
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

// runOnTerminals fans out fn across all terminals concurrently and waits for
// all goroutines to finish. Each invocation receives a child context with
// terminalOpTimeout so a single unresponsive terminal cannot block the
// entire batch beyond that deadline.
func (s *SyncService) runOnTerminals(ctx context.Context, terminals []models.Terminal, fn func(context.Context, models.Terminal) error) error {
	var wg sync.WaitGroup
	for _, t := range terminals {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			tCtx, cancel := context.WithTimeout(ctx, terminalOpTimeout)
			defer cancel()
			if err := fn(tCtx, t); err != nil {
				log.Printf("terminal %d (%s): %v", t.ID, t.IP, err)
			}
		}()
	}
	wg.Wait()
	return nil
}
