package db

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"database/sql"
	"dunkod/config"
	"dunkod/utils"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

func GetDb() *sqlx.DB {
	return db
}

func Close() error {
	return db.Close()
}

func SetupDatabase() error {
	if config.DatabaseFile == "" {
		panic("config.DatabaseFile value is not initialized")
	}
	_, err := os.Stat(config.DatabaseFile)
	if os.IsNotExist(err) {
		log.Println("Database file not found. Creating a new database.")
		file, err := os.Create(config.DatabaseFile)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		file.Close()
	} else if err != nil {
		return utils.ErrorWithTrace(err)
	}
	db, err = sqlx.Connect("sqlite3", config.DatabaseFile+"?_journal_mode=WAL&_sync=NORMAL&_fk=true&_busy_timeout=5000")
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func RunMigrations() error {
	m, err := migrate.New(
		"file://db/migrations",
		"sqlite3://"+config.DatabaseFile,
	)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return utils.ErrorWithTrace(err)
	}
	if sourceErr, destErr := m.Close(); sourceErr != nil || destErr != nil {
		return utils.ErrorWithTrace(errors.Join(sourceErr, destErr))
	}
	return nil
}

func ValidateMigrations() error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count); err != nil {
		return utils.ErrorWithTrace(err)
	}

	if count != 31 {
		return utils.ErrorWithTrace(fmt.Errorf("expected 31 teams, found %d", count))
	}

	var name string
	if err := db.QueryRow("SELECT name FROM teams WHERE id = 1610612752").Scan(&name); err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("failed to find Knicks: %v", err))
	}
	if name != "New York Knicks" {
		return utils.ErrorWithTrace(fmt.Errorf("expected team.id 1610612752 to have name 'New York Knicks', got '%s'", name))
	}
	err := db.QueryRow("SELECT name FROM teams WHERE id = 0").Scan(&name)
	if err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("faild to find NULL_TEAM: %v", err))
	}
	if name != "NULL_TEAM" {
		return utils.ErrorWithTrace(fmt.Errorf("expected team.id 0 to have name 'NULL_TEAM', got '%s'", name))
	}
	return nil
}

type DatabaseGame struct {
	ID          string    `db:"id"`
	Season      string    `db:"season"`
	GameDate    string    `db:"game_date"`
	Matchup     string    `db:"matchup"`
	SeasonType  string    `db:"season_type"`
	WinnerName  string    `db:"winner_name"`
	WinnerID    int       `db:"winner_id"`
	WinnerScore int       `db:"winner_score"`
	LoserName   string    `db:"loser_name"`
	LoserID     int       `db:"loser_id"`
	LoserScore  int       `db:"loser_score"`
	HomeTeamId  int       `db:"home_team_id"`
	AwayTeamId  int       `db:"away_team_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (g DatabaseGame) ToString() string {
	dateTime, err := time.Parse("2006-01-02", g.GameDate)
	if err != nil {
		panic(err)
	}
	dateString := dateTime.Format("1/2/06")

	splitWinner := strings.Split(g.WinnerName, " ")
	winnerString := splitWinner[len(splitWinner)-1]
	if winnerString == "Timberwolves" {
		winnerString = "Wolves"
	} else if winnerString == "Mavericks" {
		winnerString = "Mavs"
	}

	splitLoser := strings.Split(g.LoserName, " ")
	loserString := splitLoser[len(splitLoser)-1]
	if loserString == "Timberwolves" {
		loserString = "Wolves"
	} else if loserString == "Mavericks" {
		loserString = "Mavs"
	}

	if g.WinnerID == g.HomeTeamId {
		return fmt.Sprintf("%s (%d) vs %s (%d) %s",
			winnerString,
			g.WinnerScore,
			loserString,
			g.LoserScore,
			dateString,
		)
	} else {
		return fmt.Sprintf("%s (%d) @ %s (%d) %s",
			winnerString,
			g.WinnerScore,
			loserString,
			g.LoserScore,
			dateString,
		)
	}
}

func InsertGames(games []DatabaseGame) error {
	tx, err := db.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	query := `
		REPLACE INTO games (
			id, season, game_date, matchup, season_type, winner_name,
			winner_id, winner_score, loser_name, loser_id, loser_score,
			home_team_id, away_team_id
		) VALUES (
			:id, :season, :game_date, :matchup, :season_type, :winner_name,
			:winner_id, :winner_score, :loser_name, :loser_id, :loser_score,
			:home_team_id, :away_team_id
		)
	`
	batch_size := 500
	for i := 0; i < len(games); i += batch_size {
		if i+batch_size > len(games) {
			batch_size = len(games) - i
		}
		batch := games[i : i+batch_size]
		_, err = tx.NamedExec(query, batch)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
	}

	if err := commitTx(tx, 5*time.Second); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func SelectGamesBySeason(season string) ([]DatabaseGame, error) {
	if utils.IsInvalidSeason(season) {
		return nil, fmt.Errorf("invalid season provided: %s", season)
	}
	tx, err := db.Beginx()
	defer tx.Rollback()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	query := `
		SELECT * FROM games WHERE season = ? ORDER BY game_date DESC;
	`

	games := []DatabaseGame{}
	err = tx.Select(&games, query, season)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, 5*time.Second); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	return games, nil
}

func SelectGamesById(ids []string) ([]DatabaseGame, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`SELECT * FROM games WHERE id = ?`)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer stmt.Close()

	games := make([]DatabaseGame, 0, len(ids))
	for _, id := range ids {
		var game DatabaseGame
		if err := stmt.Get(&game, id); err != nil {
			fmt.Println(id)
			return nil, utils.ErrorWithTrace(err)
		}
		games = append(games, game)
	}
	if err := commitTx(tx, 5*time.Second); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	return games, nil
}

type Job struct {
	Id           int       `db:"id"`
	Players      string    `db:"players"`
	Games        string    `db:"games"`
	Season       string    `db:"season"`
	Slug         string    `db:"slug"`
	State        string    `db:"job_state"`
	Hash         string    `db:"job_hash"`
	ErrorDetails *string   `db:"error_details"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewJob(playerIds, gameIds []string, season string) *Job {
	slices.Sort(playerIds)
	slices.Sort(gameIds)
	hashString := strings.Join(playerIds, ",") + "|" + strings.Join(gameIds, ",") + "|" + season
	jobHash := fmt.Sprintf("%x", sha1.Sum([]byte(hashString)))
	gameIdsCSV := strings.Join(gameIds, ",")
	playerIdsCSV := strings.Join(playerIds, ",")

	return &Job{
		State:   "PENDING",
		Games:   gameIdsCSV,
		Players: playerIdsCSV,
		Season:  season,
		Hash:    jobHash,
	}
}

func (j *Job) GamesIDs() []string {
	return strings.Split(j.Games, ",")
}

func (j *Job) PlayerIDs() []string {
	return strings.Split(j.Players, ",")
}

func (j *Job) OhNo(e error) error {
	log.Println(e.Error())
	j.State = "ERROR"
	errorDetails := e.Error()
	j.ErrorDetails = &errorDetails
	return UpdateJob(j)
}

func InsertJob(job *Job) (*Job, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	var existingJob Job
	err = tx.Get(&existingJob, "SELECT * FROM jobs WHERE job_hash = ?", job.Hash)
	if err == nil {
		_ = tx.Commit()
		return &existingJob, nil
	} else if err != sql.ErrNoRows {
		return nil, utils.ErrorWithTrace(err)
	}

	maxAttempts := 10
	var slug string
	var count int
	slugStmt, err := tx.Prepare("SELECT COUNT(*) FROM jobs WHERE slug = ?")
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	for range maxAttempts {
		slug = utils.CreateSlug()
		err := slugStmt.QueryRow(slug).Scan(&count)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		if count == 0 {
			break
		}
	}
	if count != 0 {
		return nil, utils.ErrorWithTrace(fmt.Errorf("failed to create unique slug after %d attempts", maxAttempts))
	}

	job.Slug = slug

	query := `
		INSERT INTO jobs (
			players, games, season, slug, job_state, job_hash
		) VALUES (
			:players, :games, :season, :slug, :job_state, :job_hash
		)
	`
	_, err = tx.NamedExec(query, job)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, 5*time.Second); err != nil {
		return nil, err
	}
	jobRes := Job{}
	if err := db.Get(&jobRes, "SELECT * FROM jobs WHERE job_hash = ?", job.Hash); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &jobRes, nil
}

func SelectJobBySlug(slug string) (*Job, error) {
	var job Job
	if err := db.Get(&job, "SELECT * from jobs where slug = ?", slug); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &job, nil
}

func SelectJobForUpdate() (*Job, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	errChan := make(chan error, 1)
	var job Job

	go func() {
		for {
			err := tx.Get(&job, "SELECT * FROM jobs WHERE job_state = 'PENDING' ORDER BY created_at LIMIT 1")
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					errChan <- utils.ErrorWithTrace(fmt.Errorf("QUEUE EMPTY"))
					return
				}
				if strings.Contains(err.Error(), "lock") {
					time.Sleep(20 * time.Millisecond)
					if ctx.Err() != nil {
						return
					}
					continue

				} else {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
	case <-ctx.Done():
		return nil, utils.ErrorWithTrace(fmt.Errorf("timed out while attempting to select a job after %d seconds", int(timeout.Seconds())))
	}

	job.State = "PROCESSING"
	go func() {
		for {
			_, err := tx.Exec("UPDATE jobs SET job_state = 'PROCESSING' WHERE id = ?", job.Id)
			if err != nil {
				if strings.Contains(err.Error(), "lock") {
					time.Sleep(20 * time.Millisecond)
					if ctx.Err() != nil {
						return
					}
					continue
				} else {
					errChan <- err
				}
			}
			errChan <- nil
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
	case <-ctx.Done():
		return nil, utils.ErrorWithTrace(fmt.Errorf("timed out while attempting to select a job after %d seconds", int(timeout.Seconds())))
	}

	if err := commitTx(tx, 5*time.Second); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &job, nil
}

func UpdateJob(job *Job) error {
	tx, err := db.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	if _, err := tx.NamedExec("UPDATE jobs SET (job_state, error_details) = (:job_state, :error_details) WHERE id = :id", job); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, 5*time.Second); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

type Video struct {
	Id          int       `db:"id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	YoutubeUrl  string    `db:"youtube_url"`
	JobId       int       `db:"job_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func NewVideo(title, description, url string, jobId int) *Video {
	return &Video{
		Title:       title,
		Description: description,
		YoutubeUrl:  url,
		JobId:       jobId,
	}
}

func InsertVideo(video *Video) error {
	tx, err := db.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}

	_, err = tx.NamedExec(`
		INSERT INTO videos (
			title, description, youtube_url, job_id
		) VALUES (
			:title, :description, :youtube_url, :job_id
		)
	`, video)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, 5*time.Second); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func SelectVideoByJobId(id int) (*Video, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	res := Video{}
	if err := tx.Get(&res, "SELECT * FROM videos WHERE job_id = ?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.ErrorWithTrace(fmt.Errorf("unable to find a video associated with this job (◞‸ ◟ ；)"))
		} else {
			return nil, utils.ErrorWithTrace(err)
		}
	}
	return &res, nil
}

func commitTx(tx *sqlx.Tx, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errChan := make(chan error, 1)
	defer close(errChan)

	go func() {
		for {
			err := tx.Commit()
			if err == nil || errors.Is(err, sql.ErrTxDone) {
				errChan <- nil
				return
			} else if !strings.Contains(err.Error(), "lock") {
				errChan <- err
				return
			}
			time.Sleep(200 * time.Millisecond)
			if ctx.Err() != nil {
				return
			}
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return utils.ErrorWithTrace(err)
		} else {
			return nil
		}
	case <-ctx.Done():
		return utils.ErrorWithTrace(fmt.Errorf("timed out while attempting to commit sql transaction after %d seconds", int(timeout.Seconds())))
	}

}
