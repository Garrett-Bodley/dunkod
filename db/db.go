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

var dbRW *sqlx.DB
var dbRO *sqlx.DB

func Close() error {
	timeout := 15 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return errors.Join(walCheckpoint(&ctx), dbRW.Close(), dbRO.Close())
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
	dbRW, err = sqlx.Connect("sqlite3", "file:"+config.DatabaseFile+"?_journal_mode=WAL&_fk=true&mode=rw&_txlock=immediate")
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	dbRW.SetMaxOpenConns(1)
	if err := validateDbRW(dbRW); err != nil {
		return utils.ErrorWithTrace(err)
	}

	dbRO, err = sqlx.Connect("sqlite3", "file:"+config.DatabaseFile+"?_journal_mode=WAL&_fk=true&mode=ro")
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := validateDbRO(dbRO); err != nil {
		return utils.ErrorWithTrace(err)
	}

	return nil
}

func validateDbRW(db *sqlx.DB) error {
	var readOnly int
	if err := db.QueryRow("PRAGMA query_only;").Scan(&readOnly); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if readOnly != 0 {
		return utils.ErrorWithTrace(fmt.Errorf("dbRW is somehow in read only mode (◞‸ ◟ ；)"))
	}
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if journalMode != "wal" {
		return utils.ErrorWithTrace(fmt.Errorf("dbRW journal mode is somehow \"%s\" (◞‸ ◟ ；)", journalMode))
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if foreignKeys != 1 {
		return utils.ErrorWithTrace(fmt.Errorf("dbRW somehow has foreign key constraints disabled (◞‸ ◟ ；)"))
	}
	return nil
}

func validateDbRO(db *sqlx.DB) error {
	var readOnly int
	if err := dbRO.QueryRow("PRAGMA query_only;").Scan(&readOnly); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if readOnly != 1 {
		utils.ErrorWithTrace(fmt.Errorf("dbRO is somehow not in read only mode (◞‸ ◟ ；)"))
	}
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if journalMode != "wal" {
		return utils.ErrorWithTrace(fmt.Errorf("dbRW journal mode is somehow \"%s\" (◞‸ ◟ ；)", journalMode))
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if foreignKeys != 1 {
		return utils.ErrorWithTrace(fmt.Errorf("dbRW somehow has foreign key constraints disabled (◞‸ ◟ ；)"))
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
	if err := dbRO.QueryRow("SELECT COUNT(*) FROM teams;").Scan(&count); err != nil {
		return utils.ErrorWithTrace(err)
	}

	if count != 31 {
		return utils.ErrorWithTrace(fmt.Errorf("expected 31 teams, found %d", count))
	}

	var name string
	if err := dbRO.QueryRow("SELECT team_name FROM teams WHERE id = 1610612752;").Scan(&name); err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("failed to find Knicks: %v", err))
	}
	if name != "New York Knicks" {
		return utils.ErrorWithTrace(fmt.Errorf("expected team.id 1610612752 to have name 'New York Knicks', got '%s'", name))
	}
	err := dbRO.QueryRow("SELECT team_name FROM teams WHERE id = 0;").Scan(&name)
	if err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("failed to find NULL_TEAM: %v", err))
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

func InsertGames(games []DatabaseGame, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	query := `
		INSERT OR IGNORE INTO games (
			id, season, game_date, matchup, season_type, winner_name,
			winner_id, winner_score, loser_name, loser_id, loser_score,
			home_team_id, away_team_id
		) VALUES (
			:id, :season, :game_date, :matchup, :season_type, :winner_name,
			:winner_id, :winner_score, :loser_name, :loser_id, :loser_score,
			:home_team_id, :away_team_id
		);
	`
	batchSize := 500
	if err := batchInsert(tx, &ctx, batchSize, query, games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func SelectAllGames(timeout ...time.Duration) ([]DatabaseGame, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	defer tx.Rollback()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	games := []DatabaseGame{}
	if err := selekt(tx, &ctx, &games, "SELECT * FROM games ORDER BY game_date DESC;"); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return games, nil
}

func SelectGamesBySeason(season string, timeout ...time.Duration) ([]DatabaseGame, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if utils.IsInvalidSeason(season) {
		return nil, fmt.Errorf("invalid season provided: %s", season)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	defer tx.Rollback()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	games := []DatabaseGame{}
	if err := selekt(tx, &ctx, &games, "SELECT * FROM games WHERE season = ? ORDER BY game_date DESC;", season); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	return games, nil
}

func SelectGamesPastNDays(n int, timeout ...time.Duration) ([]DatabaseGame, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	defer tx.Rollback()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	query := `
		SELECT *
		FROM   games
		WHERE  game_date > Date('now', Concat('-', ?, ' days'))
		ORDER  BY game_date ASC;
	`
	games := []DatabaseGame{}
	if err := selekt(tx, &ctx, &games, query, n); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return games, nil
}

func SelectGamesById(ids []string, timeout ...time.Duration) ([]DatabaseGame, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex("SELECT * FROM games WHERE id = ?;")
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer stmt.Close()

	games := make([]DatabaseGame, 0, len(ids))
	for _, id := range ids {
		var game DatabaseGame
		if err := stmtDotGet(stmt, &ctx, &game, id); err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		games = append(games, game)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return games, nil
}

func SelectAllGamesWithScrapingErrors(timeout ...time.Duration) ([]DatabaseGame, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	query := `
		SELECT g.*
		FROM   games g
		INNER JOIN box_score_scraping_errors screrrors
			ON g.id = screrrors.game_id;
	`
	games := []DatabaseGame{}
	if err := selekt(tx, &ctx, &games, query); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return games, nil
}

type Player struct {
	Id        int       `db:"id"`
	Name      string    `db:"player_name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewPlayer(id int, name string) *Player {
	return &Player{
		Id:   id,
		Name: name,
	}
}

func InsertPlayers(players []Player, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	defer tx.Rollback()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	query := `
		INSERT OR IGNORE INTO players (
			id, player_name
		) VALUES (
			:id, :player_name
		);
	`
	batchSize := 500
	if err := batchInsert(tx, &ctx, batchSize, query, players); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func SelectAllPlayers(timeout ...time.Duration) ([]Player, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	defer tx.Rollback()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	players := []Player{}
	if err := selekt(tx, &ctx, &players, "SELECT * FROM players;"); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return players, nil
}

// func SelectAllPlayersBySeason(season string) ([]Player, error) {
// 	timeout := 5 * time.Second
// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()

// 	tx, err := dbRO.Beginx()
// 	defer tx.Rollback()
// 	if err != nil {
// 		return nil, utils.ErrorWithTrace(err)
// 	}

// 	query = `
// 		SELECT p.*
// 		FROM players
// 		INNER JOIN players_games_teams_seasons pgts ON p.id = pgts.player_id
// 		WHERE pgts.
// 	`
// 	players := []Player{}
// 	if err := selekt(tx, &ctx, )
// }

type PlayersGamesTeamsSeasonsEntry struct {
	Id        int       `db:"id"`
	PlayerID  int       `db:"player_id"`
	TeamID    int       `db:"team_id"`
	GameID    string    `db:"game_id"`
	Season    string    `db:"season"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewPlayersGamesTeamsSeasonsEntry(playerID, teamID int, gameID, season string) *PlayersGamesTeamsSeasonsEntry {
	return &PlayersGamesTeamsSeasonsEntry{
		PlayerID: playerID,
		TeamID:   teamID,
		GameID:   gameID,
		Season:   season,
	}
}

func InsertPlayersGamesTeamsSeasonsEntries(entries []PlayersGamesTeamsSeasonsEntry, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()
	query := `
		INSERT OR IGNORE INTO players_games_teams_seasons (
			player_id, team_id, game_id, season
		) VALUES (
			:player_id, :team_id, :game_id, :season
		);
	`
	batchSize := 1
	if err := batchInsert(tx, &ctx, batchSize, query, entries); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
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

func InsertJob(job *Job, timeout ...time.Duration) (*Job, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	var existingJob Job
	err = get(tx, &ctx, &existingJob, "SELECT * FROM jobs WHERE job_hash = ?;", job.Hash)
	if err == nil {
		return &existingJob, nil
	} else if !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		return nil, utils.ErrorWithTrace(err)
	}

	maxAttempts := 50
	var slug string
	var count int
	slugStmt, err := tx.Prepare("SELECT COUNT(*) FROM jobs WHERE slug = ?;")
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	for range maxAttempts {
		slug = utils.CreateSlug()
		err := stmtDotQueryRow(slugStmt, &ctx, []any{slug}, []any{&count})
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
		INSERT OR IGNORE INTO jobs (
			players, games, season, slug, job_state, job_hash
		) VALUES (
			:players, :games, :season, :slug, :job_state, :job_hash
		);
	`
	if err := namedExec(tx, &ctx, query, job); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, err
	}
	if err := walCheckpoint(&ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	tx, err = dbRO.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := get(tx, &ctx, job, "SELECT * FROM jobs WHERE job_hash = ?;", job.Hash); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return job, nil
}

func SelectJobBySlug(slug string, timeout ...time.Duration) (*Job, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	var job Job
	if err := get(tx, &ctx, &job, "SELECT * from jobs where slug = ?;", slug); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &job, nil
}

func SelectJobForUpdate(timeout ...time.Duration) (*Job, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	var job Job
	if err := get(tx, &ctx, &job, "SELECT * FROM jobs WHERE job_state = 'PENDING' ORDER BY created_at LIMIT 1;"); err != nil {
		if strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
			return nil, fmt.Errorf("QUEUE EMPTY")
		} else {
			return nil, utils.ErrorWithTrace(err)
		}
	}
	job.State = "PROCESSING"
	if err := exec(tx, &ctx, "UPDATE jobs SET job_state = 'PROCESSING' WHERE id = ?;", job.Id); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &job, nil
}

func UpdateJob(job *Job, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()
	query := "UPDATE jobs SET (job_state, error_details) = (:job_state, :error_details) WHERE id = :id;"
	if err := namedExec(tx, &ctx, query, job); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

type Video struct {
	Id          int       `db:"id"`
	Title       string    `db:"title"`
	Description string    `db:"video_description"`
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

func InsertVideo(video *Video, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}

	query := `
		INSERT OR IGNORE INTO videos (
			title, video_description, youtube_url, job_id
		) VALUES (
			:title, :video_description, :youtube_url, :job_id
		);
	`
	if err := namedExec(tx, &ctx, query, video); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func SelectVideoByJobId(id int, timeout ...time.Duration) (*Video, error) {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRO.Beginx()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	res := Video{}
	if err := get(tx, &ctx, &res, "SELECT * FROM videos WHERE job_id = ?;", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.ErrorWithTrace(fmt.Errorf("unable to find a video associated with this job (◞‸ ◟ ；)"))
		} else {
			return nil, utils.ErrorWithTrace(err)
		}
	}
	return &res, nil
}

type BoxScoreScrapingError struct {
	Id           int       `db:"id"`
	GameID       string    `db:"game_id"`
	ErrorDetails string    `db:"error_details"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewBoxScoreScrapingError(gameID, errorDetails string) *BoxScoreScrapingError {
	return &BoxScoreScrapingError{
		GameID:       gameID,
		ErrorDetails: errorDetails,
	}
}

func InsertBoxScoreScrapingErrors(errors []BoxScoreScrapingError, timeout ...time.Duration) error {
	parsedTimeout, err := parseTimeout(timeout...)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parsedTimeout)
	defer cancel()

	tx, err := dbRW.Beginx()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO box_score_scraping_errors (
			game_id, error_details
		) VALUES (
			:game_id, :error_details
		)
	`
	batchSize := 500
	if err := batchInsert(tx, &ctx, batchSize, query, errors); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := walCheckpoint(&ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func get(tx *sqlx.Tx, ctx *context.Context, dest any, query string, args ...any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				utils.ErrorWithTrace(fmt.Errorf("timed out while attempting tx.Get"))
			default:
				err := tx.Get(dest, query, args...)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func exec(tx *sqlx.Tx, ctx *context.Context, query string, args ...any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting tx.Exec"))
				return
			default:
				_, err := tx.Exec(query, args...)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func selekt(tx *sqlx.Tx, ctx *context.Context, dest any, query string, args ...any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting tx.Select"))
				return
			default:
				err := tx.Select(dest, query, args...)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func namedExec(tx *sqlx.Tx, ctx *context.Context, query string, arg any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting tx.NamedExec"))
				return
			default:
				_, err := tx.NamedExec(query, arg)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func stmtDotGet(stmt *sqlx.Stmt, ctx *context.Context, dest any, args ...any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting stmt.Get"))
				return
			default:
				err := stmt.Get(dest, args...)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func stmtDotQueryRow(stmt *sql.Stmt, ctx *context.Context, args []any, dest []any) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting stmt.QueryRow"))
				return
			default:
				err := stmt.QueryRow(args...).Scan(dest...)
				if err == nil || errors.Is(err, sql.ErrTxDone) {
					errChan <- nil
					return
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func commitTx(tx *sqlx.Tx, ctx *context.Context) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		for {
			for {
				select {
				case <-(*ctx).Done():
					errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out will attempting tx.Commit"))
					return
				default:
					err := tx.Commit()
					if err == nil || errors.Is(err, sql.ErrTxDone) {
						errChan <- nil
						return
					} else if !strings.Contains(err.Error(), "lock") {
						errChan <- utils.ErrorWithTrace(err)
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
		}
	}()
	return <-errChan
}

func walCheckpoint(ctx *context.Context) error {
	errChan := make(chan error, 1)
	defer close(errChan)
	go func() {
		var (
			mode, pagesWritten, pagesMoved int
		)
		for {
			select {
			case <-(*ctx).Done():
				errChan <- utils.ErrorWithTrace(fmt.Errorf("timed out while attempting wal_checkpoint(TRUNCATE)"))
				return
			default:
				err := dbRW.QueryRow("PRAGMA wal_checkpoint(TRUNCATE);").Scan(&mode, &pagesWritten, &pagesMoved)
				if err == nil {
					if mode == 0 {
						errChan <- nil
						return
					} else if pagesWritten == -1 || pagesMoved == -1 {
						errChan <- utils.ErrorWithTrace(fmt.Errorf("database not in WAL mode"))
					}
				} else if !strings.Contains(err.Error(), "lock") {
					errChan <- utils.ErrorWithTrace(err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return <-errChan
}

func batchInsert[T any](tx *sqlx.Tx, ctx *context.Context, batchSize int, query string, arg []T) error {
	for i := 0; i < len(arg); i += batchSize {
		if i+batchSize > len(arg) {
			batchSize = len(arg) - i
		}
		batch := arg[i : i+batchSize]
		if err := namedExec(tx, ctx, query, batch); err != nil {
			return utils.ErrorWithTrace(err)
		}
	}
	return nil
}

const DEFAULT_TIMEOUT time.Duration = 5 * time.Second

func parseTimeout(timeout ...time.Duration) (time.Duration, error) {
	if len(timeout) == 0 {
		return DEFAULT_TIMEOUT, nil
	}
	if len(timeout) > 1 {
		return 0, utils.ErrorWithTrace(fmt.Errorf("expected single timeout value, received %d", len(timeout)))
	}
	return timeout[0], nil
}
