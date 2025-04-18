CREATE TABLE
  IF NOT EXISTS assets_measures (
    id INTEGER PRIMARY KEY UNIQUE,
    context_measure_id INTEGER NOT NULL,
    asset_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE INDEX IF NOT EXISTS idx_context_measure_id ON assets_measures (context_measure_id);

CREATE INDEX IF NOT EXISTS idx_asset_id ON assets_measures (asset_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_measure_unique ON assets_measures (context_measure_id, asset_id);

CREATE TRIGGER IF NOT EXISTS update_assets_measures AFTER
UPDATE ON assets_measures FOR EACH ROW BEGIN
UPDATE assets_measures
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;