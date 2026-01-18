package urlsgocraper

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/config"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

//browser session data dir

const (
	minTimeS = 5
	maxTimeS = 10
)

func getHTMLContent(chromeDpCtx context.Context, url string) (string, error) {
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
	if err != nil {
		log.Printf("Error %s", err)
	}

	return html, err
}

func getUrlsFromContent(html, selector string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return nil, err
	}

	var urls []string

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			urls = append(urls, href)
		}
	})

	return urls, nil
}

func getMaxPagePracujPl(html string) (int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return -1, err
	}

	pageNumSelector := "span[data-test=\"top-pagination-max-page-number\"]"

	maxPage := doc.Find(pageNumSelector).Text()
	maxPageNum, _ := strconv.Atoi(strings.TrimSpace(maxPage))

	//testing
	//return 1,nil
	return maxPageNum, nil
}

func CollectPracujPl(ctx context.Context) []string {
	source := "https://it.pracuj.pl/praca" //?its=agile
	urlsSelector := "[data-test=\"link-offer\"]"
	var urls []string

	//chromdp config
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(config.BrowserDir),
		chromedp.UserDataDir(config.PracujDataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		//chromedp.Flag("proxy-server", proxyList[rand.Intn(len(proxyList))]),

		//flags for ANR
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.Flag("disable-background-timer-throttling", true), 
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-gpu-compositing", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	html, err := getHTMLContent(chromeDpCtx, source)
	if err != nil {
		log.Printf("Error getting HTML content")
	}

	maxPage, err := getMaxPagePracujPl(html)
	if err != nil {
		log.Printf("Error getting max page")
	}

	firstPageUrls, err := getUrlsFromContent(html, urlsSelector)
	if err != nil {
		log.Printf("Error getting main page urls")
	} else {
		urls = append(urls, firstPageUrls...)
		log.Printf("Scraped first page")
	}
	//for testing
	//return firstPageUrls[:20]

	if maxPage > 2 {
		for i := 2; i < maxPage; i++ {
			html, err = getHTMLContent(chromeDpCtx, source+"?pn="+strconv.Itoa(i))
			if err != nil {
				log.Printf("Error {%v} while getting HTML content on page: %v", err, i)
			}
			freshUrls, err1 := getUrlsFromContent(html, urlsSelector)
			if err1 != nil {
				log.Printf("Error {%v} while getting urls on page: %v", err, i)
			} else {
				urls = append(urls, freshUrls...)
				log.Printf("Scraped page number: %v", i)

			}
			randomDelay := rand.Intn(maxTimeS-minTimeS) + minTimeS
			log.Printf("Sleeping for: %ds", randomDelay)
			time.Sleep(time.Duration(randomDelay) * time.Second)
		}
	}
	log.Printf("Collected: %d urls", len(urls))
	return urls
}

/*
func collectURLs(ctx context.Context, url string, selector string) []string {
	var html string

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/chromium-browser"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-site-isolation-trials", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	// Pobierz pełny HTML
	err := chromedp.Run(
		chromeDpCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(url),
		chromedp.Sleep(time.Duration(rand.Intn(10)+20)*time.Second),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)

	if err != nil {
		log.Printf("Chromedp error: %s", err)
	}

	// Parsowanie HTML goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Fatal("Goquery error:", err)
	}

	var collected []string

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			collected = append(collected, href)
		}
	})
	log.Printf("Found: %d links",len(collected))

	return collected
}

func CollectNoFluffJobs(ctx context.Context) []string {
	urls := collectURLs(ctx,"https://nofluffjobs.com/pl/artificial-intelligence?criteria=category%3Dsys-administrator,business-analyst,architecture,backend,data,ux,devops,erp,embedded,frontend,fullstack,game-dev,mobile,project-manager,security,support,testing,other", "a.posting-list-item")
	var formatted []string
	for _, url := range urls {
		formatted = append(formatted, "https://nofluffjobs.com"+url)
	}
	return formatted
}

func CollectJustJoinIT(ctx context.Context) []string {
	urls := collectURLs(ctx,"https://justjoin.it/", "a.offer-card")
	var formatted []string
	for _, url := range urls {
		formatted = append(formatted, "https://justjoin.it"+url)
	}
	return formatted
}

func CollectPracujPL(ctx context.Context) []string {
	return collectURLs(ctx,"https://it.pracuj.pl/praca", "[data-test=\"link-offer\"]")
}
*/
