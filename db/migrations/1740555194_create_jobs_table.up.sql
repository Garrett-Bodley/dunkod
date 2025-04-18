CREATE TABLE
  IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY UNIQUE,
    players TEXT NOT NULL,
    games TEXT NOT NULL,
    season TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    job_state TEXT NOT NULL DEFAULT "PENDING",
    job_hash TEXT UNIQUE NOT NULL,
    error_details TEXT,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime'))
  );

CREATE TRIGGER IF NOT EXISTS update_jobs_modtime AFTER
UPDATE ON jobs FOR EACH ROW BEGIN
UPDATE jobs
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;