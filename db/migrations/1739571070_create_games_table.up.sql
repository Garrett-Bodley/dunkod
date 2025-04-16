CREATE TABLE
  IF NOT EXISTS games (
    id TEXT PRIMARY KEY UNIQUE,
    season TEXT NOT NULL,
    game_date TEXT NOT NULL,
    matchup TEXT NOT NULL,
    season_type TEXT NOT NULL,
    winner_name TEXT NOT NULL,
    winner_id INTEGER,
    winner_score INTEGER NOT NULL,
    loser_name TEXT NOT NULL,
    loser_id INTEGER,
    loser_score INTEGER NOT NULL,
    home_team_id INTEGER,
    away_team_id INTEGER,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    FOREIGN KEY (winner_id) REFERENCES teams (id),
    FOREIGN KEY (loser_id) REFERENCES teams (id),
    FOREIGN KEY (home_team_id) REFERENCES teams (id),
    FOREIGN KEY (away_team_id) REFERENCES teams (id)
  );

CREATE INDEX IF NOT EXISTS idx_home_team_id ON games (home_team_id);

CREATE INDEX IF NOT EXISTS idx_away_team_id ON games (away_team_id);

CREATE INDEX IF NOT EXISTS idx_winner_id ON games (winner_id);

CREATE INDEX IF NOT EXISTS idx_loser_id ON games (loser_id);

CREATE TRIGGER IF NOT EXISTS update_games_modtime AFTER
UPDATE ON games FOR EACH ROW BEGIN
UPDATE games
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;