package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
)

var weekdayAbbr = map[time.Weekday]string{
	time.Monday: "mon", time.Tuesday: "tue", time.Wednesday: "wed",
	time.Thursday: "thu", time.Friday: "fri", time.Saturday: "sat", time.Sunday: "sun",
}

// IsAccessAllowed checks whether a tariff's schedule permits access at the given time.
func IsAccessAllowed(tariff *models.ClientTariffDetail, now time.Time) bool {
	switch tariff.ScheduleDays {
	case "all", "":
		// no restriction
	case "weekdays":
		if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
			return false
		}
	case "weekends":
		if now.Weekday() != time.Saturday && now.Weekday() != time.Sunday {
			return false
		}
	case "even":
		if now.Day()%2 != 0 {
			return false
		}
	case "odd":
		if now.Day()%2 == 0 {
			return false
		}
	default:
		// Comma-separated custom days: "mon,wed,fri"
		current := weekdayAbbr[now.Weekday()]
		allowed := false
		for _, d := range strings.Split(tariff.ScheduleDays, ",") {
			if strings.TrimSpace(d) == current {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	if tariff.TimeFrom != nil && tariff.TimeTo != nil {
		currentMins := now.Hour()*60 + now.Minute()
		fromMins := parseHHMM(*tariff.TimeFrom)
		toMins := parseHHMM(*tariff.TimeTo)
		if currentMins < fromMins || currentMins >= toMins {
			return false
		}
	}

	return true
}

func parseHHMM(s string) int {
	var h, m int
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h*60 + m
}

// SchedulerService runs a background job to sync terminal access based on tariff schedules.
type SchedulerService struct {
	clientRepo       repository.ClientRepository
	clientTariffRepo repository.ClientTariffRepository
	syncSvc          *SyncService
}

func NewSchedulerService(
	clientRepo repository.ClientRepository,
	clientTariffRepo repository.ClientTariffRepository,
	syncSvc *SyncService,
) *SchedulerService {
	return &SchedulerService{clientRepo, clientTariffRepo, syncSvc}
}

// Run starts the scheduler loop. Call in a goroutine.
func (s *SchedulerService) Run(ctx context.Context) {
	// Run immediately on startup, then every minute.
	s.actualize(ctx)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.actualize(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerService) actualize(ctx context.Context) {
	now := time.Now()
	clients, _, err := s.clientRepo.List(ctx, "", 1, 100000)
	if err != nil {
		log.Printf("scheduler: list clients: %v", err)
		return
	}

	for _, cl := range clients {
		if !cl.IsActive {
			continue
		}
		cl := cl
		go s.actualizeClient(ctx, cl.ID, now)
	}
}

func (s *SchedulerService) actualizeClient(ctx context.Context, clientID int, now time.Time) {
	tariff, err := s.clientTariffRepo.GetActive(ctx, clientID)
	if err != nil {
		log.Printf("scheduler: get tariff client %d: %v", clientID, err)
		return
	}
	if tariff == nil {
		// No active tariff — terminal already has expired endTime from assign/revoke sync.
		return
	}

	var endTime time.Time
	if IsAccessAllowed(tariff, now) {
		// Within access window — set endTime to end of the last day so the
		// terminal allows access the entire last day (not just until midnight).
		d := tariff.EndDate
		endTime = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
	} else {
		// Outside access window (wrong day/time) — expire immediately.
		endTime = now.AddDate(0, 0, -1)
	}

	if err := s.syncSvc.SyncPersonToAllTerminals(ctx, clientID, endTime); err != nil {
		log.Printf("scheduler: sync client %d: %v", clientID, err)
	}
}
