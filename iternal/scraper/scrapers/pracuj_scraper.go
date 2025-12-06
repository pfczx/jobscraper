package scrapers

import (
	"bufio"
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

var proxyList = []string{
	"213.73.25.231:8080",
}

//browser session data dir

const (
	browserDataDir = `/home/devpad/.config/google-chrome-canary/Default`
)

// selectors
const (
	titleSelector           = `h1[data-test="text-positionName"]`
	companySelector         = `h2[data-scroll-id='employer-name']`
	locationSelector        = `#offer-details li`
	descriptionSelector     = `ul[data-test="text-about-project"]`                                                         //concat in code
	skillsSelector          = `span[data-test="item-technologies-expected"], span[data-test="item-technologies-optional"]` //concat in code
	salarySectionSelector   = `div[data-test="section-salaryPerContractType"]`
	requirementsSelector   = `section[data-test="section-requirements"]`
	responsibilitiesSelector =`section[data-test="section-responsibilities"]`
)
// wait times are random (min,max) in seconds
type PracujScraper struct {
	minTimeS int
	maxTimeS int
	urls     []string
}

func NewPracujScraper(urls []string) *PracujScraper {
	return &PracujScraper{
		minTimeS: 5,
		maxTimeS: 10,
		urls:     urls,
	}
}

func (*PracujScraper) Source() string {
	return "pracuj.pl"
}

func waitForCaptcha() {
	log.Println("Cloudflare detected, solve and press enter")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadBytes('\n')
}

// extracting data from string html with goquer selectors
func (p *PracujScraper) extractDataFromHTML(html string, url string) (scraper.JobOffer, error, bool) {
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
	job.Title = strings.TrimSpace(doc.Find(titleSelector).Text())

	company := strings.TrimSpace(doc.Find(companySelector).Text())
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
	job.Location = strings.TrimSpace(doc.Find(locationSelector).First().Find(`div[data-test="offer-badge-title"]`).Text())
	job.Location += ", "
	doc.Find(locationSelector).Each(func(i int, li *goquery.Selection) {
		value := strings.ToLower(strings.TrimSpace(li.Find(`div[data-test="offer-badge-title"]`).Text()))

		if !strings.Contains(value, "zaraz") && (strings.Contains(value, "miejsce pracy") ||
			strings.Contains(value, "workplace") ||
			strings.Contains(value, "location") ||
			strings.Contains(value, "lokalizacja") ||
			strings.Contains(value, "office") ||
			strings.Contains(value, "hybrid") ||
			strings.Contains(value, "hybryd") ||
			strings.Contains(value, "remote") ||
			strings.Contains(value, "Company location") ||
			strings.Contains(value, "praca") ||
			strings.Contains(value, "work") ||
			strings.Contains(value, "zdal")) {
			job.Location += value + ", "
		}

	})

	var htmlBuilder strings.Builder

	//description
	descText := strings.TrimSpace(doc.Find(descriptionSelector).Text())	
	if descText != "" {
		htmlBuilder.WriteString("<p>" + descText + "</p>\n")
	}

	//requirements 
	doc.Find(requirementsSelector).Each(func(i int, s *goquery.Selection) {
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
	doc.Find(responsibilitiesSelector).Each(func(i int, s *goquery.Selection) {
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

	doc.Find(skillsSelector).Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			job.Skills = append(job.Skills, t)
		}
	})
	doc.Find(salarySectionSelector).Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		text = strings.ReplaceAll(text, "\u00A0", " ")

		lower := strings.ToLower(text)
		switch {
		case strings.Contains(lower, "prac") || strings.Contains(lower,"employment"):
			job.SalaryEmployment = text
		case strings.Contains(lower, "zlec") || strings.Contains(lower,"mandate"):
			job.SalaryContract = text
		case strings.Contains(lower, "b2b"):
			job.SalaryB2B = text
		}
	})

	return job, nil, false
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
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.Evaluate(`Object.defineProperty(navigator, "webdriver", { get: () => false })`, nil),
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
		chromedp.ExecPath("/usr/bin/google-chrome"),
		chromedp.UserDataDir(browserDataDir),
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
