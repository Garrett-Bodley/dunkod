CREATE TABLE
  IF NOT EXISTS games (
    id TEXT PRIMARY KEY UNIQUE,
    season TEXT NOT NULL,
    game_date TEXT NOT NULL,
    matchup TEXT NOT NULL,
    season_type TEXT NOT NULL,
    winner_name TEXT NOT NULL,
    winner_id INTEGER REFERENCES teams (id),
    winner_score INTEGER NOT NULL,
    loser_name TEXT NOT NULL,
    loser_id INTEGER REFERENCES teams (id),
    loser_score INTEGER NOT NULL,
    home_team_id INTEGER REFERENCES teams (id),
    away_team_id INTEGER REFERENCES teams (id),
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE TRIGGER IF NOT EXISTS update_games_modtime AFTER
UPDATE ON games FOR EACH ROW BEGIN
UPDATE games
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;