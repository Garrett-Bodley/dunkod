package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"dunkod/config"
	"dunkod/db"
	"dunkod/jobs"
	"dunkod/nba"
	"dunkod/scrape"
	"dunkod/utils"
	"dunkod/youtube"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
	return &Templates{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"derefFloat64": derefFloat64,
		}).ParseGlob("views/*.html")),
	}
}

func derefFloat64(f *float64) float64 {
	if f == nil {
		return float64(0)
	}
	return *f
}

type State struct {
	Season       string
	ValidSeasons []string
	GameData     *GameData
	PlayerData   *PlayerData
	Error        string
}

func newState(season string, validSeasons []string, gameData *GameData, playerData *PlayerData) *State {
	return &State{
		Season:       season,
		ValidSeasons: validSeasons,
		GameData:     gameData,
		PlayerData:   playerData,
	}
}

type GameData struct {
	Selected    []db.DatabaseGame
	NotSelected []db.DatabaseGame
}

func newGameData(selected, notSelected []db.DatabaseGame) *GameData {
	return &GameData{
		Selected:    selected,
		NotSelected: notSelected,
	}
}

type PlayerData struct {
	Selected    []db.PlayerSearchInfo
	NotSelected []db.PlayerSearchInfo
}

func newPlayerData(selected, notSelected []db.PlayerSearchInfo) *PlayerData {
	return &PlayerData{
		Selected:    selected,
		NotSelected: notSelected,
	}
}

type JobState struct {
	Players []string
	Games   []string
	Job     *db.Job
	Video   *db.Video
	Error   string
}

func newJobState(job *db.Job) *JobState {
	return &JobState{
		Job:     job,
		Players: []string{},
		Games:   []string{},
		Error:   "",
	}
}

var sigChan = make(chan os.Signal, 1)

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
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt, syscall.SIGINT)
	go cleanup()
	if err := youtube.InitService(); err != nil {
		panic(err)
	}
	if *config.BigScrape {
		if err := scrape.BigScrape(); err != nil {
			panic(err)
		}
		os.Exit(0)
	}
	go scrape.ScrapingDaemon(30 * time.Minute)
	go youtube.ServiceJanitor(8 * time.Hour)
	go jobs.StalledJobsJanitory(5 * time.Minute)
	fmt.Println("The New York Knickerbockers are named after pants")
}

func cleanup() {
	<-sigChan
	fmt.Println("\nclosing database...")
	if err := db.Close(); err != nil {
		panic(err)
	}
	os.Exit(0)
}

func main() {
	scheduler1 := jobs.NewScheduler(0, 2, time.Second*2)
	scheduler2 := jobs.NewScheduler(0, 2, time.Second*2)
	go scheduler1.Start()
	go scheduler2.Start()

	e := echo.New()
	e.Use(middleware.Logger())

	e.Renderer = newTemplate()
	e.Static("/static", "static")

	e.GET("/", func(c echo.Context) error {
		season := "2024-25"
		games, err := db.SelectGamesBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		players, err := db.GetPlayerPlayerSearchInfoBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

		playerData := newPlayerData([]db.PlayerSearchInfo{}, players)
		gameData := newGameData([]db.DatabaseGame{}, games)

		state := newState(season, config.ValidSeasons, gameData, playerData)

		return c.Render(200, "index", state)
	})

	e.POST("/season", func(c echo.Context) error {
		season := c.Request().FormValue("season")
		allGames, err := db.SelectGamesBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		gameData := newGameData([]db.DatabaseGame{}, allGames)

		allPlayers, err := db.GetPlayerPlayerSearchInfoBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		playerData := newPlayerData([]db.PlayerSearchInfo{}, allPlayers)

		state := newState(season, config.ValidSeasons, gameData, playerData)

		return c.Render(200, "games-and-players", state)
	})

	e.POST("/game-search", func(c echo.Context) error {
		req := c.Request()
		if err := req.ParseForm(); err != nil {
			return utils.ErrorWithTrace(err)
		}

		selectedGameIDs := req.Form["game"]
		season := req.FormValue("season")
		query := req.FormValue("game-search")

		selectedGames := []db.DatabaseGame{}
		notSelectedGames := []db.DatabaseGame{}

		allGames, err := db.SelectGamesBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

	outer1:
		for _, g := range allGames {
			for _, id := range selectedGameIDs {
				if g.ID == id {
					selectedGames = append(selectedGames, g)
					continue outer1
				}
			}
		}

		filteredGames, err := filterGamesByQuery(allGames, query)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
	outer2:
		for _, g := range filteredGames {
			for _, id := range selectedGameIDs {
				if g.ID == id {
					continue outer2
				}
			}
			notSelectedGames = append(notSelectedGames, g)
		}

		gameData := newGameData(selectedGames, notSelectedGames)
		return c.Render(200, "game-options", gameData)
	})

	e.POST("/player-search", func(c echo.Context) error {
		req := c.Request()
		if err := req.ParseForm(); err != nil {
			return utils.ErrorWithTrace(err)
		}

		query := req.FormValue("player-search")
		season := req.FormValue("season")
		checked := req.Form["player"]

		seasonPlayers, err := db.GetPlayerPlayerSearchInfoBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

		selected := []db.PlayerSearchInfo{}
		notSelected := []db.PlayerSearchInfo{}

	outer1:
		for _, p := range seasonPlayers {
			for _, id := range checked {
				if id == fmt.Sprintf("%d", p.PlayerID) {
					selected = append(selected, p)
					continue outer1
				}
			}
		}

		filtered, err := filterPlayersByQuery(seasonPlayers, query)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
	outer2:
		for _, p := range filtered {
			for _, cid := range checked {
				if cid == fmt.Sprintf("%d", p.PlayerID) {
					continue outer2
				}
			}
			notSelected = append(notSelected, p)
		}

		playerData := newPlayerData(selected, notSelected)

		return c.Render(200, "player-options", playerData)
	})

	e.POST("/", func(c echo.Context) error {
		req := c.Request()
		if err := req.ParseForm(); err != nil {
			return utils.ErrorWithTrace(err)
		}

		season := req.FormValue("season")
		gameIDs := req.Form["game"]
		playerIDs := req.Form["player"]

		assets, err := getAssets(season, gameIDs, playerIDs)
		if err != nil {
			log.Println(utils.ErrorWithTrace(err))
			return c.Render(200, "error", "unable to process request "+utils.Sad)
		}
		if len(assets) == 0 {
			return c.Render(200, "error", "no assets found "+utils.Sad)
		}
		job := db.NewJob(playerIDs, gameIDs, season)
		job, err = db.InsertJob(job)
		if err != nil {
			return c.Render(200, "error", err.Error())
		}

		redirect := fmt.Sprintf("/%s", job.Slug)
		c.Response().Header().Set("HX-Redirect", redirect)
		return c.NoContent(200)
	})

	e.GET("/:slug", func(c echo.Context) error {
		slug := c.Param("slug")
		job, err := db.SelectJobBySlug(slug)
		if err != nil {
			jobState := newJobState(nil)
			jobState.Error = err.Error()
			return c.Render(200, "job", jobState)
		}
		jobState := newJobState(job)

		games, err := db.SelectGamesById(strings.Split(job.Games, ","))
		if err != nil {
			jobState.Error = err.Error()
			return c.Render(200, "job", jobState)
		}
		if len(games) == 0 {
			jobState.Error = "did not find any games " + utils.Sad
			return c.Render(200, "job", jobState)
		}
		matchups := []string{}
		for _, g := range games {
			matchups = append(matchups, fmt.Sprintf("%s %s", g.Matchup, g.GameDate))
		}
		jobState.Games = matchups

		playerIds := []int{}
		for _, idString := range strings.Split(job.Players, ",") {
			id, err := strconv.Atoi(idString)
			if err != nil {
				jobState.Error = err.Error()
				return c.Render(200, "job", jobState)
			}
			playerIds = append(playerIds, id)
		}
		players, err := db.GetPlayerPlayerSearchInfoBySeason(job.Season)
		if err != nil {
			jobState.Error = err.Error()
			return c.Render(200, "job", jobState)
		}
		playerNames := make([]string, 0, len(playerIds))
		for _, p := range players {
			for _, id := range playerIds {
				if id == p.PlayerID {
					playerNames = append(playerNames, p.PlayerName)
				}
			}
		}
		jobState.Players = playerNames

		if job.State == "FINISHED" {
			video, err := db.SelectVideoByJobId(job.Id)
			if err != nil {
				jobState.Error = err.Error()
				return c.Render(200, "job", jobState)
			}
			jobState.Video = video
		}

		return c.Render(200, "job", jobState)
	})

	e.POST("/:slug/status/:state", func(c echo.Context) error {
		slug := c.Param("slug")
		state := c.Param("state")
		job, err := db.SelectJobBySlug(slug)
		redirect := fmt.Sprintf("/%s", slug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.NoContent(404)
			} else {
				c.Response().Header().Set("HX-Redirect", redirect)
				return c.NoContent(200)

			}
		}
		if job.State == state {
			return c.NoContent(204)
		}

		if job.State == "FINISHED" {
			c.Response().Header().Set("HX-Redirect", redirect)
			return c.NoContent(200)
		}
		return c.Render(200, "state", job)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func filterGamesByQuery(games []db.DatabaseGame, query string) ([]db.DatabaseGame, error) {
	filtered := []db.DatabaseGame{}
	words := strings.Fields(strings.ToLower(utils.RemoveDiacritics(query)))
	pattern := strings.Join(words, "|")
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	for _, g := range games {
		searchString := strings.ToLower(fmt.Sprintf("%s,%s,%s", g.Matchup, g.WinnerName, g.LoserName))
		if re.Match([]byte(searchString)) {
			filtered = append(filtered, g)
		}
	}
	return filtered, nil
}

func filterPlayersByQuery(players []db.PlayerSearchInfo, query string) ([]db.PlayerSearchInfo, error) {
	filtered := []db.PlayerSearchInfo{}
	words := strings.Fields(strings.ToLower(utils.RemoveDiacritics(query)))
	pattern := strings.Join(words, "|")
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	for _, p := range players {
		searchString := p.SearchString()
		if re.Match([]byte(searchString)) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

var contextMeasures = []nba.VideoDetailsAssetContextMeasure{
	nba.VideoDetailsAssetContextMeasures.FGA,
	nba.VideoDetailsAssetContextMeasures.REB,
	nba.VideoDetailsAssetContextMeasures.AST,
	nba.VideoDetailsAssetContextMeasures.STL,
	nba.VideoDetailsAssetContextMeasures.TOV,
	nba.VideoDetailsAssetContextMeasures.BLK,
}

func getAssets(season string, gameIDs []string, playerIDs []string) ([]nba.VideoDetailsAssetEntry, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: '%s' "+utils.Sad, season))
	}
	assetChan := make(chan nba.VideoDetailsAssetEntry, 1024)
	errChan := make(chan error, 1024)
	wg := sync.WaitGroup{}

	for _, gid := range gameIDs {
		for _, pid := range playerIDs {
			for _, m := range contextMeasures {
				time.Sleep(200 * time.Millisecond)
				wg.Add(1)
				go func() {
					defer wg.Done()
					assets, err := nba.VideoDetailsAsset(season, gid, pid, m)
					if err != nil {
						errChan <- utils.ErrorWithTrace(err)
					}
					for _, a := range assets {
						assetChan <- a
					}
				}()
			}
		}
	}

	wg.Wait()
	close(errChan)
	close(assetChan)

	if len(errChan) > 0 {
		errs := []error{}
		for e := range errChan {
			errs = append(errs, e)
		}
		return nil, errors.Join(errs...)
	}

	assetMap := map[float64]nba.VideoDetailsAssetEntry{}
	for a := range assetChan {
		if a.EventID == nil {
			continue
		}
		assetMap[*a.EventID] = a
	}
	assets := make([]nba.VideoDetailsAssetEntry, 0, len(assetMap))
	for _, v := range assetMap {
		assets = append(assets, v)
	}

	return assets, nil
}
