package scrapers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocolly/colly"
	"github.com/pfczx/jobscraper/iternal/scraper"
	 // "github.com/pfczx/jobscraper/iternal/scraper/scrapers" - reusing pracujscraper without domain lock
)

// Mock HTML
const mockHTML = `
<html>
<body>
<h1 data-scroll-id="job-title">Senior Golang Developer</h1>
<h2 data-scroll-id="employer-name">Tech Corp</h2>
<div data-test="offer-badge-title">Warszawa</div>
<ul data-test="text-about-project">
	<li>Praca nad API</li>
	<li>Mikroserwisy</li>
</ul>
<span data-test="item-technologies-expected">Go</span>
<span data-test="item-technologies-optional">Docker</span>
<div data-test="section-salaryPerContractType">10 000 zł | umowa o pracę</div>
</body>
</html>
`

// new pracujscraper without domain lock
type TestPracujScraper struct {
	urls []string
}

func (t *TestPracujScraper) Source() string {
	return "pracuj.pl"
}

func (t *TestPracujScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {
	c := colly.NewCollector()

	c.OnHTML("html", func(e *colly.HTMLElement) {
		var job scraper.JobOffer
		job.Title = e.ChildText("h1[data-scroll-id='job-title']")
		job.Company = e.ChildText("h2[data-scroll-id='employer-name']")
		job.Location = e.ChildText("div[data-test='offer-badge-title']")

		e.ForEach("ul[data-test='text-about-project'] li", func(_ int, el *colly.HTMLElement) {
			job.Description += el.Text + "\n"
		})

		var skills []string
		e.ForEach("span[data-test='item-technologies-expected'], span[data-test='item-technologies-optional']", func(_ int, el *colly.HTMLElement) {
			skills = append(skills, el.Text)
		})
		job.Skills = skills

		e.ForEach("div[data-test='section-salaryPerContractType']", func(_ int, el *colly.HTMLElement) {
			parts := strings.Split(el.Text, "|")
			if len(parts) != 2 {
				return
			}
			amount := strings.TrimSpace(parts[0])
			ctype := strings.TrimSpace(parts[1])
			if ctype == "umowa o pracę" {
				job.SalaryEmployment = amount
			}
		})

		select {
		case <-ctx.Done():
			return
		case q <- job:
		}
	})

	for _, url := range t.urls {
		time.Sleep(0)
		if err := c.Visit(url); err != nil {
			return err
		}
	}

	c.Wait()
	return nil
}

func TestPracujScraperWithoutAllowedDomains(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	s := &TestPracujScraper{
		urls: []string{ts.URL},
	}

	ctx := context.Background()
	out := make(chan scraper.JobOffer)
	go func() {
		_ = s.Scrape(ctx, out)
		close(out)
	}()

	var offers []scraper.JobOffer
	for o := range out {
		offers = append(offers, o)
	}

	if len(offers) != 1 {
		t.Fatalf("expected 1 offer, got %d", len(offers))
	}

	job := offers[0]
	if job.Title != "Senior Golang Developer" {
		t.Errorf("wrong title: %s", job.Title)
	}
	if job.Company != "Tech Corp" {
		t.Errorf("wrong company: %s", job.Company)
	}
	if job.Location != "Warszawa" {
		t.Errorf("wrong location: %s", job.Location)
	}
	if len(job.Skills) != 2 {
		t.Errorf("wrong skills: %v", job.Skills)
	}
	if job.SalaryEmployment != "10 000 zł" {
		t.Errorf("wrong salary: %s", job.SalaryEmployment)
	}
}

