-- +goose Up 
ALTER TABLE job_offers ADD COLUMN embedding BLOB;

-- +goose Down
ALTER TABLE job_offers DROP COLUMN embedding BLOB;
