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
	return errors.Join(dbRW.Close(), dbRO.Close())
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
	if err := tx.Commit(); err != nil {
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

func SelectAllGamesWithPendingScrapingErrors(timeout ...time.Duration) ([]DatabaseGame, error) {
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
		SELECT	g.*
		FROM	games g
			INNER JOIN box_score_scraping_errors screrrors
				ON g.id = screrrors.game_id
		WHERE	screrrors.error_status = 'PENDING';
	`
	games := []DatabaseGame{}
	if err := selekt(tx, &ctx, &games, query); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
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
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return players, nil
}

func SelectPlayersBySeason(season string, timeout ...time.Duration) ([]Player, error) {
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
		SELECT p.*
		FROM   players
		INNER JOIN box_score_player_stats bsps
			ON p.id = bsps.player_id
		WHERE  bsps.season = ?
	`
	players := []Player{}
	if err := selekt(tx, &ctx, &players, query, season); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return players, nil
}

func SelectPlayerNamesById(ids []string, timeout ...time.Duration) ([]string, error) {
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
		SELECT player_name FROM players WHERE id IN (?);
	`
	query, args, err := sqlx.In(query, ids)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	names := []string{}
	if err := selekt(tx, &ctx, &names, query, args...); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return names, nil
}

type BoxScorePlayerStat struct {
	Id        int       `db:"id"`
	PlayerID  int       `db:"player_id"`
	TeamID    int       `db:"team_id"`
	GameID    string    `db:"game_id"`
	Season    string    `db:"season"`
	MIN       *string   `db:"min"`
	FGM       *float64  `db:"fgm"`
	FGA       *float64  `db:"fga"`
	FG_PCT    *float64  `db:"fg_pct"`
	FG3M      *float64  `db:"fg3m"`
	FG3A      *float64  `db:"fg3a"`
	FG3_PCT   *float64  `db:"fg3_pct"`
	FTM       *float64  `db:"ftm"`
	FTA       *float64  `db:"fta"`
	FT_PCT    *float64  `db:"ft_pct"`
	OREB      *float64  `db:"oreb"`
	DREB      *float64  `db:"dreb"`
	REB       *float64  `db:"reb"`
	AST       *float64  `db:"ast"`
	STL       *float64  `db:"stl"`
	BLK       *float64  `db:"blk"`
	TOV       *float64  `db:"tov"`
	PF        *float64  `db:"pf"`
	PTS       *float64  `db:"pts"`
	PlusMinus *float64  `db:"plus_minus"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewBoxScorePlayerStat(
	playerID, teamID int,
	gameID, season string,
	min *string,
	fgm, fga, fg_pct, fg3m, fg3a, fg3_pct, ftm, fta, ft_pct, oreb, dreb, reb, ast, stl, blk, tov, pf, pts, plusMinus *float64,
) *BoxScorePlayerStat {
	return &BoxScorePlayerStat{
		PlayerID:  playerID,
		TeamID:    teamID,
		GameID:    gameID,
		Season:    season,
		MIN:       min,
		FGM:       fgm,
		FGA:       fga,
		FG_PCT:    fg_pct,
		FG3M:      fg3m,
		FG3A:      fg3a,
		FG3_PCT:   fg3_pct,
		FTM:       ftm,
		FTA:       fta,
		FT_PCT:    ft_pct,
		OREB:      oreb,
		DREB:      dreb,
		REB:       reb,
		AST:       ast,
		STL:       stl,
		BLK:       blk,
		TOV:       tov,
		PF:        pf,
		PTS:       pts,
		PlusMinus: plusMinus,
	}
}

func InsertBoxScorePlayerStats(stats []BoxScorePlayerStat, timeout ...time.Duration) error {
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
		INSERT
		or     IGNORE
		into   box_score_player_stats
		(
			player_id,
			team_id,
			game_id,
			season,
			min,
			fgm,
			fga,
			fg_pct,
			fg3m,
			fg3a,
			fg3_pct,
			ftm,
			fta,
			ft_pct,
			oreb,
			dreb,
			reb,
			ast,
			stl,
			blk,
			tov,
			pf,
			pts,
			plus_minus
		)
		VALUES
		(
			:player_id,
			:team_id,
			:game_id,
			:season,
			:min,
			:fgm,
			:fga,
			:fg_pct,
			:fg3m,
			:fg3a,
			:fg3_pct,
			:ftm,
			:fta,
			:ft_pct,
			:oreb,
			:dreb,
			:reb,
			:ast,
			:stl,
			:blk,
			:tov,
			:pf,
			:pts,
			:plus_minus
		);
	`

	batchSize := 500
	if err := batchInsert(tx, &ctx, batchSize, query, stats); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
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
	if err := commitTx(tx, &ctx); err != nil {
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
	return nil
}

func ResetStaleJobs(timeout ...time.Duration) error {
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
		UPDATE jobs
		SET job_state = CASE
				WHEN job_state = 'PROCESSING'
					AND Datetime(updated_at) < Datetime('now', '-5 minutes') THEN
					'PENDING'
				WHEN job_state = 'DOWNLOADING CLIPS'
					AND Datetime(updated_at) < Datetime('now', '-10 minutes') THEN
					'PENDING'
				WHEN job_state = 'UPLOADING'
					AND Datetime(updated_at) < Datetime('now', '-60 minutes') THEN
					'PENDING'
				ELSE
					job_state
				END
		WHERE job_state IN ( 'PROCESSING', 'DOWNLOADING CLIPS', 'UPLOADING' );
	`
	if err := exec(tx, &ctx, query); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
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
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return &res, nil
}

type BoxScoreScrapingError struct {
	Id           int       `db:"id"`
	GameID       string    `db:"game_id"`
	ErrorDetails string    `db:"error_details"`
	ErrorStatus  string    `db:"error_status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewBoxScoreScrapingError(gameID, errorDetails string) *BoxScoreScrapingError {
	return &BoxScoreScrapingError{
		GameID:       gameID,
		ErrorDetails: errorDetails,
		ErrorStatus:  "PENDING",
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
			game_id, error_details, error_status
		) VALUES (
			:game_id, :error_details, :error_status
		)
	`
	batchSize := 500
	if err := batchInsert(tx, &ctx, batchSize, query, errors); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func UpdateResolvedBoxScoreScrapingErrors(ids []string, timeout ...time.Duration) error {
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
		UPDATE box_score_scraping_errors SET error_stats = 'RESOLVED' WHERE game_id IN (?);
	`
	query, args, err := sqlx.In(query, ids)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := exec(tx, &ctx, query, args...); err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

type Team struct {
	ID           int       `db:"id"`
	TeamName     string    `db:"team_name"`
	City         string    `db:"city"`
	Abbreviation string    `db:"abbreviation"`
	Conference   string    `db:"conference"`
	Division     string    `db:"division"`
	Code         string    `db:"code"`
	Slug         string    `db:"slug"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewTeam(id int, teamName, city, abbreviation, conference, division, code, slug string) *Team {
	return &Team{
		ID:           id,
		TeamName:     teamName,
		City:         city,
		Abbreviation: abbreviation,
		Conference:   conference,
		Division:     division,
		Code:         code,
		Slug:         slug,
	}
}

func InsertTeams(teams []Team, timeout ...time.Duration) error {
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
		INSERT OR IGNORE INTO teams (
			id,
			team_name,
			city,
			abbreviation,
			conference,
			division,
			code,
			slug
		) VALUES (
			:id,
			:team_name,
			:city,
			:abbreviation,
			:conference,
			:division,
			:code,
			:slug
		);
	`
	if err := namedExec(tx, &ctx, query, teams); err != nil {
		utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		utils.ErrorWithTrace(err)
	}
	return nil
}

type PlayerSearchInfo struct {
	PlayerID          int    `db:"player_id"`
	PlayerName        string `db:"player_name"`
	TeamNames         string `db:"team_names"`
	TeamCities        string `db:"team_cities"`
	TeamAbbreviations string `db:"team_abbreviations"`
	TeamConferences   string `db:"team_conferences"`
	TeamDivisions     string `db:"team_divisions"`
	TeamCodes         string `db:"team_codes"`
	TeamSlugs         string `db:"team_slugs"`
}

func (s *PlayerSearchInfo) SearchString() string {
	return strings.ToLower(strings.Join([]string{
		fmt.Sprintf("%d", s.PlayerID),
		s.PlayerName,
		s.TeamNames,
		s.TeamCities,
		s.TeamAbbreviations,
		s.TeamConferences,
		s.TeamDivisions,
		s.TeamCodes,
		s.TeamSlugs,
	}, ","))
}

func GetPlayerPlayerSearchInfoBySeason(season string, timeout ...time.Duration) ([]PlayerSearchInfo, error) {
	if utils.IsInvalidSeason(season) {
		return nil, fmt.Errorf("invalid season provided: %s", season)
	}
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

	query := `
		SELECT	p.id					AS player_id,
			p.player_name 				AS player_name,
			Group_concat(DISTINCT t.team_name)	AS team_names,
			Group_concat(DISTINCT t.city)		AS team_cities,
			Group_concat(DISTINCT t.abbreviation)	AS team_abbreviations,
			Group_concat(DISTINCT t.conference)	AS team_conferences,
			Group_concat(DISTINCT t.division)	AS team_divisions,
			Group_concat(DISTINCT t.code)		AS team_codes,
			Group_concat(DISTINCT t.slug)		AS team_slugs
		FROM	players p
			INNER JOIN box_score_player_stats bsps
				ON p.id = bsps.player_id
			INNER JOIN teams t
				ON bsps.team_id = t.id
		WHERE	bsps.season = ?
		GROUP	BY p.id
		ORDER 	BY p.player_name ASC;
	`
	info := []PlayerSearchInfo{}
	if err := selekt(tx, &ctx, &info, query, season); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if err := commitTx(tx, &ctx); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return info, nil
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
