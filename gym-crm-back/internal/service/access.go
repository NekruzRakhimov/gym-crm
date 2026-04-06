package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

type AccessService struct {
	terminalRepo    repository.TerminalRepository
	clientRepo      repository.ClientRepository
	clientTariffRepo repository.ClientTariffRepository
	eventRepo       repository.AccessEventRepository
	hub             *Hub
}

func NewAccessService(
	terminalRepo repository.TerminalRepository,
	clientRepo repository.ClientRepository,
	clientTariffRepo repository.ClientTariffRepository,
	eventRepo repository.AccessEventRepository,
	hub *Hub,
) *AccessService {
	return &AccessService{terminalRepo, clientRepo, clientTariffRepo, eventRepo, hub}
}

func (s *AccessService) ProcessEvent(
	ctx context.Context,
	terminalID int,
	employeeNo string,
	eventTime time.Time,
	authMethod string,
	rawEvent []byte,
) (*models.AccessEvent, error) {
	// 1. Find terminal
	terminal, err := s.terminalRepo.GetByID(ctx, terminalID)
	if err != nil {
		return nil, fmt.Errorf("terminal not found: %w", err)
	}

	direction := terminal.Direction
	method := &authMethod
	terminalName := &terminal.Name
	var clientName *string
	var clientPhoto *string

	event := models.AccessEvent{
		TerminalID:    &terminalID,
		Direction:     direction,
		AuthMethod:    method,
		AccessGranted: false,
		EventTime:     eventTime,
		RawEvent:      rawEvent,
	}

	saveAndBroadcast := func(reason string) (*models.AccessEvent, error) {
		if reason != "" {
			event.DenyReason = &reason
		}
		saved, err := s.eventRepo.Create(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("save event: %w", err)
		}
		s.broadcastEvent(&models.AccessEventDetail{
			AccessEvent:  *saved,
			ClientName:   clientName,
			ClientPhoto:  clientPhoto,
			TerminalName: terminalName,
		})
		return saved, nil
	}

	// 2. Parse client ID
	var clientID int
	if _, err := fmt.Sscanf(employeeNo, "%d", &clientID); err != nil || clientID == 0 {
		log.Printf("unknown employee: %s", employeeNo)
		return saveAndBroadcast("unknown")
	}

	// 3. Find client
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		log.Printf("client %d not found: %v", clientID, err)
		return saveAndBroadcast("unknown")
	}
	event.ClientID = &clientID
	clientName = &client.FullName
	clientPhoto = client.PhotoPath

	// 4. Check active
	if !client.IsActive {
		return saveAndBroadcast("blocked")
	}

	// 5. Check tariff
	activeTariff, err := s.clientTariffRepo.GetActive(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get active tariff: %w", err)
	}
	if activeTariff == nil {
		reason := "no_tariff"
		if hasExpired, _ := s.clientTariffRepo.HasExpired(ctx, clientID); hasExpired {
			reason = "expired"
		} else if hasUpcoming, _ := s.clientTariffRepo.HasUpcoming(ctx, clientID); hasUpcoming {
			reason = "not_started"
		}
		return saveAndBroadcast(reason)
	}

	// 6. Check visit days limit for entries
	if direction == "entry" && activeTariff.MaxVisitDays != nil {
		// Only count a new visit day if client hasn't entered today yet
		hasEntryToday, err := s.eventRepo.HasGrantedEntryToday(ctx, clientID)
		if err != nil {
			return nil, fmt.Errorf("check entry today: %w", err)
		}
		if !hasEntryToday {
			// This would be a new visit day — check if limit reached
			usedDays, err := s.eventRepo.CountVisitDaysInPeriod(ctx, clientID, activeTariff.StartDate, activeTariff.EndDate)
			if err != nil {
				return nil, fmt.Errorf("count visit days: %w", err)
			}
			if usedDays >= *activeTariff.MaxVisitDays {
				return saveAndBroadcast("limit_reached")
			}
		}
	}

	// 7. All checks passed
	event.AccessGranted = true
	return saveAndBroadcast("")
}

// Verify is called for Remote Verification requests from terminals.
// It applies the same access rules as ProcessEvent, saves the event, and
// returns (granted bool, denyReason string, error).
func (s *AccessService) Verify(
	ctx context.Context,
	terminalID int,
	employeeNo string,
	authMethod string,
	rawEvent []byte,
) (bool, string, error) {
	terminal, err := s.terminalRepo.GetByID(ctx, terminalID)
	if err != nil {
		return false, "unknown", fmt.Errorf("terminal not found: %w", err)
	}

	direction := terminal.Direction
	method := &authMethod
	eventTime := time.Now()

	event := models.AccessEvent{
		TerminalID:    &terminalID,
		Direction:     direction,
		AuthMethod:    method,
		AccessGranted: false,
		EventTime:     eventTime,
		RawEvent:      rawEvent,
	}

	terminalName := &terminal.Name
	var clientName *string
	var clientPhoto *string

	save := func(granted bool, reason string) (bool, string, error) {
		event.AccessGranted = granted
		if reason != "" {
			event.DenyReason = &reason
		}
		saved, err := s.eventRepo.Create(ctx, event)
		if err != nil {
			return granted, reason, fmt.Errorf("save event: %w", err)
		}
		s.broadcastEvent(&models.AccessEventDetail{
			AccessEvent:  *saved,
			ClientName:   clientName,
			ClientPhoto:  clientPhoto,
			TerminalName: terminalName,
		})
		return granted, reason, nil
	}

	var clientID int
	if _, err := fmt.Sscanf(employeeNo, "%d", &clientID); err != nil || clientID == 0 {
		log.Printf("remote verify: unknown employee %s", employeeNo)
		return save(false, "unknown")
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		log.Printf("remote verify: client %d not found: %v", clientID, err)
		return save(false, "unknown")
	}
	event.ClientID = &clientID
	clientName = &client.FullName
	clientPhoto = client.PhotoPath

	if !client.IsActive {
		return save(false, "blocked")
	}

	activeTariff, err := s.clientTariffRepo.GetActive(ctx, clientID)
	if err != nil {
		return false, "", fmt.Errorf("get active tariff: %w", err)
	}
	if activeTariff == nil {
		reason := "no_tariff"
		if hasExpired, _ := s.clientTariffRepo.HasExpired(ctx, clientID); hasExpired {
			reason = "expired"
		} else if hasUpcoming, _ := s.clientTariffRepo.HasUpcoming(ctx, clientID); hasUpcoming {
			reason = "not_started"
		}
		return save(false, reason)
	}

	if direction == "entry" && activeTariff.MaxVisitDays != nil {
		hasEntryToday, err := s.eventRepo.HasGrantedEntryToday(ctx, clientID)
		if err != nil {
			return false, "", fmt.Errorf("check entry today: %w", err)
		}
		if !hasEntryToday {
			usedDays, err := s.eventRepo.CountVisitDaysInPeriod(ctx, clientID, activeTariff.StartDate, activeTariff.EndDate)
			if err != nil {
				return false, "", fmt.Errorf("count visit days: %w", err)
			}
			if usedDays >= *activeTariff.MaxVisitDays {
				return save(false, "limit_reached")
			}
		}
	}

	return save(true, "")
}

func (s *AccessService) broadcastEvent(event *models.AccessEventDetail) {
	wsEvt := WSEvent{Type: "access_event", Data: event}
	data, err := json.Marshal(wsEvt)
	if err != nil {
		log.Printf("marshal ws event: %v", err)
		return
	}
	s.hub.Broadcast(data)
}
