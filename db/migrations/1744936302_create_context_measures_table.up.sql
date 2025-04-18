CREATE TABLE
  IF NOT EXISTS context_measures (
    id INTEGER PRIMARY KEY UNIQUE,
    measure TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE TRIGGER IF NOT EXISTS update_context_measures AFTER
UPDATE ON context_measures FOR EACH ROW BEGIN
UPDATE context_measures
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;

INSERT OR IGNORE INTO
  context_measures (measure)
VALUES
  ("FGM"),
  ("FGA"),
  ("FG_PCT"),
  ("FG3M"),
  ("FG3A"),
  ("FG3_PCT"),
  ("FTM"),
  ("FTA"),
  ("OREB"),
  ("DREB"),
  ("AST"),
  ("FGM_AST"),
  ("FG3_AST"),
  ("STL"),
  ("BLK"),
  ("BLKA"),
  ("TOV"),
  ("PF"),
  ("PFD"),
  ("POSS_END_FT"),
  ("PTS_PAINT"),
  ("PTS_FB"),
  ("PTS_OFF_TOV"),
  ("PTS_2ND_CHANCE"),
  ("REB"),
  ("TM_FGM"),
  ("TM_FGA"),
  ("TM_FG3M"),
  ("TM_FG3A"),
  ("TM_FTM"),
  ("TM_FTA"),
  ("TM_OREB"),
  ("TM_DREB"),
  ("TM_REB"),
  ("TM_TEAM_REB"),
  ("TM_AST"),
  ("TM_STL"),
  ("TM_BLK"),
  ("TM_BLKA"),
  ("TM_TOV"),
  ("TM_TEAM_TOV"),
  ("TM_PF"),
  ("TM_PFD"),
  ("TM_PTS"),
  ("TM_PTS_PAINT"),
  ("TM_PTS_FB"),
  ("TM_PTS_OFF_TOV"),
  ("TM_PTS_2ND_CHANCE"),
  ("TM_FGM_AST"),
  ("TM_FG3_AST"),
  ("TM_POSS_END_FT"),
  ("OPP_FGM"),
  ("OPP_FGA"),
  ("OPP_FG3M"),
  ("OPP_FG3A"),
  ("OPP_FTM"),
  ("OPP_FTA"),
  ("OPP_OREB"),
  ("OPP_DREB"),
  ("OPP_REB"),
  ("OPP_TEAM_REB"),
  ("OPP_AST"),
  ("OPP_STL"),
  ("OPP_BLK"),
  ("OPP_BLKA"),
  ("OPP_TOV"),
  ("OPP_TEAM_TOV"),
  ("OPP_PF"),
  ("OPP_PFD"),
  ("OPP_PTS"),
  ("OPP_PTS_PAINT"),
  ("OPP_PTS_FB"),
  ("OPP_PTS_OFF_TOV"),
  ("OPP_PTS_2ND_CHANCE"),
  ("OPP_FGM_AST"),
  ("OPP_FG3_AST"),
  ("OPP_POSS_END_FT"),
  ("PTS");