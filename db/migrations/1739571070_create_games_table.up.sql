CREATE TABLE
  IF NOT EXISTS games (
    id TEXT PRIMARY KEY UNIQUE,
    season TEXT,
    game_date TEXT,
    matchup TEXT,
    season_type TEXT,
    winner_name TEXT,
    winner_id INTEGER REFERENCES teams (id),
    winner_score INTEGER,
    loser_name TEXT,
    loser_id INTEGER REFERENCES teams (id),
    loser_score INTEGER,
    home_team_id INTEGER REFERENCES teams (id),
    away_team_id INTEGER REFERENCES teams (id)
  )