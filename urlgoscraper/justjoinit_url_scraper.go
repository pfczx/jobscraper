package urlsgocraper

import (
	"context"
	"fmt"

	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/pfczx/jobscraper/config"

	//"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

//browser session data dir

const (
	minTimeMs             = 3000
	maxTimeMs             = 4000
	justjoinsource        = "https://justjoin.it/job-offers/" //all-locations/html
	justjoinprefix        = "https://justjoin.it/"
	justjoinofferSelector = "a.offer-card"
)

func getJustJoinJtUrlsFromContent(html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return nil, err
	}

	var urls []string

	doc.Find(justjoinofferSelector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			urls = append(urls, justjoinprefix+href)
		}
	})

	return urls, nil
}

func JustJoinScrollAndRead(parentCtx context.Context) ([]string, error) {
	var urls []string

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(config.BrowserDir),
		chromedp.UserDataDir(config.JustjoinDataDir),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-web-security", true),
		//flags for ANR
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		//chromedp.Flag("disable-gpu-compositing", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	//justjoin scraping often crashes, defer for rescuing data
	defer func() {
		urls = UniqueSliceElements(urls)
	}()

	log.Println("JUSTJOINIT: Uruchamianie przeglądarki...")

	_ = chromedp.Run(chromeDpCtx,

		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(justjoinsource),
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			var prevHeight int64 = -99
			var currentHeight int64
			var html string

			log.Println("JUSTJOINIT: Strona załadowana. Rozpoczynanie pętli wewnętrznej...")

			sameHeightCount := 0
			for i := 1; ; i++ {
				err := chromedp.Evaluate(`document.body.scrollHeight`, &currentHeight).Do(ctx)
				if err != nil {
					return fmt.Errorf("błąd pobierania wysokości: %w", err)
				}

				if currentHeight == prevHeight {
					sameHeightCount++
				} else {
					sameHeightCount = 0
				}

				if sameHeightCount > 10 {
					log.Printf("JUSTJOINIT: Koniec strony, wysokość (%d).", currentHeight)
					break
				}

				if err := chromedp.OuterHTML("html", &html).Do(ctx); err != nil {
					log.Printf("Błąd odczytu HTML: %v", err)
				} else {
					collected, err := getJustJoinJtUrlsFromContent(html)
					if err == nil {
						urls = append(urls, collected...)
						log.Printf("JUSTJOINIT: Iteracja %d: Znaleziono %d linków (razem: %d)", i, len(collected), len(urls))
					}
				}

				prevHeight = currentHeight
				log.Printf("JUSTJOINIT: Scrollowanie do: %d", currentHeight)

				randomDelay := rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}

				err = chromedp.Evaluate(`window.scrollBy(0, 1400);`, nil).Do(ctx)
				if err != nil {
					return err
				}
				if err != nil {
					return err
				}

				randomDelay = rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
	)

	urls = UniqueSliceElements(urls)
	log.Printf("JUSTJOINIT: Usunięto duplikaty, %v unikalnych linków", len(urls))
	return urls, nil
}
