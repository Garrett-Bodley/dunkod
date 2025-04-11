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
	if err := scrapeBoxScores(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func scrapeBoxScores(games []db.DatabaseGame) error {
	log.Printf("querying %d box scores...", len(games))
	if len(games) == 0 {
		return nil
	}

	playerStats := make([]db.BoxScorePlayerStat, 0, 4096)
	statChan := make(chan *db.BoxScorePlayerStat, 100)
	scrapingErrs := []db.BoxScoreScrapingError{}
	errChan := make(chan *db.BoxScoreScrapingError, 100)
	gameWg := sync.WaitGroup{}
	resWg := sync.WaitGroup{}
	limiter := rate.NewLimiter(rate.Limit(5), 3)
	entryCount := atomic.Int64{}
	errCount := atomic.Int64{}
	go func() {
		for err := range errChan {
			resWg.Done()
			entryInt := entryCount.Load()
			errInt := errCount.Load()
			log.Printf("\tProcessed %d Entries, %d Errors     ", entryInt, errInt+1)
			errCount.Add(1)
			scrapingErrs = append(scrapingErrs, *err)
		}
	}()
	go func() {
		for playerStat := range statChan {
			resWg.Done()
			entryInt := entryCount.Load()
			errInt := errCount.Load()
			log.Printf("\tProcessed %d Entries, %d Errors     ", entryInt+1, errInt)
			entryCount.Add(1)
			playerStats = append(playerStats, *playerStat)
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
				playerStat := db.NewBoxScorePlayerStat(
					int(*p.PlayerId),
					int(*p.TeamId),
					*p.GameID,
					g.Season,
					p.MIN,
					p.FGM,
					p.FGA,
					p.FG_PCT,
					p.FG3M,
					p.FG3A,
					p.FG3_PCT,
					p.FTM,
					p.FTA,
					p.FT_PCT,
					p.OREB,
					p.DREB,
					p.REB,
					p.AST,
					p.STL,
					p.BLK,
					p.TO,
					p.PF,
					p.PTS,
					p.PlusMinus,
				)
				resWg.Add(1)
				statChan <- playerStat
			}
		}()
	}
	gameWg.Wait()
	close(statChan)
	close(errChan)
	resWg.Wait()
	entryErr := db.InsertBoxScorePlayerStats(playerStats)
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

func scrapeBoxScore(game db.DatabaseGame) *BoxScoreScrapingRes {
	boxScore, err := nba.BoxScoreTraditionalV2(game.ID)
	if err != nil {
		return NewBoxScoreScrapingRes(nil, &[]*db.BoxScoreScrapingError{
			db.NewBoxScoreScrapingError(game.ID, utils.ErrorWithTrace(err).Error()),
		})
	}
	playerStats := []*db.BoxScorePlayerStat{}
	errs := []*db.BoxScoreScrapingError{}

	for _, p := range boxScore.PlayerStats {
		didNotPlay, err := p.DidNotPlay()
		if err != nil {
			scrapingErr := db.NewBoxScoreScrapingError(game.ID, utils.ErrorWithTrace(err).Error())
			errs = append(errs, scrapingErr)
			continue
		}
		if didNotPlay {
			continue
		}
		if p.PlayerId == nil || p.TeamId == nil {
			errDetails := utils.ErrorWithTrace(fmt.Errorf("missing PlayerID and/or TeamId for %s", *p.PlayerName))
			scrapingErr := db.NewBoxScoreScrapingError(game.ID, errDetails.Error())
			errs = append(errs, scrapingErr)
			continue
		}
		playerStat := db.NewBoxScorePlayerStat(
			int(*p.PlayerId),
			int(*p.TeamId),
			game.ID,
			game.Season,
			p.MIN,
			p.FGM,
			p.FGA,
			p.FG_PCT,
			p.FG3M,
			p.FG3A,
			p.FG3_PCT,
			p.FTM,
			p.FTA,
			p.FT_PCT,
			p.OREB,
			p.DREB,
			p.REB,
			p.AST,
			p.STL,
			p.BLK,
			p.TO,
			p.PF,
			p.PTS,
			p.PlusMinus,
		)
		playerStats = append(playerStats, playerStat)
	}
	return NewBoxScoreScrapingRes(&playerStats, &errs)
}

func rescrapeLastNGameBoxScores(n int) error {
	games, err := db.SelectGamesPastNDays(n)
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	if err := scrapeBoxScores(games); err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func rescrapeBoxScoreErrors() error {
	games, err := db.SelectAllGamesWithPendingScrapingErrors()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	log.Printf("querying %d box scores...", len(games))
	// this is grotesque but idk there's a lot that can go wrong here
	stats := []*db.BoxScorePlayerStat{}
	statMu := sync.Mutex{}
	errMap := map[string][]*db.BoxScoreScrapingError{}
	errMu := sync.Mutex{}
	wg := sync.WaitGroup{}
	limiter := rate.NewLimiter(rate.Limit(5), 3)

	for _, g := range games {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Wait(context.Background()); err != nil {
				errMu.Lock()
				defer errMu.Unlock()
				if _, exists := errMap[g.ID]; !exists {
					errMap[g.ID] = []*db.BoxScoreScrapingError{}
				}
				scrapErr := db.NewBoxScoreScrapingError(g.ID, utils.ErrorWithTrace(err).Error())
				errMap[g.ID] = append(errMap[g.ID], scrapErr)
				return
			}
			res := scrapeBoxScore(g)
			statMu.Lock()
			defer statMu.Unlock()
			stats = append(stats, *res.PlayerStats...)
			if len(*res.Errors) > 0 {
				errMu.Lock()
				defer errMu.Unlock()
				if _, exists := errMap[g.ID]; !exists {
					errMap[g.ID] = []*db.BoxScoreScrapingError{}
				}
				errMap[g.ID] = append(errMap[g.ID], *res.Errors...)
			}
		}()
	}
	wg.Wait()

	derefStats := make([]db.BoxScorePlayerStat, 0, len(stats))
	for _, s := range stats {
		derefStats = append(derefStats, *s)
	}
	if err := db.InsertBoxScorePlayerStats(derefStats); err != nil {
		return utils.ErrorWithTrace(err)
	}

	resolvedIDs := []string{}
	for _, g := range games {
		if _, hasErr := errMap[g.ID]; hasErr {
			continue
		}
		resolvedIDs = append(resolvedIDs, g.ID)
	}
	if len(resolvedIDs) > 0 {
		if err := db.UpdateResolvedBoxScoreScrapingErrors(resolvedIDs); err != nil {
			return utils.ErrorWithTrace(err)
		}
	}
	errs := []db.BoxScoreScrapingError{}
	for _, errSlice := range errMap {
		for _, e := range errSlice {
			errs = append(errs, *e)
		}
	}
	if err := db.InsertBoxScoreScrapingErrors(errs); err != nil {
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
