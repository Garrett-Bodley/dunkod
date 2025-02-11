package main

import (
	"dunkod/config"
	"dunkod/db"
	"dunkod/nba"

	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

func init() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}
	if err := db.SetupDatabase(); err != nil {
		panic(err)
	}
	if err := db.RunMigrations(); err != nil {
		panic(err)
	}
	if err := db.ValidateMigrations(); err != nil {
		panic(err)
	}
	fmt.Println("The New York Knickerbockers are named after pants")
}

var playerCacheMu = sync.Mutex{}
var playerCache = map[string][]nba.CommonAllPlayer{}

func main() {
	scrapingDaemon()
}

func fetchPlayers(season string) ([]nba.CommonAllPlayer, error) {
	playerCacheMu.Lock()
	defer playerCacheMu.Unlock()
	if cached, ok := playerCache[season]; ok {
		return cached, nil
	}

	players, err := nba.CommonAllPlayersBySeason(season)
	if err != nil {
		return nil, err
	}
	playerCache[season] = players
	return players, nil
}

func scrapingDaemon() {
	log.Println("scraping all games")
	scrapeAllGames()
	last := time.Now()
	for {
		if now := time.Now(); now.Sub(last) >= 30*time.Minute {
			last = now
			log.Println("scraping all games")
			scrapeAllGames()
		}
	}
}

func scrapeAllGames() error {
	dbGames := []db.DatabaseGame{}

	for _, t := range config.SeasonTypes {
		fmt.Println(t)
		for _, s := range config.ValidSeasons {
			hash := map[string][]nba.LeagueGameLogGame{}
			games, err := nba.LeagueGameLog(s, t)
			if err != nil {
				return err
			}

			for _, g := range games {
				// Skip if no available video
				if *g.VideoAvailable == float64(0) {
					continue
				}
				if _, exists := hash[*g.GameID]; !exists {
					hash[*g.GameID] = make([]nba.LeagueGameLogGame, 0, 2)
				}
				hash[*g.GameID] = append(hash[*g.GameID], g)
			}

			for k, v := range hash {
				a, b := v[0], v[1]
				var winner nba.LeagueGameLogGame
				var loser nba.LeagueGameLogGame
				var homeTeam nba.LeagueGameLogGame
				var awayTeam nba.LeagueGameLogGame
				if a.PTS == nil || b.PTS == nil {
					log.Printf("found matchup containing nil points field: \n\tMatchup: %s\n\tGameID: %s", *a.Matchup, k)
					continue
				}
				if *a.PTS > *b.PTS {
					winner, loser = a, b
				} else {
					winner, loser = b, a
				}

				if strings.Contains(*a.Matchup, "@") {
					homeTeam, awayTeam = a, b
				} else {
					homeTeam, awayTeam = b, a
				}

				dbGames = append(dbGames, db.DatabaseGame{
					Id:          k,
					Season:      s,
					GameDate:    *a.GameDate,
					Matchup:     *winner.Matchup,
					SeasonType:  t,
					WinnerName:  *winner.TeamName,
					WinnerID:    int(*winner.TeamID),
					WinnerScore: int(*winner.PTS),
					LoserName:   *loser.TeamName,
					LoserID:     int(*loser.TeamID),
					LoserScore:  int(*loser.PTS),
					HomeTeamId:  int(*homeTeam.TeamID),
					AwayTeamId:  int(*awayTeam.TeamID),
				})
			}
		}
		db.InsertGames(dbGames)
	}
	fmt.Println("done")
	return nil
}
