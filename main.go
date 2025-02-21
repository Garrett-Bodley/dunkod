package main

import (
	"dunkod/config"
	"dunkod/db"
	"dunkod/nba"
	"dunkod/utils"

	"crypto/md5"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	Selected    []nba.CommonAllPlayer
	NotSelected []nba.CommonAllPlayer
}

func newPlayerData(selected, notSelected []nba.CommonAllPlayer) *PlayerData {
	return &PlayerData{
		Selected:    selected,
		NotSelected: notSelected,
	}
}

type VideoRequest struct {
	Season    string
	GameIDs   []string
	PlayerIDs []string
}

func newVideoRequest(season string, gameIDs []string, playerIDs []string) *VideoRequest {
	return &VideoRequest{
		Season:    season,
		GameIDs:   gameIDs,
		PlayerIDs: playerIDs,
	}
}

var playerCacheMu = sync.Mutex{}
var playerCache = map[string][]nba.CommonAllPlayer{}

func main() {
	// go scrapingDaemon()

	e := echo.New()
	e.Use(middleware.Logger())

	e.Renderer = newTemplate()

	e.GET("/", func(c echo.Context) error {
		season := "2024-25"
		games, err := db.SelectGamesBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		players, err := nba.CommonAllPlayersBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

		playerData := newPlayerData([]nba.CommonAllPlayer{}, players)
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

		allPlayers, err := getPlayersBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		playerData := newPlayerData([]nba.CommonAllPlayer{}, allPlayers)

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

		filteredGames := filterGamesByQuery(allGames, query)
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

		seasonPlayers, err := getPlayersBySeason(season)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}

		selected := []nba.CommonAllPlayer{}
		notSelected := []nba.CommonAllPlayer{}

	outer1:
		for _, p := range seasonPlayers {
			for _, id := range checked {
				if id == fmt.Sprintf("%.0f", *p.PersonID) {
					selected = append(selected, p)
					continue outer1
				}
			}
		}

		filtered := filterPlayersByQuery(seasonPlayers, query)
	outer2:
		for _, p := range filtered {
			for _, cid := range checked {
				if cid == fmt.Sprintf("%.0f", *p.PersonID) {
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

		// vidReq := newVideoRequest(season, gameIDs, playerIDs)
		assets, err := compileAssetURLs(season, gameIDs, playerIDs)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		if err := sortAssetURLs(&assets); err != nil {
			return utils.ErrorWithTrace(err)
		}

		vid, err := downloadAndConcat(assets)
		if err != nil {
			return utils.ErrorWithTrace(err)
		}
		fmt.Println(vid)
		return c.NoContent(200)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func getPlayersBySeason(season string) ([]nba.CommonAllPlayer, error) {
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
					ID:          k,
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

func filterGamesByQuery(games []db.DatabaseGame, query string) []db.DatabaseGame {
	filtered := []db.DatabaseGame{}
	query = strings.ToLower(query)
	for _, g := range games {
		searchString := strings.ToLower(fmt.Sprintf("%s,%s,%s", g.Matchup, g.WinnerName, g.LoserName))
		if strings.Contains(searchString, query) {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

func filterPlayersByQuery(players []nba.CommonAllPlayer, query string) []nba.CommonAllPlayer {
	filtered := []nba.CommonAllPlayer{}
	query = strings.ToLower(query)
	for _, p := range players {
		searchString := strings.ToLower(fmt.Sprintf("%s,%s,%s,%s,%s,%s",
			*p.DisplayFirstLast,
			*p.PlayerCode,
			*p.TeamName,
			*p.TeamCity,
			*p.TeamSlug,
			*p.TeamAbbreviation,
		))
		if strings.Contains(searchString, query) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

var contextMeasures = []nba.VideoDetailsAssetContextMeasure{
	nba.VideoDetailsAssetContextMeasures.FGA,
	nba.VideoDetailsAssetContextMeasures.REB,
	nba.VideoDetailsAssetContextMeasures.AST,
	nba.VideoDetailsAssetContextMeasures.STL,
	nba.VideoDetailsAssetContextMeasures.TOV,
	nba.VideoDetailsAssetContextMeasures.BLK,
}

func compileAssetURLs(season string, gameIDs []string, playerIDs []string) ([]string, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided :%s", season))
	}
	assetChan := make(chan string, 1024)
	errChan := make(chan error, 1024)
	wg := sync.WaitGroup{}

	for _, gid := range gameIDs {
		for _, pid := range playerIDs {
			for _, m := range contextMeasures {
				time.Sleep(200 * time.Millisecond)
				wg.Add(1)
				go func() {
					defer wg.Done()
					assets, err := nba.VideoDetailsAsset(gid, pid, m)
					if err != nil {
						errChan <- utils.ErrorWithTrace(err)
					}
					for _, a := range assets {
						if a.LargeUrl != nil {
							assetChan <- *a.LargeUrl
						} else if a.MedUrl != nil {
							assetChan <- *a.MedUrl
						} else if a.SmallUrl != nil {
							assetChan <- *a.SmallUrl
						}
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

	urls := make([]string, 0, len(assetChan))
	for a := range assetChan {
		urls = append(urls, a)
	}
	return urls, nil
}

func downloadAndConcat(urls []string) (string, error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}

	wg := sync.WaitGroup{}
	errChan := make(chan error, 1024)

	for i, u := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fileName := fmt.Sprintf("%s/%04d.mp4", tmpDir, i)
			if err := utils.CurlToFile(u, fileName); err != nil {
				errChan <- utils.ErrorWithTrace(err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		errs := make([]error, 0, len(errChan))
		for err := range errChan {
			errs = append(errs, err)
		}
		_ = os.RemoveAll(tmpDir)
		return "", utils.ErrorWithTrace(errors.Join(errs...))
	}

	vid, err := ffmpegConcat(tmpDir)
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}

	return vid, nil
}

func sortAssetURLs(assets *[]string) error {
	re := regexp.MustCompile(`(?:https:\/\/videos.nba.com\/nba\/pbp\/media\/\d+\/\d+\/\d+\/)(\d+)\/(\d+)`)
	errs := []error{}
	slices.SortStableFunc(*assets, func(a, b string) int {
		matchesA := re.FindStringSubmatch(a)
		matchesB := re.FindStringSubmatch(b)

		sortNumA := matchesA[1] + fmt.Sprintf("%03s", matchesA[2])
		sortNumB := matchesB[1] + fmt.Sprintf("%03s", matchesB[2])

		numA, err := strconv.Atoi(sortNumA)
		if err != nil {
			errs = append(errs, err)
			return 0
		}
		numB, err := strconv.Atoi(sortNumB)
		if err != nil {
			errs = append(errs, err)
			return 0
		}

		return numA - numB
	})
	if len(errs) > 0 {
		errors.Join(errs...)
	}
	return nil
}

// ffmpeg is written in c and assembly language
func ffmpegConcat(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}

	listName := fmt.Sprintf("%s/files.txt", dir)
	list, err := os.Create(listName)
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}
	defer list.Close()

	for _, f := range files {
		_, err := list.Write([]byte(fmt.Sprintf("file '%s'\n", f.Name())))
		if err != nil {
			return "", utils.ErrorWithTrace(err)
		}
	}

	timeString := fmt.Sprintf("%d%d", time.Now().Unix(), rand.Intn(math.MaxInt64))
	sum := md5.Sum([]byte(timeString))
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}
	outputFileName := home + "/Downloads/" + fmt.Sprintf("%x", sum) + ".mp4"

	fmt.Println(dir, listName)

	args := []string{"-hide_banner", "-v", "fatal", "-f", "concat", "-safe", "0", "-vsync", "0", "-i", fmt.Sprintf("%s/files.txt", dir), "-c", "copy", outputFileName}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		_ = os.Remove(outputFileName)
		return "", utils.ErrorWithTrace(err)
	}

	return outputFileName, nil
}
