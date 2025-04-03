CREATE TABLE
  IF NOT EXISTS players_games_teams_seasons (
    id INTEGER PRIMARY KEY UNIQUE,
    player_id INT NOT NULL,
    game_id TEXT NOT NULL,
    team_id INT NOT NULL,
    season TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    FOREIGN KEY (player_id) REFERENCES players (id),
    FOREIGN KEY (game_id) REFERENCES games (id),
    FOREIGN KEY (team_id) REFERENCES teams (id)
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_game_team ON players_games_teams_seasons (player_id, game_id, team_id, season);

CREATE TRIGGER IF NOT EXISTS update_players_games_teams_seasons_modtime AFTER
UPDATE ON players_games_teams_seasons FOR EACH ROW BEGIN
UPDATE players_games_teams_seasons
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;