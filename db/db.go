package db

import (
	"dunkod/config"
	"dunkod/utils"

	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

type DatabaseGame struct {
	Id          string `db:"id"`
	Season      string `db:"season"`
	GameDate    string `db:"game_date"`
	Matchup     string `db:"matchup"`
	SeasonType  string `db:"season_type"`
	WinnerName  string `db:"winner_name"`
	WinnerID    int    `db:"winner_id"`
	WinnerScore int    `db:"winner_score"`
	LoserName   string `db:"loser_name"`
	LoserID     int    `db:"loser_id"`
	LoserScore  int    `db:"loser_score"`
	HomeTeamId  int    `db:"home_team_id"`
	AwayTeamId  int    `db:"away_team_id"`
}

func SetupDatabase() error {
	_, err := os.Stat(config.DatabaseFile)
	if os.IsNotExist(err) {
		log.Println("Database file not found. Creating a new database.")
		file, err := os.Create(config.DatabaseFile)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		file.Close()
	} else if err != nil {
		return err
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
	return nil
}

func ValidateMigrations() error {
	db, err := sqlx.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	defer db.Close()

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
	err = db.QueryRow("SELECT name FROM teams WHERE id = 0").Scan(&name)
	if err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("faild to find NULL_TEAM: %v", err))
	}
	if name != "NULL_TEAM" {
		return utils.ErrorWithTrace(fmt.Errorf("expected team.id 0 to have name 'NULL_TEAM', got '%s'", name))
	}
	return nil
}

func InsertGames(games []DatabaseGame) error {
	db, err := sqlx.Open("sqlite3", config.DatabaseFile)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Beginx()
	if err != nil {
		return err
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
	for _, g := range games {
		_, err := tx.NamedExec(query, g)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

