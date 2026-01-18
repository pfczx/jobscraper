package scrapers

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/config"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"log"
	"math/rand"
	"strings"
	"time"
)

// selectors
const (
	nofluffjobstitleSelector            = "div.posting-details-description h1"
	nofluffjobscompanySelector          = "a#postingCompanyUrl"
	nofluffjobslocationSelector         = "span.locations-text span"
	nofluffjobsdescriptionSelector      = "#posting-description nfj-read-more"
	nofluffjobsskillsSelector           = "#posting-requirements"
	nofluffjobssalarySectionSelector    = "common-posting-salaries-list div.salary"
	nofluffjobsrequirementsSelector     = "#JobOfferRequirements nfj-read-more"
	nofluffjobsresponsibilitiesSelector = "postings-tasks ol li"
	nofluffjobshybridLocationSelector   = "div.popover-body ul li a"
)

// wait times are random (min,max) in seconds
type NoFluffScraper struct {
	minTimeS int
	maxTimeS int
	urls     []string
}

func NewNoFluffScraper(urls []string) *NoFluffScraper {
	return &NoFluffScraper{
		minTimeS: 5,
		maxTimeS: 10,
		urls:     urls,
	}
}

func (*NoFluffScraper) Source() string {
	return "nofluffjobs.com"
}

// extracting data from string html with goquer selectors
func (p *NoFluffScraper) extractDataFromHTML(html string, url string) (scraper.JobOffer, error, bool) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return scraper.JobOffer{}, err, false
	}

	if strings.Contains(html, "Verifying you are human") {
		waitForCaptcha()
		return scraper.JobOffer{}, nil, true
	}

	var job scraper.JobOffer
	job.URL = url
	job.Source = p.Source()
	job.Title = strings.TrimSpace(doc.Find(nofluffjobstitleSelector).Text())

	company := strings.TrimSpace(doc.Find(nofluffjobscompanySelector).Text())
	unwantedDetails := []string{
		"O firmie",
		"About company",
		"About the company",
	}

	for _, u := range unwantedDetails {
		company = strings.TrimSuffix(company, u)
	}

	job.Company = strings.TrimSpace(company)

	//first element is usually an andress
	rawLocation := strings.TrimSpace(doc.Find(nofluffjobslocationSelector).Text())
	// location pin, not always present
	locationPin := doc.Find("[data-cy='location_pin'] span")
	if strings.Contains(rawLocation, "Praca zdalna") {
		job.Location = "Zdalnie"
	} else if strings.Contains(rawLocation, "Hybrydowo") {
		job.Location = "Hybrydowo, "
		// znajduje lokalizacje z pop upa i usuwa słowo "Hybrydowo" które zawsze było przyklejone na końcu bez spacji"
		location := strings.TrimSpace(doc.Find(nofluffjobshybridLocationSelector).Text())
		job.Location += location
	} else {
		job.Location = ""
	}
	if locationPin.Length() > 0 {
		location := strings.TrimSpace(locationPin.First().Text())
		location = strings.ReplaceAll(location, "Hybrydowo", "")
		job.Location += location
	}

	var htmlBuilder strings.Builder

	//description
	descText := strings.TrimSpace(doc.Find(nofluffjobsdescriptionSelector).Text())
	if descText != "" {
		htmlBuilder.WriteString("<p>" + descText + "</p>\n")
	}

	//requirements
	doc.Find(nofluffjobsrequirementsSelector).Each(func(i int, s *goquery.Selection) {
		heading := strings.TrimSpace(s.Find("h2, h3").First().Text())
		if heading != "" {
			htmlBuilder.WriteString("<h2>" + heading + "</h2>\n")
		}

		htmlBuilder.WriteString("<ul>\n")
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				htmlBuilder.WriteString("<li>" + text + "</li>\n")
			}
		})
		htmlBuilder.WriteString("</ul>\n")
	})

	//responsibilities
	doc.Find(nofluffjobsresponsibilitiesSelector).Each(func(i int, s *goquery.Selection) {
		heading := strings.TrimSpace(s.Find("h2, h3").First().Text())
		if heading != "" {
			htmlBuilder.WriteString("<h3>" + heading + "</h3>\n")
		}

		htmlBuilder.WriteString("<ul>\n")
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				htmlBuilder.WriteString("<li>" + text + "</li>\n")
			}
		})
		htmlBuilder.WriteString("</ul>\n")
	})

	job.Description = htmlBuilder.String()

	doc.Find(nofluffjobsskillsSelector).Each(func(_ int, s *goquery.Selection) {
		rawText := strings.ReplaceAll(s.Text(), "Obowiązkowe", "")

		lines := strings.Split(rawText, "\n")

		var result []string
		for _, line := range lines {
			cleaned := strings.TrimSpace(strings.ReplaceAll(line, "\u00a0", " "))

			if cleaned != "" {
				result = append(result, cleaned)
			}
		}
		job.Skills = result
	})

	allSalaries := doc.Find("common-posting-salaries-list div.salary")
	filteredSalaries := allSalaries.Not("[data-cy='JobOffer_SalaryDetails'] div.salary")

	filteredSalaries.Each(func(_ int, s *goquery.Selection) {
		rawAmount := s.Find("h4").Text()
		rawDesc := s.Find(".paragraph").Text()
		lowerDesc := strings.ToLower(rawDesc)

		fullInfo := strings.Join(strings.Fields(strings.ReplaceAll(rawAmount+" "+rawDesc, "\u00a0", " ")), " ")
		fullInfo = strings.ReplaceAll(fullInfo, "oblicz \"na rękę\"", "")
		fullInfo = strings.ReplaceAll(fullInfo, "oblicz netto", "")

		switch {
		case strings.Contains(lowerDesc, "uop") || strings.Contains(lowerDesc, "employment"):
			job.SalaryEmployment = fullInfo

		case strings.Contains(lowerDesc, "uz") || strings.Contains(lowerDesc, "mandate"):
			job.SalaryContract = fullInfo

		case strings.Contains(lowerDesc, "b2b"):
			job.SalaryB2B = fullInfo
		}
	})

	return job, nil, false
}

// html chromedp
func (p *NoFluffScraper) getHTMLContent(chromeDpCtx context.Context, url string) (string, error) {
	var html string
	maxRetries := 3

	for retry := 0; retry < maxRetries; retry++ {
		err := chromedp.Run(
			chromeDpCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
			}),
			chromedp.Navigate(url),
			chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
			chromedp.Evaluate(`Object.defineProperty(navigator, "webdriver", { get: () => false })`, nil),
			chromedp.Sleep(time.Duration(rand.Intn(800)+300)*time.Millisecond),
			chromedp.WaitVisible("body", chromedp.ByQuery),
			chromedp.OuterHTML("html", &html),
		)

		if err == nil {
			return html, nil
		}

		if retry < maxRetries-1 {
			time.Sleep(time.Second)
		}
	}

	return "", fmt.Errorf("Browser crashed 3 times")
}

// main func for scraping
func (p *NoFluffScraper) Scrape(ctx context.Context, q chan<- scraper.JobOffer) error {

	//chromdp config
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(config.BrowserDir),
		chromedp.UserDataDir(config.NofluffDataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
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

	for i := 0; i < len(p.urls); i++ {
		url := p.urls[i]
		html, err := p.getHTMLContent(chromeDpCtx, url)
		if err != nil {
			log.Printf("Chromedp error: %v", err)
			continue
		}

		job, err, captchaAppeared := p.extractDataFromHTML(html, url)
		if captchaAppeared == true {
			time.Sleep(5 * time.Second)
			i--
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case q <- job:
		}

		log.Printf("Scraped %d: %s", i+1, url)
		randomDelay := rand.Intn(p.maxTimeS-p.minTimeS) + p.minTimeS
		log.Printf("Sleeping for: %ds", randomDelay)
		time.Sleep(time.Duration(randomDelay) * time.Second)
	}

	return nil
}
