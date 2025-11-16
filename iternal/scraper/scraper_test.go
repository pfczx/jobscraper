package scraper_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
  "github.com/pfczx/jobscraper/iternal/scraper"
)

// Mock Scraper
type mockScraper struct {
	source string
	offers []scraper.JobOffer
	err    error
	delay  time.Duration
}

func (m *mockScraper) Source() string {
	return m.source
}

func (m *mockScraper) Scrape(ctx context.Context, out chan<- scraper.JobOffer) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	for _, o := range m.offers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- o:
		}
	}

	return m.err
}

func TestRunScrapersCollectsResults(t *testing.T) {
	s1 := &mockScraper{
		source: "s1",
		offers: []scraper.JobOffer{
			{ID: "1", Title: "Offer1"},
			{ID: "2", Title: "Offer2"},
		},
	}

	s2 := &mockScraper{
		source: "s2",
		offers: []scraper.JobOffer{
			{ID: "3", Title: "Offer3"},
		},
	}

	out := scraper.RunScrapers(context.Background(), []scraper.Scraper{s1, s2})

	var results []scraper.JobOffer
	for o := range out {
		results = append(results, o)
	}

	assert.Len(t, results, 3)
	assert.ElementsMatch(t,
		[]string{"1", "2", "3"},
		[]string{results[0].ID, results[1].ID, results[2].ID},
	)
}

func TestRunScrapersHandlesErrors(t *testing.T) {
	s := &mockScraper{
		source: "errScraper",
		err:    errors.New("scrape failed"),
	}

	out := scraper.RunScrapers(context.Background(), []scraper.Scraper{s})
	var results []scraper.JobOffer
	for o := range out {
		results = append(results, o)
	}

	assert.Empty(t, results)
}

func TestRunScrapersClosesChannel(t *testing.T) {
	s := &mockScraper{
		source: "s1",
		offers: []scraper.JobOffer{},
	}

	out := scraper.RunScrapers(context.Background(), []scraper.Scraper{s})

	_, ok := <-out
	assert.False(t, ok, "kanał powinien być zamknięty")
}

func TestRunScrapersParallelExecution(t *testing.T) {
	s1 := &mockScraper{
		source: "s1",
		delay:  200 * time.Millisecond,
		offers: []scraper.JobOffer{{ID: "1"}},
	}

	s2 := &mockScraper{
		source: "s2",
		delay:  200 * time.Millisecond,
		offers: []scraper.JobOffer{{ID: "2"}},
	}

	start := time.Now()
	out := scraper.RunScrapers(context.Background(), []scraper.Scraper{s1, s2})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for range out {
			// read all
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 350*time.Millisecond, "Scrapers should run concurrently")
}

