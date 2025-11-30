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
	browserDataDir=`/home/devpad/.config/google-chrome-canary/Default`
)

//selectors
const (
	titleSelector         = "h1[data-scroll-id='job-title']"
	companySelector       = "h2[data-scroll-id='employer-name']"
	locationSelector      = "div[data-test='offer-badge-title']"
	descriptionSelector   = `ul[data-test="text-about-project"]`                                                         //concat in code
	skillsSelector        = `span[data-test="item-technologies-expected"], span[data-test="item-technologies-optional"]` //concat in code
	salarySectionSelector = `div[data-test="section-salaryPerContractType"]`
	addidtionalInfoSelector = `#offer-details section`
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
	job.Company = strings.TrimSpace(doc.Find(companySelector).Text())
	job.Location = strings.TrimSpace(doc.Find(locationSelector).Text())

	doc.Find(descriptionSelector).Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			job.Description += t + "\n"
		}
	})


	doc.Find(addidtionalInfoSelector).Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		if text != "" {
			if !strings.Contains(text, "text-about-project") {
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						job.Description += "• " + line + "\n"
					}
				}
			}
		}
	})

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
		case strings.Contains(lower, "umowa o pracę"):
			job.SalaryEmployment = text
		case strings.Contains(lower, "umowa zlecenie"):
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
