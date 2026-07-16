package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/mefrraz/bounce/internal/models"
)

type GameWatcher func(game models.Game)

type Scheduler struct {
	mu         sync.Mutex
	windows    map[string]*pollingWindow
	watcher    GameWatcher
	fetchGame  func(string) (*models.Game, error)
	fetchDaily func() ([]models.Game, error)
	stopCh     chan struct{}
}

type pollingWindow struct {
	gameID    string
	startTime time.Time
	stopCh    chan struct{}
}

func New(fetchGame func(string) (*models.Game, error), fetchDaily func() ([]models.Game, error), watcher GameWatcher) *Scheduler {
	return &Scheduler{
		windows: make(map[string]*pollingWindow), watcher: watcher,
		fetchGame: fetchGame, fetchDaily: fetchDaily,
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	log.Println("Scheduler started")
	go s.dailyRefreshLoop()
	go s.pollingLoop()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.windows {
		close(w.stopCh)
	}
}

func (s *Scheduler) dailyRefreshLoop() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
		if now.Hour() < 3 {
			next = time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		}
		select {
		case <-time.After(next.Sub(now)):
			if s.fetchDaily != nil {
				s.fetchDaily()
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) pollingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, w := range s.windows {
				if now.After(w.startTime) {
					go s.pollGame(id)
					delete(s.windows, id)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) pollGame(gameID string) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	s.pollOnce(gameID)
	for {
		select {
		case <-ticker.C:
			if s.pollOnce(gameID) {
				return
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) pollOnce(gameID string) bool {
	game, err := s.fetchGame(gameID)
	if err != nil {
		return false
	}
	if s.watcher != nil {
		s.watcher(*game)
	}
	return game.Status == "FINALIZADO"
}

// ScheduleGame adds a polling window starting at the given time.
func (s *Scheduler) ScheduleGame(gameID string, startTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.windows[gameID]; exists { return }
	s.windows[gameID] = &pollingWindow{
		gameID: gameID, startTime: startTime, stopCh: make(chan struct{}),
	}
	log.Printf("⏰ Scheduled game %s at %s", gameID, startTime.Format("15:04"))
}

// ScheduleGameNow starts polling immediately, ignoring startTime.
func (s *Scheduler) ScheduleGameNow(gameID string) {
	s.mu.Lock()
	if w, exists := s.windows[gameID]; exists {
		close(w.stopCh)
		delete(s.windows, gameID)
	}
	s.mu.Unlock()
	log.Printf("🔍 Polling game %s now", gameID)
	go s.pollGame(gameID)
}

// UnscheduleGame stops polling and removes the game from the scheduler.
func (s *Scheduler) UnscheduleGame(gameID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, exists := s.windows[gameID]; exists {
		close(w.stopCh)
		delete(s.windows, gameID)
		log.Printf("⏰ Unscheduled game %s", gameID)
	}
}
