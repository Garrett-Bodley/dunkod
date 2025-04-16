CREATE TABLE
  IF NOT EXISTS box_score_scraping_errors (
    id INTEGER PRIMARY KEY UNIQUE,
    game_id TEXT NOT NULL,
    error_details TEXT NOT NULL,
    error_status TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    FOREIGN KEY (game_id) REFERENCES games (id)
  );

CREATE INDEX IF NOT EXISTS idx_game_id ON box_score_scraping_errors (game_id);

CREATE TRIGGER IF NOT EXISTS update_box_score_scraping_errors_modtime AFTER
UPDATE ON box_score_scraping_errors FOR EACH ROW BEGIN
UPDATE box_score_scraping_errors
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;