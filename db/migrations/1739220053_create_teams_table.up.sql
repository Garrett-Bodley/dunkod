CREATE TABLE
  IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY UNIQUE,
    name TEXT NOT NULL,
    city TEXT,
    abbreviation TEXT,
    conference TEXT,
    division TEXT,
    code TEXT,
    slug TEXT,
    min_year INT,
    max_year INT,
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

INSERT
OR IGNORE INTO teams (id, name)
VALUES
  (1610612737, 'Atlanta Hawks'),
  (1610612738, 'Boston Celtics'),
  (1610612751, 'Brooklyn Nets'),
  (1610612766, 'Charlotte Hornets'),
  (1610612741, 'Chicago Bulls'),
  (1610612739, 'Cleveland Cavaliers'),
  (1610612742, 'Dallas Mavericks'),
  (1610612743, 'Denver Nuggets'),
  (1610612765, 'Detroit Pistons'),
  (1610612744, 'Golden State Warriors'),
  (1610612745, 'Houston Rockets'),
  (1610612754, 'Indiana Pacers'),
  (1610612746, 'Los Angeles Clippers'),
  (1610612747, 'Los Angeles Lakers'),
  (1610612763, 'Memphis Grizzlies'),
  (1610612748, 'Miami Heat'),
  (1610612749, 'Milwaukee Bucks'),
  (1610612750, 'Minnesota Timberwolves'),
  (1610612740, 'New Orleans Pelicans'),
  (1610612752, 'New York Knicks'),
  (1610612760, 'Oklahoma City Thunder'),
  (1610612753, 'Orlando Magic'),
  (1610612755, 'Philadelphia 76ers'),
  (1610612756, 'Phoenix Suns'),
  (1610612757, 'Portland Trail Blazers'),
  (1610612758, 'Sacramento Kings'),
  (1610612759, 'San Antonio Spurs'),
  (1610612761, 'Toronto Raptors'),
  (1610612762, 'Utah Jazz'),
  (1610612764, 'Washington Wizards'),
  (0, 'NULL_TEAM');