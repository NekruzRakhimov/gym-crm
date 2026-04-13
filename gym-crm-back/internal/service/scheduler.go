package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
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

// hasTimeRestrictions reports whether a tariff has any time or day-of-week
// restrictions. Clients without restrictions never need scheduler-driven syncs
// because their Valid.endTime is already set correctly when the tariff is
// assigned and the terminal enforces it at the hardware level.
func hasTimeRestrictions(tariff *models.ClientTariffDetail) bool {
	if tariff.ScheduleDays != "" && tariff.ScheduleDays != "all" {
		return true
	}
	return tariff.TimeFrom != nil || tariff.TimeTo != nil
}

// SchedulerService runs a background job to sync terminal access based on
// tariff schedules. It only syncs a client when their access window actually
// changes (allowed → denied or denied → allowed), which keeps terminal HTTP
// traffic near zero during steady-state operation.
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

const (
	schedulerInterval = 1 * time.Minute
	// maxConcurrentSyncs caps the number of simultaneous UpsertPerson calls to
	// avoid flooding terminals when many clients cross a boundary at once
	// (e.g. gym opens at 09:00 and hundreds of time-restricted tariffs flip).
	maxConcurrentSyncs = 10
)

// Run starts the scheduler loop. Call in a goroutine.
func (s *SchedulerService) Run(ctx context.Context) {
	// Run immediately on startup to apply any boundaries missed while offline.
	s.actualize(ctx)

	ticker := time.NewTicker(schedulerInterval)
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
	// Compare against one interval ago to detect boundary crossings.
	prev := now.Add(-schedulerInterval)

	clients, _, err := s.clientRepo.List(ctx, "", 1, 100000)
	if err != nil {
		log.Printf("scheduler: list clients: %v", err)
		return
	}

	sem := make(chan struct{}, maxConcurrentSyncs)
	var wg sync.WaitGroup

	for _, cl := range clients {
		if !cl.IsActive {
			continue
		}
		cl := cl
		wg.Add(1)
		sem <- struct{}{} // acquire slot (blocks if maxConcurrentSyncs reached)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.actualizeClient(ctx, cl.ID, now, prev)
		}()
	}
	wg.Wait()
}

func (s *SchedulerService) actualizeClient(ctx context.Context, clientID int, now, prev time.Time) {
	tariff, err := s.clientTariffRepo.GetActive(ctx, clientID)
	if err != nil {
		log.Printf("scheduler: get tariff client %d: %v", clientID, err)
		return
	}
	if tariff == nil {
		// No active tariff — Valid.endTime is already set to past on the terminal
		// (done at assign/revoke time). Nothing to do.
		return
	}

	// Clients without time/day restrictions don't need scheduler-driven syncs.
	// Their endTime is set to tariff.EndDate at assignment time; the terminal
	// enforces expiry itself.
	if !hasTimeRestrictions(tariff) {
		return
	}

	wasAllowed := IsAccessAllowed(tariff, prev)
	isAllowed := IsAccessAllowed(tariff, now)

	// No boundary crossing — nothing to sync.
	if wasAllowed == isAllowed {
		return
	}

	var endTime time.Time
	if isAllowed {
		// Entered access window — restore endTime to end of tariff's last day.
		d := tariff.EndDate
		endTime = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
	} else {
		// Left access window — expire immediately so terminal denies entry.
		endTime = now.AddDate(0, 0, -1)
	}

	if err := s.syncSvc.SyncPersonToAllTerminals(ctx, clientID, endTime); err != nil {
		log.Printf("scheduler: sync client %d: %v", clientID, err)
	}
}
