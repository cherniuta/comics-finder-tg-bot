package core

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
)

const numWorkers = 3

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
	index *Index
}

func NewService(log *slog.Logger, db DB, words Words) (*Service, error) {
	return &Service{
		log:   log,
		db:    db,
		words: words,
		index: NewIndex()}, nil
}

func workerSearch(word string, s *Service, ctx context.Context) ([]int, error) {
	ids, err := s.db.Search(ctx, word)
	return ids, err
}
func prioritySorting(output <-chan []int) []int {

	countingIds := make(map[int]int, 0)
	for result := range output {
		for _, id := range result {
			countingIds[id]++
		}
	}

	sortedIds := slices.SortedFunc(maps.Keys(countingIds), func(a, b int) int {
		return cmp.Compare(countingIds[b], countingIds[a])
	})

	return sortedIds
}
func (s *Service) Search(ctx context.Context, query SearchQuery) ([]Comics, error) {
	wordsNorm, err := s.words.Norm(ctx, query.Keywords)
	if err != nil {
		return nil, err
	}

	input := make(chan string)
	go func() {
		for _, i := range wordsNorm {
			input <- i
		}
		close(input)
	}()

	output := make(chan []int)
	errChan := make(chan error, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	firstErrChan := make(chan error, 1)
	go func() {
		for err := range errChan {
			s.log.Error("couldn't find the comics", "error", err)
			select {
			case firstErrChan <- err:
			default:
			}
		}
		close(firstErrChan)
	}()

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for word := range input {
				ids, err := workerSearch(word, s, ctx)
				if err != nil {
					errChan <- fmt.Errorf("error when searching comics by word %s: %w", word, err)
					return
				}
				output <- ids
			}
		}()
	}

	go func() {
		wg.Wait()
		close(output)
		close(errChan)

	}()

	sortedIds := prioritySorting(output)

	var (
		errGet error
		count  int
	)
	cachedComics := make([]Comics, 0)

	for _, index := range sortedIds {

		var comics Comics
		comics, errGet = s.db.Get(ctx, index)
		if errGet != nil {
			s.log.Error("err get comics", "error", errGet)
			errGet = fmt.Errorf("failed to get comic from the database %d: %w", index, errGet)
			break
		}

		cachedComics = append(cachedComics, comics)
		count++
		if count == query.Limit {
			break
		}
	}

	select {
	case err := <-firstErrChan:
		return cachedComics, err
	default:
		return cachedComics, errGet
	}

}
func workerSearchIndex(word string, s *Service) []int {
	ids := s.index.Get(word)
	return ids
}
func (s *Service) SearchIndex(ctx context.Context, query SearchQuery) ([]Comics, error) {
	wordsNorm, err := s.words.Norm(ctx, query.Keywords)
	if err != nil {
		return nil, err
	}

	input := make(chan string)
	go func() {
		for _, i := range wordsNorm {
			input <- i
		}
		close(input)
	}()

	output := make(chan []int)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for word := range input {
				ids := workerSearchIndex(word, s)
				output <- ids
			}
		}()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	sortedIds := prioritySorting(output)

	var (
		errGet error
		count  int
	)
	cachedComics := make([]Comics, 0)

	for _, index := range sortedIds {

		var comics Comics
		comics, errGet = s.db.Get(ctx, index)
		if errGet != nil {
			s.log.Error("err get comics", "error", errGet)
			errGet = fmt.Errorf("failed to get comic from the database %d: %w", index, errGet)
			break
		}

		cachedComics = append(cachedComics, comics)
		count++
		if count == query.Limit {
			break
		}
	}

	return cachedComics, errGet

}

func (s *Service) BuildIndex(ctx context.Context) error {

	s.index.Drop()
	maxId, err := s.db.MaxId(ctx)
	if err != nil {
		return err
	}

	for comicsId := 1; comicsId <= maxId; comicsId++ {
		var item Comics
		item, err = s.db.Get(ctx, comicsId)
		if err != nil {
			return err
		}
		s.index.Add(comicsId, item.Words)
	}

	return nil
}
