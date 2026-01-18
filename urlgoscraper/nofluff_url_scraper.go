package urlsgocraper

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/pfczx/jobscraper/config"
)

const (
	// tylko do testow
	nofluffsource = "https://nofluffjobs.com/pl/" //JavaScript?criteria=requirement%3DUML
	nofluffprefix = "https://nofluffjobs.com"
	//nofluffsource        = "https://nofluffjobs.com/pl/artificial-intelligence?criteria=category%3Dsys-administrator,business-analyst,architecture,backend,data,ux,devops,erp,embedded,frontend,fullstack,game-dev,mobile,project-manager,security,support,testing,other"
	nofluffofferSelector = "a.posting-list-item"
	//nofluffcookiesButtonSelector = "button#save"                                // zamknięcie cookies
	//noflufloginButtonSelector    = "button[.//inline-icon[@maticon=\"close\"]]" // zamknięcie prośby o zalogowanie
	nofluffloadMoreSelector = "button[nfjloadmore]"
)

func UniqueSliceElements[T comparable](inputSlice []T) []T {
	uniqueSlice := make([]T, 0, len(inputSlice))
	seen := make(map[T]bool, len(inputSlice))
	for _, element := range inputSlice {
		if !seen[element] {
			uniqueSlice = append(uniqueSlice, element)
			seen[element] = true
		}
	}
	return uniqueSlice
}

func getNoFluffUrlsFromContent(html string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("goquery parse error: %v", err)
		return nil, err
	}

	var urls []string

	doc.Find(nofluffofferSelector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			urls = append(urls, nofluffprefix+href)
		}
	})

	return urls, nil
}

func NofluffScrollAndRead(parentCtx context.Context) ([]string, error) {
	var urls []string

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(config.BrowserDir),
		chromedp.UserDataDir(config.NofluffDataDir),
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
		chromedp.Flag("disable-gpu-compositing", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parentCtx, opts...)
	defer cancelAlloc()

	chromeDpCtx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	log.Println("NOFLUFFJOBS: Uruchamianie przeglądarki...")

	var html string

	err := chromedp.Run(chromeDpCtx,

		chromedp.ActionFunc(func(ctx context.Context) error {
			return emulation.SetDeviceMetricsOverride(1280, 900, 1.0, false).Do(ctx)
		}),
		chromedp.Navigate(nofluffsource),
		chromedp.Evaluate(`delete navigator.__proto__.webdriver`, nil),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		//klika wymagane cookies jeśli jest komunikat, blokuje program jeśli ich nie ma :/
		//chromedp.Click(
		//	cookiesButtonSelector,
		//	chromedp.NodeVisible,
		//),
		//chromedp.Click(
		//	loginButtonSelector,
		//	chromedp.NodeVisible,
		//),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("NOFLUFFJOBS: Strona załadowana. Rozpoczynanie pętli wewnętrznej...")
			var nodes []*cdp.Node

			for i := 1; ; i++ {
				log.Printf("NOFLUFFJOBS: Iteracja: %v", i)
				randomDelay := rand.Intn(maxTimeMs-minTimeMs) + minTimeMs
				err := chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}

				err = chromedp.Nodes(nofluffloadMoreSelector, &nodes, chromedp.AtLeast(0)).Do(ctx)
				if err != nil {
					return err
				}
				if len(nodes) == 0 {
					break
				}

				err = chromedp.Click(nofluffloadMoreSelector).Do(ctx)
				if err != nil {
					return err
				}

				randomDelay = rand.Intn((maxTimeMs+10*i)-(minTimeMs+10*i)) + (minTimeMs + 10*i) // czym więcej kontentu (kolejne iteracje) tym dłużej czekamy (wolniejsza strona)
				err = chromedp.Sleep(time.Duration(randomDelay) * time.Millisecond).Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		chromedp.OuterHTML("html", &html),
	)
	urls, err = getNoFluffUrlsFromContent(html)
	urls = UniqueSliceElements(urls)
	log.Printf("NOFLUFFJOBS: Usunięto duplikaty, %v unikalnych linków", len(urls))
	if err != nil {
		log.Println("Błąd wyciąganie url z kontentu")
		return nil, err
	}

	return urls, nil
}
