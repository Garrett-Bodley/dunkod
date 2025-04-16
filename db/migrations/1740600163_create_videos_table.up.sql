CREATE TABLE
  IF NOT EXISTS videos (
    id INTEGER PRIMARY KEY UNIQUE,
    title TEXT NOT NULL,
    video_description TEXT NOT NULL,
    youtube_url TEXT NOT NULL,
    job_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    updated_at TIMESTAMP DEFAULT (datetime ('now', 'localtime')),
    FOREIGN KEY (job_id) REFERENCES jobs (id)
  );

CREATE INDEX IF NOT EXISTS idx_job_id ON videos(job_id);

CREATE TRIGGER IF NOT EXISTS update_videos_modtime AFTER
UPDATE ON videos FOR EACH ROW BEGIN
UPDATE videos
SET
  updated_at = datetime ('now', 'localtime')
WHERE
  id = NEW.id;

END;