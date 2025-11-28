package scrapers

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"log"
	"math/rand"
	"strings"
	"time"
)

var proxyList = []string{
	"213.73.25.231:8080",
	
}

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

// wait times are random (min,max) in seconds
type PracujScraper struct {
	minTimeS int
	maxTimeS int
	urls     []string
}

func NewPracujScraper(urls []string) *PracujScraper {
	return &PracujScraper{
		minTimeS: 20,
		maxTimeS: 40,
		urls:     urls,
	}
}

func (*PracujScraper) Source() string {
	return "pracuj.pl"
}

// extracting data from string html with goquer selectors
func (p *PracujScraper) extractDataFromHTML(html string, url string) (scraper.JobOffer, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return scraper.JobOffer{}, err
	}
	var job scraper.JobOffer
	job.URL = url
	job.Source = p.Source()
	job.Title = strings.TrimSpace(doc.Find(titleSelector).Text())
	job.Company = strings.TrimSpace(doc.Find(companySelector).Text())
	job.Location = strings.TrimSpace(doc.Find(locationSelector).Text())

	doc.Find(descriptionSelector).Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			job.Description += t + "\n"
		}
	})

	doc.Find(skillsSelector).Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			job.Skills = append(job.Skills, t)
		}
	})

	doc.Find(salarySectionSelector).Each(func(_ int, s *goquery.Selection) {
		parts := strings.Split(strings.TrimSpace(s.Text()), "|")
		if len(parts) != 2 {
			return
		}

		amount := strings.TrimSpace(parts[0])
		ctype := strings.TrimSpace(parts[1])

		switch ctype {
		case "umowa o pracę":
			job.SalaryEmployment = amount
		case "umowa zlecenie":
			job.SalaryContract = amount
		case "kontrakt B2B":
			job.SalaryB2B = amount
		}
	})

	return job, nil
}

// html chromedp
func (p *PracujScraper) getHTMLContent(chromeDpCtx context.Context, url string) (string, error) {
	var html string

	//chromdp run config
	err := chromedp.Run(
		chromeDpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(time.Duration(rand.Intn(800)+300)*time.Millisecond),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	return html, err
}

// main func for scraping
func (p *PracujScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {

	//chromdp config
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		//chromedp.Flag("proxy-server", proxyList[rand.Intn(len(proxyList))]),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-site-isolation-trials", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	for index, url := range p.urls {
		html, err := p.getHTMLContent(chromeDpCtx, url)
		if err != nil {
			log.Printf("Chromedp error: %v", err)
			continue
		}

		job, err := p.extractDataFromHTML(html, url)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case q <- job:
		}

		log.Printf("Scraped %d: %s", index, url)
		randomDelay := rand.Intn(p.maxTimeS-p.minTimeS) + p.minTimeS
		log.Printf("Sleeping for: %ds", randomDelay)
		time.Sleep(time.Duration(randomDelay) * time.Second)
	}

	return nil
}
