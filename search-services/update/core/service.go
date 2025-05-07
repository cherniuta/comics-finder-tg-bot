package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int
	updateMu    sync.RWMutex
	isUpdating  bool
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {

	s.updateMu.Lock()
	if s.isUpdating {
		s.updateMu.Unlock()
		return ErrAlreadyExists
	}
	s.isUpdating = true
	s.updateMu.Unlock()

	defer func() {
		s.updateMu.Lock()
		s.isUpdating = false
		s.updateMu.Unlock()
	}()

	comicsFetchedId, err := s.db.IDs(ctx)
	if err != nil {
		return fmt.Errorf("unable to get indexes from local database:%d", err)

	}

	comicsTotal, err := s.xkcd.LastID(ctx)
	if err != nil {
		return fmt.Errorf("unable to get last comics index:%d", err)
	}

	savedIds := make(map[int]bool)
	for _, id := range comicsFetchedId {
		savedIds[id] = true
	}

	var newIds []int
	for id := 1; id <= comicsTotal; id++ {
		if !savedIds[id] {
			newIds = append(newIds, id)
		}
	}

	output := make(chan Comics)
	sema := make(chan struct{}, s.concurrency)
	errChan := make(chan error, len(newIds))
	var count int
	var wg sync.WaitGroup
	wg.Add(len(newIds))

	firstErrChan := make(chan error, 1)
	go func() {
		for err := range errChan {
			s.log.Error("failed to process comic", "error", err)
			select {
			case firstErrChan <- err:
			default:
			}
		}
		close(firstErrChan)
	}()

	for _, id := range newIds {
		go func() {
			defer func() {
				<-sema
				wg.Done()
			}()
			sema <- struct{}{}

			comicsInfo, getErr := s.xkcd.Get(ctx, id)
			if err != nil {
				errChan <- fmt.Errorf("failed to get comics %d: %w", id, getErr)
				return

			}

			words, normErr := s.words.Norm(ctx, comicsInfo.Description)
			if err != nil {
				errChan <- fmt.Errorf("failed to normalize words for comic %d: %w", id, normErr)
				return
			}
			comicsData := Comics{
				ID:    comicsInfo.ID,
				URL:   comicsInfo.URL,
				Words: words}

			output <- comicsData
		}()
	}

	go func() {
		wg.Wait()
		close(output)
		close(errChan)
	}()

	for comics := range output {
		if err = s.db.Add(ctx, comics); err != nil {
			s.log.Debug("err added comics")
			err = fmt.Errorf("failed to add comic to database %d: %w", comics.ID, err)
			continue
		}
		count++
	}
	s.log.Debug("added new comics", "count", count)

	select {
	case err := <-firstErrChan:
		return err
	default:
		return err
	}
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {

	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("unable to get database stats:%d", err)
	}

	comicsRemoteStats, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("unable to get comics stats:%d", err)
	}

	return ServiceStats{
			DBStats:     dbStats,
			ComicsTotal: comicsRemoteStats},
		nil

}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	s.updateMu.RLock()
	defer s.updateMu.RUnlock()

	if s.isUpdating {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {

	err := s.db.Drop(ctx)
	if err != nil {
		return fmt.Errorf("unable to remove comics from database:%d", err)
	}
	return nil
}
