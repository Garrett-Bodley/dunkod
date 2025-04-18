CREATE TABLE
  IF NOT EXISTS assets (
    id INTEGER PRIMARY KEY UNIQUE,
    asset_description TEXT NOT NULL,
    asset_url TEXT NOT NULL UNIQUE,
    is_dunk BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE INDEX IF NOT EXISTS is_dunk ON assets (is_dunk);

CREATE TRIGGER IF NOT EXISTS update_assets AFTER
UPDATE ON assets FOR EACH ROW BEGIN
UPDATE assets
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;