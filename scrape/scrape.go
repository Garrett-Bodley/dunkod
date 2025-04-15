package scrape

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"dunkod/config"
	"dunkod/db"
	"dunkod/nba"
	"dunkod/utils"

	"golang.org/x/time/rate"
)

func ScrapingDaemon(duration time.Duration) {
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
	log.Println("Scraping Team Info")
	if err := scrapeTeamInfoCommon(); err != nil {
		return utils.ErrorWithTrace(err)
	}
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

func BigScrape() error {
	log.Println("Scraping Team Info")
	if err := scrapeTeamInfoCommon(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Scraping All Games")
	if err := scrapeAllGames(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Scraping All Players")
	if err := scrapeAllPlayers(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Printf("Scraping all Box Scores")
	if err := ScrapeAllBoxScores(); err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Println("Finished Scraping")
	return nil
}

func ScrapeAllBoxScores() error {
	for _, s := range config.ValidSeasons {
		log.Println(s)
		if err := scrapeBoxScoresBySeason(s); err != nil {
			log.Println(err)
		}
		time.Sleep(10 * time.Second)
	}
	return nil
}

func scrapeBoxScoresBySeason(season string) error {
	if utils.IsInvalidSeason(season) {
		return utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}
	games, err := db.SelectGamesBySeason(season)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGames(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeBoxScores(games []db.DatabaseGame) ([]db.BoxScorePlayerStat, []db.BoxScoreScrapingError) {
	log.Printf("querying %d box scores...", len(games))
	if len(games) == 0 {
		return nil, nil
	}
	timeout := 2 * time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	playerStats := make([]db.BoxScorePlayerStat, 0, len(games)*30) // max 15 active players per team
	mu := sync.Mutex{}
	scrapingErrs := []db.BoxScoreScrapingError{}
	limiter := rate.NewLimiter(rate.Limit(5), 3) // let's try not to blow up the nba API if we can help it
	wg := sync.WaitGroup{}

	for _, g := range games {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Wait(ctx); err != nil {
				// this is kinda gross and annoying
				timeoutErr := fmt.Errorf("timed out after %s while waiting to query %s", timeout)
				scrapeErr := db.NewBoxScoreScrapingError(g.ID, utils.ErrorWithTrace(errors.Join(err, timeoutErr)))
				mu.Lock()
				defer mu.Unlock()
				scrapingErrs = append(scrapingErrs, *scrapeErr)
				return
			}

			gameStats, gameErrs := scrapeBoxScore(g.ID, g.Season)
			mu.Lock()
			defer mu.Unlock()
			playerStats = append(playerStats, gameStats...)
			if len(scrapingErrs) > 0 {
				scrapingErrs = append(scrapingErrs, gameErrs...)
			}
			log.Printf("Processed %d Entries, %d Errors", len(playerStats), len(scrapingErrs))
		}()
	}

	wg.Wait()
	return playerStats, scrapingErrs
}

func scrapeBoxScore(gid, season string) ([]db.BoxScorePlayerStat, []db.BoxScoreScrapingError) {
	boxScore, err := nba.BoxScoreTraditionalV3(gid)
	if err != nil {
		return nil, []db.BoxScoreScrapingError{*db.NewBoxScoreScrapingError(gid, err)}
	}

	stats := make([]db.BoxScorePlayerStat, 0, 15)
	errs := make([]db.BoxScoreScrapingError, 0, 15)

	homeStats, homeErrs := scrapeBoxScoreTeam(gid, season, boxScore.HomeTeam)
	stats = append(stats, homeStats...)
	errs = append(errs, homeErrs...)

	awayStats, awayErrs := scrapeBoxScoreTeam(gid, season, boxScore.AwayTeam)
	stats = append(stats, awayStats...)
	errs = append(errs, awayErrs...)

	return stats, errs
}

func scrapeBoxScoreTeam(gid, season string, team nba.BoxScoreTraditionalV3TeamStats) ([]db.BoxScorePlayerStat, []db.BoxScoreScrapingError) {
	if team.TeamId == nil {
		err := utils.ErrorWithTrace(fmt.Errorf("nil TeamID"))
		return nil, []db.BoxScoreScrapingError{*db.NewBoxScoreScrapingError(gid, err)}
	}

	stats := []db.BoxScorePlayerStat{}
	errs := []db.BoxScoreScrapingError{}

	for _, p := range team.Players {
		if p.PersonId == nil {
			err := utils.ErrorWithTrace(fmt.Errorf("nil PersonID"))
			errs = append(errs, *db.NewBoxScoreScrapingError(gid, err))
			continue
		}
		stat := db.NewBoxScorePlayerStat(
			int(*p.PersonId),
			int(*team.TeamId),
			gid,
			season,
			p.DidNotPlay(),
			p.Statistics.Minutes,
			p.Statistics.FieldGoalsMade,
			p.Statistics.FieldGoalsAttempted,
			p.Statistics.FieldGoalsPercentage,
			p.Statistics.ThreePointersMade,
			p.Statistics.ThreePointersAttempted,
			p.Statistics.ThreePointersPercentage,
			p.Statistics.FreeThrowsMade,
			p.Statistics.FreeThrowsAttempted,
			p.Statistics.FreeThrowsPercentage,
			p.Statistics.ReboundsOffensive,
			p.Statistics.ReboundsDefensive,
			p.Statistics.ReboundsTotal,
			p.Statistics.Assists,
			p.Statistics.Steals,
			p.Statistics.Blocks,
			p.Statistics.Turnovers,
			p.Statistics.FoulsPersonal,
			p.Statistics.Points,
			p.Statistics.PlusMinusPoints,
		)
		stats = append(stats, *stat)
	}
	return stats, errs
}

type BoxScoreScrapingRes struct {
	PlayerStats *[]*db.BoxScorePlayerStat
	Errors      *[]*db.BoxScoreScrapingError
}

func NewBoxScoreScrapingRes(stats *[]*db.BoxScorePlayerStat, errs *[]*db.BoxScoreScrapingError) *BoxScoreScrapingRes {
	return &BoxScoreScrapingRes{
		PlayerStats: stats,
		Errors:      errs,
	}
}

func rescrapeLastNGameBoxScores(n int) error {
	games, err := db.SelectGamesPastNDays(n)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGames(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeGames(games []db.DatabaseGame) error {
	playerStats, scrapingErrs := scrapeBoxScores(games)
	insertPlayerErr := db.InsertBoxScorePlayerStats(playerStats)
	if insertPlayerErr != nil {
		insertPlayerErr = utils.ErrorWithTrace(insertPlayerErr)
	}
	errorFromInsertingScrapingErrs := db.InsertBoxScoreScrapingErrors(scrapingErrs)
	if errorFromInsertingScrapingErrs != nil {
		errorFromInsertingScrapingErrs = utils.ErrorWithTrace(errorFromInsertingScrapingErrs)
	}
	return errors.Join(insertPlayerErr, errorFromInsertingScrapingErrs)
}

func rescrapeBoxScoreErrors() error {
	games, err := db.SelectAllGamesWithPendingScrapingErrors()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeGames(games); err != nil {
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
			dbPlayers = append(dbPlayers, *db.NewPlayer(int(*p.PersonID), utils.RemoveDiacritics(*p.DisplayFirstLast)))
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

func scrapeTeamInfoCommon() error {
	teams := make([]db.Team, 0, len(config.TeamIDs))
	for _, id := range config.TeamIDs {
		info, err := nba.TeamInfoCommon(id)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

		if info.ID == nil ||
			info.Name == nil ||
			info.City == nil ||
			info.Abbreviation == nil ||
			info.Conference == nil ||
			info.Division == nil ||
			info.Code == nil ||
			info.Slug == nil {
			log.Printf("missing required info for id: %d", id)
			log.Printf("\t%v", info)
			continue
		}

		team := db.NewTeam(
			int(*info.ID),
			*info.Name,
			*info.City,
			*info.Abbreviation,
			*info.Conference,
			*info.Division,
			*info.Code,
			*info.Slug,
		)
		teams = append(teams, *team)
	}

	if err := db.InsertTeams(teams); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}
