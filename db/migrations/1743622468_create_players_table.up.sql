CREATE TABLE
  IF NOT EXISTS players (
    id INTEGER PRIMARY KEY UNIQUE,
    player_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE TRIGGER IF NOT EXISTS update_players_modtime AFTER
UPDATE ON players FOR EACH ROW BEGIN
UPDATE players
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;