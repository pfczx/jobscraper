package iternal

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pfczx/jobscraper/database"
	"github.com/pfczx/jobscraper/iternal/scraper"
	"log"
	"time"
)

func StartCollector(ctx context.Context, db *sql.DB, scrapers []scraper.Scraper,parallel bool) {
	out := scraper.RunScrapers(ctx, scrapers,parallel)
	querier := database.New(db)

	for job := range out {
		log.Printf("Saving job: %s from %s", job.Title, job.Company)
		skillsJSON, _ := json.Marshal(job.Skills)
		params := database.UpsertJobOfferParams{
			ID:               uuid.New().String(),
			Title:            job.Title,
			Company:          sql.NullString{String: job.Company, Valid: job.Company != ""},
			Location:         sql.NullString{String: job.Location, Valid: job.Location != ""},
			Description:      sql.NullString{String: job.Description, Valid: job.Description != ""},
			Url:              urlNormalizer(job.URL),
			Source:           job.Source,
			PublishedAt:      sql.NullTime{Time: time.Now(), Valid: true},
			Skills:           sql.NullString{String: string(skillsJSON), Valid: len(job.Skills) > 0},
			SalaryEmployment: sql.NullString{String: job.SalaryEmployment, Valid: job.SalaryEmployment != ""},
			SalaryB2b:        sql.NullString{String: job.SalaryB2B, Valid: job.SalaryB2B != ""},
			SalaryContract:   sql.NullString{String: job.SalaryContract, Valid: job.SalaryContract != ""},
		}
		if _, err := querier.UpsertJobOffer(ctx, params); err != nil {
			log.Printf("Error %s in saving: %s from %s",err, job.Title, job.Company)
		}
	}
}
