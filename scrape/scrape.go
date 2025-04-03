package scrape

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dunkod/config"
	"dunkod/db"
	"dunkod/nba"
	"dunkod/utils"

	"golang.org/x/time/rate"
)

func ScrapingDaemon() {
	if err := Scrape(); err != nil {
		log.Println(err)
	}
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		if err := Scrape(); err != nil {
			log.Println(err)
		}
	}
}

func Scrape() error {
	log.Println("Scraping All Games")
	if err := scrapeAllGames(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Scraping All Players")
	if err := scrapeAllPlayers(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	n := 3
	log.Printf("Re-Scraping the last %d Days of Games\n", n)
	if err := rescrapeLastNGameBoxScores(n); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Re-Scraping prior Box Score Scraping Errors")
	if err := rescrapeBoxScoreErrors(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Finished Scraping")
	return nil
}

func ScrapeAllGamesPlayers() error {
	for _, s := range config.ValidSeasons {
		log.Println(s)
		if err := scrapeGamesPlayersBySeason(s); err != nil {
			log.Println(err)
		}
		time.Sleep(10 * time.Second)
	}
	return nil
}

func scrapeGamesPlayersBySeason(season string) error {
	if utils.IsInvalidSeason(season) {
		return utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}
	games, err := db.SelectGamesBySeason(season)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGamesPlayers(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeGamesPlayers(games []db.DatabaseGame) error {
	entries := make([]db.PlayersGamesTeamsSeasonsEntry, 0, 4096)
	entryChan := make(chan *db.PlayersGamesTeamsSeasonsEntry, 100)
	scrapingErrs := []db.BoxScoreScrapingError{}
	errChan := make(chan *db.BoxScoreScrapingError, 100)
	gameWg := sync.WaitGroup{}
	resWg := sync.WaitGroup{}
	limiter := rate.NewLimiter(rate.Limit(5), 3)
	entryCount := atomic.Int64{}
	errCount := atomic.Int64{}
	log.Printf("querying %d box scores...", len(games))

	go func() {
		for err := range errChan {
			resWg.Done()
			entryInt := entryCount.Load()
			errInt := errCount.Load()
			fmt.Printf("Processed %d Entries, %d Errors     ", entryInt, errInt+1)
			errCount.Add(1)
			scrapingErrs = append(scrapingErrs, *err)
		}
	}()
	go func() {
		for entry := range entryChan {
			resWg.Done()
			entryInt := entryCount.Load()
			errInt := errCount.Load()
			fmt.Printf("Processed %d Entries, %d Errors     ", entryInt+1, errInt)
			entryCount.Add(1)
			entries = append(entries, *entry)
		}
	}()

	for _, g := range games {
		gameWg.Add(1)
		go func() {
			defer gameWg.Done()
			if err := limiter.Wait(context.Background()); err != nil {
				scrapingErr := db.NewBoxScoreScrapingError(g.ID, utils.ErrorWithTrace(err).Error())
				errChan <- scrapingErr
				return
			}
			boxScore, err := nba.BoxScoreTraditionalV2(g.ID)
			if err != nil {
				resWg.Add(1)
				errDetails := utils.ErrorWithTrace(fmt.Errorf("%s - %s\n\t%s", g.Matchup, g.GameDate, err.Error()))
				scrapingErr := db.NewBoxScoreScrapingError(g.ID, errDetails.Error())
				errChan <- scrapingErr
				return
			}
			for _, p := range boxScore.PlayerStats {
				didNotPlay, err := p.DidNotPlay()
				if err != nil {
					log.Println(err)
					continue
				}
				if didNotPlay {
					continue
				}
				if p.PlayerId == nil || p.TeamId == nil {
					resWg.Add(1)
					errDetails := utils.ErrorWithTrace(fmt.Errorf("missing PlayerID and/or TeamId for %s", *p.PlayerName))
					scrapingErr := db.NewBoxScoreScrapingError(g.ID, errDetails.Error())
					errChan <- scrapingErr
					continue
				}
				entry := db.NewPlayersGamesTeamsSeasonsEntry(int(*p.PlayerId), int(*p.TeamId), g.ID, g.Season)
				resWg.Add(1)
				entryChan <- entry
			}
		}()
	}
	gameWg.Wait()
	close(entryChan)
	close(errChan)
	resWg.Wait()
	// // We are done logging progress so let's print a new line
	// fmt.Printf("\n")
	entryErr := db.InsertPlayersGamesTeamsSeasonsEntries(entries)
	if entryErr != nil {
		entryErr = utils.ErrorWithTrace(entryErr)
	}
	errorsErr := db.InsertBoxScoreScrapingErrors(scrapingErrs)
	if errorsErr != nil {
		errorsErr = utils.ErrorWithTrace(errorsErr)
	}
	if entryErr != nil || errorsErr != nil {
		return errors.Join(entryErr, errorsErr)
	}
	return nil
}

func scrapePlayerGames(playerID int) ([]db.PlayersGamesTeamsSeasonsEntry, error) {
	entries := []db.PlayersGamesTeamsSeasonsEntry{}
	for _, s := range config.ValidSeasons {
		games, err := nba.LeagueGameFinderByPlayerIDAndSeason(playerID, s)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		for _, g := range games {
			if g.IsRegularSeason() || g.IsPlayIn() || g.IsPlayoffs() {
				log.Println(g.SeasonID)
				entry := db.NewPlayersGamesTeamsSeasonsEntry(playerID, int(*g.TeamID), *g.GameID, s)
				entries = append(entries, *entry)
			}
		}
	}
	return entries, nil
}

func rescrapeLastNGameBoxScores(n int) error {
	games, err := db.SelectGamesPastNDays(3)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGamesPlayers(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func rescrapeBoxScoreErrors() error {
	games, err := db.SelectAllGamesWithScrapingErrors()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGamesPlayers(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeAllPlayers() error {
	players, err := nba.CommonAllPlayerAllSeasons()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	dbPlayers := make([]db.Player, 0, len(players))
	for _, p := range players {
		if p.PersonID != nil && p.DisplayFirstLast != nil {
			dbPlayers = append(dbPlayers, *db.NewPlayer(int(*p.PersonID), *p.DisplayFirstLast))
			continue
		}
		if p.PersonID == nil && p.DisplayFirstLast != nil {
			log.Printf("%s missing PersonID\n", *p.DisplayFirstLast)
		} else if p.PersonID != nil && p.DisplayFirstLast == nil {
			log.Printf("%d missing DisplayFirstLast\n", int(*p.PersonID))
		} else {
			log.Printf("missing both PersonID and DisplayFirstLast:\n\t%v", p)
		}
	}
	if err := db.InsertPlayers(dbPlayers); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeAllGames() error {
	dbGames := []db.DatabaseGame{}
	for _, t := range config.SeasonTypes {
		log.Println(t)
		for _, s := range config.ValidSeasons {
			games, err := nba.LeagueGameLog(s, t)
			if err != nil {
				log.Println(s, t)
				return utils.ErrorWithTrace(err)
			}
			deduped, err := dedupGames(t, s, games)
			if err != nil {
				return utils.ErrorWithTrace(err)
			}
			dbGames = append(dbGames, deduped...)
		}
	}
	if err := db.InsertGames(dbGames); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("done")
	return nil
}

func dedupGames(seasonType, season string, games []nba.LeagueGameLogGame) ([]db.DatabaseGame, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}
	dbGames := []db.DatabaseGame{}
	hash := map[string][]nba.LeagueGameLogGame{}
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
			ID:          k,
			Season:      season,
			GameDate:    *a.GameDate,
			Matchup:     *winner.Matchup,
			SeasonType:  seasonType,
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
	return dbGames, nil
}
