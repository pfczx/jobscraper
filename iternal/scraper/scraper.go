package scraper

import (
	"context"
	"log"
	"sync"
	"time"
)

type JobOffer struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Company          string   `json:"company"`
	Location         string   `json:"location"`
	SalaryEmployment string   `json:"salary_employment"`
	SalaryContract   string   `json:"salary_contract"`
	SalaryB2B        string   `json:"salary_b2b"`
	Description      string   `json:"description"`
	URL              string   `json:"url"`
	Source           string   `json:"source"`
	PublishedAt      *string  `json:"published_at,omitempty"` //potencial problems
	Skills           []string `json:"skills,omitempty"`
}

type Scraper interface {
	Source() string
	Scrape(ctx context.Context, q chan<- JobOffer) error
}

func RunScrapers(ctx context.Context, scrapers []Scraper, parallel bool) chan JobOffer {
	out := make(chan JobOffer)
	var wg sync.WaitGroup

	go func() {
		if parallel {
			for _, s := range scrapers {
				time.Sleep(5 * time.Second)
				wg.Add(1)
				go func(scr Scraper) {
					defer wg.Done()
					log.Printf("Starting scraper: %s", scr.Source())
					if err := scr.Scrape(ctx, out); err != nil {
						log.Printf("Error in scraper %s: %v", scr.Source(), err)
					}
					log.Printf("Finished scraper: %s", scr.Source())
				}(s)
			}
			wg.Wait()
		} else {
			for _, s := range scrapers {
				log.Printf("Starting scraper: %s", s.Source())
				if err := s.Scrape(ctx, out); err != nil {
					log.Printf("Error in scraper %s: %v", s.Source(), err)
				}
				log.Printf("Finished scraper: %s", s.Source())
			}
		}

		close(out)
	}()

	return out
}
