CREATE TABLE
  IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY UNIQUE,
    team_name TEXT NOT NULL,
    city TEXT NOT NULL,
    abbreviation TEXT NOT NULL,
    conference TEXT NOT NULL,
    division TEXT NOT NULL,
    code TEXT NOT NULL,
    slug TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE TRIGGER IF NOT EXISTS update_teams_modtime AFTER
UPDATE ON teams FOR EACH ROW BEGIN
UPDATE teams
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;