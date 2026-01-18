package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pfczx/jobscraper/iternal"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"github.com/pfczx/jobscraper/iternal/scraper/scrapers"

	//"github.com/pyrczuu/urlScraper"
	//"github.com/pyrczuu/nofluff_scraper"
	"github.com/pfczx/jobscraper/urlgoscraper"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	db, err := sql.Open("sqlite3", "./database/jobs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var wg sync.WaitGroup

	ctx := context.Background()

	for {
		fmt.Println("1 - urlscraping")
		fmt.Println("2 - job scraping from url list")
		fmt.Println("3 - exit")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			wg.Add(3)
			go func() {
				defer wg.Done()
				pracujUrls := urlsgocraper.CollectPracujPl(ctx)
				urlsgocraper.SaveUrls("pracujUrls.txt", pracujUrls)
			}()

			go func() {
				defer wg.Done()
				noflufUrls, _ := urlsgocraper.NofluffScrollAndRead(ctx)
				urlsgocraper.SaveUrls("noflufUrls.txt", noflufUrls)
			}()

			go func() {
				defer wg.Done()
				justjoinUrls, _ := urlsgocraper.JustJoinScrollAndRead(ctx)
				urlsgocraper.SaveUrls("justjoinUrls.txt", justjoinUrls)
			}()

			wg.Wait()

		case "2":
			pracujUrls, _ := urlsgocraper.LoadUrls("pracujUrls.txt")
			noflufUrls, _ := urlsgocraper.LoadUrls("noflufUrls.txt")
			justjoinUrls, _ := urlsgocraper.LoadUrls("justjoinUrls.txt")

			scrapersList := []scraper.Scraper{
				scrapers.NewNoFluffScraper(noflufUrls),
				scrapers.NewPracujScraper(pracujUrls),
				scrapers.NewJustJoinItScraper(justjoinUrls),
			}
			parralel := false
			fmt.Println("Type y/yes if u want parralel scraping")
			choiceParralel, _ := reader.ReadString('\n')
			choiceParralel = strings.TrimSpace(choiceParralel)
			if choiceParralel == "y" || choiceParralel =="yes"{
				parralel = true
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				iternal.StartCollector(ctx, db, scrapersList, parralel)
			}()

			wg.Wait()
			log.Println("Scraping Completed")
		case "3":
			os.Exit(0)
		}

	}

}
