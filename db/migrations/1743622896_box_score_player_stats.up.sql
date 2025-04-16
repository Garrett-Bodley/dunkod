CREATE TABLE
  IF NOT EXISTS box_score_player_stats (
    id INTEGER PRIMARY KEY UNIQUE,
    player_id INT NOT NULL,
    team_id INT NOT NULL,
    game_id TEXT NOT NULL,
    season TEXT NOT NULL,
    dnp BOOLEAN NOT NULL,
    min TEXT,
    fgm REAL,
    fga REAL,
    fg_pct REAL,
    fg3m REAL,
    fg3a REAL,
    fg3_pct REAL,
    ftm REAL,
    fta REAL,
    ft_pct REAL,
    oreb REAL,
    dreb REAL,
    reb REAL,
    ast REAL,
    stl REAL,
    blk REAL,
    tov REAL,
    pf REAL,
    pts REAL,
    plus_minus REAL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    FOREIGN KEY (player_id) REFERENCES players (id),
    FOREIGN KEY (game_id) REFERENCES games (id),
    FOREIGN KEY (team_id) REFERENCES teams (id)
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_game_team ON box_score_player_stats (player_id, game_id, team_id, season);

CREATE INDEX IF NOT EXISTS idx_player_id ON box_score_player_stats (player_id);

CREATE INDEX IF NOT EXISTS idx_game_id ON box_score_player_stats (game_id);

CREATE INDEX IF NOT EXISTS idx_team_id ON box_score_player_stats (team_id);

CREATE TRIGGER IF NOT EXISTS update_box_score_player_stats AFTER
UPDATE ON box_score_player_stats FOR EACH ROW BEGIN
UPDATE box_score_player_stats
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;