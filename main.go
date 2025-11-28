package main

import (
	"context"
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pfczx/jobscraper/iternal"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"github.com/pfczx/jobscraper/iternal/scraper/scrapers"
	 "github.com/pyrczuu/urlScraper"
)

func main() {
	db, err := sql.Open("sqlite3", "./database/jobs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//read only for js backend
	_, err = db.Exec("PRAGMA journal_mode=WAL;")

	ctx := context.Background()

	pracujUrls := urlsgocraper.CollectPracujPL()
	pracujScraper := scrapers.NewPracujScraper(pracujUrls)

	scrapersList := []scraper.Scraper{pracujScraper}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		iternal.StartCollector(ctx, db, scrapersList)
	}()

	wg.Wait()
	log.Println("-------------------")
}
