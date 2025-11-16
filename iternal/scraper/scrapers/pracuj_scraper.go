package scrapers

import (
	"context"
	"github.com/gocolly/colly"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"log"
	"strings"
	"time"
)

const (
	titleSelector         = "h1[data-scroll-id='job-title']"
	companySelector       = "h2[data-scroll-id='employer-name']"
	locationSelector      = "div[data-test='offer-badge-title']"
	descriptionSelector   = `ul[data-test="text-about-project"]`                                                         //concat in code
	skillsSelector        = `span[data-test="item-technologies-expected"], span[data-test="item-technologies-optional"]` //concat in code
	salarySectionSelector = `div[data-test="section-salaryPerContractType"]`
	salaryAmountSelector  = `div[data-test="text-earningAmount"]`
	contractTypeSelector  = `span[data-test="text-contractTypeName"]`
)

type PracujScraper struct {
	TimeoutBetweenScraps time.Duration
	collector            *colly.Collector
	urls                 []string
}

// controls
func NewPracujScraper(urls []string) *PracujScraper {
	c := colly.NewCollector(
		colly.AllowedDomains("www.pracuj.pl", "pracuj.pl"),
		//colly.Async(true),
	)

	// #nosec G104 - false positive i guess
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*pracuj.pl*",
		Parallelism: 2,
		RandomDelay: 2 * time.Second,
	})

	return &PracujScraper{
		TimeoutBetweenScraps: 10 * time.Second,
		collector:            c,
		urls:                 urls,
	}
}

func (*PracujScraper) Source() string {
	return "pracuj.pl"
}

func (p *PracujScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {
	p.collector.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	p.collector.OnHTML("html", func(e *colly.HTMLElement) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var job scraper.JobOffer
		job.URL = e.Request.URL.String()
		job.Source = p.Source()
		job.Title = e.ChildText(titleSelector)
		job.Company = e.ChildText(companySelector)
		job.Location = e.ChildText(locationSelector)

		// description
		e.ForEach(descriptionSelector+" li", func(_ int, el *colly.HTMLElement) {
			if text := el.Text; text != "" {
				job.Description += text + "\n"
			}
		})

		// skills
		var skills []string
		e.ForEach(skillsSelector, func(_ int, el *colly.HTMLElement) {
			if text := el.Text; text != "" {
				skills = append(skills, text)
			}
		})
		job.Skills = skills

		// salary

		e.ForEach(salarySectionSelector, func(_ int, el *colly.HTMLElement) {
			sectionText := strings.TrimSpace(el.Text)
			parts := strings.Split(sectionText, "|")
			if len(parts) != 2 {
				return
			}
			amount := strings.TrimSpace(parts[0])
			ctype := strings.TrimSpace(parts[1])

			if ctype == "umowa o pracę" {
				job.SalaryEmployment = amount
			}
			if ctype == "umowa zlecenie" {
				job.SalaryContract = amount
			}
			if ctype == "kontrakt B2B" {
				job.SalaryB2B = amount
			}
		})

		select {
		case <-ctx.Done():
			return
		case q <- job:
		}
	})

	// Pętla po URL-ach
	for _, url := range p.urls {
		time.Sleep(p.TimeoutBetweenScraps)
		log.Println("Waiting timeoutBetweenScraps")
		if err := p.collector.Visit(url); err != nil {
			log.Printf("Visit error: %v", err)
			return err
		}
	}

	p.collector.Wait()
	return nil
}
