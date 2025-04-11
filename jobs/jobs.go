package jobs

import (
	"crypto/md5"
	"errors"
	"fmt"
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

	"dunkod/db"
	"dunkod/nba"
	"dunkod/utils"
	"dunkod/youtube"
)

const titleCharLimit = 100
const descCharLimit = 5000

type Worker struct {
	Id     int
	Quit   chan bool
	IsIdle bool
}

func NewWorker(id int) *Worker {
	return &Worker{
		Id:     id,
		Quit:   make(chan bool, 1),
		IsIdle: true,
	}
}

func (w *Worker) DoYourJob(job *db.Job) {
	defer func() { w.IsIdle = true }()
	gameIDs := job.GamesIDs()
	playerIDs := job.PlayerIDs()

	assets, err := getAssets(job.Season, gameIDs, playerIDs)
	if err != nil {
		errorDetails := fmt.Errorf("WorkerID: %d\n\tJob Hash: %s\n\tError: %s", w.Id, job.Hash, err.Error())
		log.Println(errorDetails.Error())
		if err := job.OhNo(errorDetails); err != nil {
			log.Println(err)
		}
		return
	}

	assetURLs := make([]string, 0, len(assets))
	for _, a := range assets {
		if a.LargeUrl != nil {
			assetURLs = append(assetURLs, *a.LargeUrl)
		} else if a.MedUrl != nil {
			assetURLs = append(assetURLs, *a.MedUrl)
		} else if a.SmallUrl != nil {
			assetURLs = append(assetURLs, *a.SmallUrl)
		}
	}

	job.State = "DOWNLOADING CLIPS"
	if err := db.UpdateJob(job); err != nil {
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}

	sortAssetURLs(&assetURLs)
	vidPath, err := downloadAndConcat(assetURLs)
	defer func() { _ = os.Remove(vidPath) }()
	if err != nil {
		errorDetails := fmt.Errorf("WorkerID: %d\n\tJob Hash: %s\n\tError: %s", w.Id, job.Hash, err.Error())
		if err := job.OhNo(errorDetails); err != nil {
			log.Println(err)
		}
		return
	}

	playerNames, err := db.SelectPlayerNamesById(playerIDs)
	if err != nil {
		log.Println(err)
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}

	games, err := db.SelectGamesById(gameIDs)
	if err != nil {
		log.Println(err)
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}

	title := makeTitle(job.Season, games, playerNames)
	desc := makeDescription(job.Season, games, playerNames)

	job.State = "UPLOADING"
	if err := db.UpdateJob(job); err != nil {
		log.Println(err)
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}
	url, err := youtube.UploadFile(vidPath, title, desc, []string{"NBA", "nba", "basketball", "highlights", "sports", "Please Hire Me"})
	if err != nil {
		_ = os.Remove(vidPath)
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}
	if err := db.InsertVideo(db.NewVideo(title, desc, url, job.Id)); err != nil {
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
	}
	job.State = "FINISHED"
	if err := db.UpdateJob(job); err != nil {
		log.Println(err)
		if err := job.OhNo(err); err != nil {
			log.Println(err)
		}
		return
	}
}

func makeTitle(season string, games []db.DatabaseGame, playerNames []string) string {
	nameCharLimit := titleCharLimit/2 - 7
	gameCharLimit := titleCharLimit/2 - 7

	formatted := []string{}
	for _, p := range playerNames {
		split := strings.Split(p, " ")
		split = split[1:]
		lastName := strings.Join(split, " ")
		formatted = append(formatted, lastName)
	}

	namesList := strings.Join(formatted, ", ")
	if len(namesList) > nameCharLimit {
		namesList = namesList[:nameCharLimit-3]
		namesList += "..."
	}

	matchups := []string{}
	for _, g := range games {
		matchups = append(matchups, g.Matchup)
	}
	gamesList := strings.Join(matchups, ", ")
	if len(gamesList) > gameCharLimit {
		gamesList = gamesList[:gameCharLimit-3]
		gamesList += "..."
	}

	return namesList + " | " + gamesList + " | " + season
}

func makeDescription(season string, games []db.DatabaseGame, playerNames []string) string {
	matchups := make([]string, 0, len(games))
	for _, g := range games {
		matchupString := fmt.Sprintf("%s %s", g.Matchup, g.GameDate)
		matchups = append(matchups, matchupString)
	}
	matchupText := strings.Join(matchups, "\n")
	nameText := strings.Join(playerNames, "\n")

	desc := "Season: " + season + "\n\nPlayers:\n" + nameText + "\n\nGames:\n" + matchupText
	if len(desc) > descCharLimit {
		desc = desc[:descCharLimit-3]
		desc += "..."
	}
	return desc
}

func makeTags(season string, games []db.DatabaseGame, players []nba.CommonAllPlayer) []string {
	tags := []string{"NBA", "nba", "basketball", "highlights", "sports"}
	for _, g := range games {
		tags = append(tags, g.Matchup)
	}
	for _, p := range players {
		tags = append(tags, *p.DisplayFirstLast)
	}
	return tags
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
	// home, err := os.UserHomeDir()
	// if err != nil {
	// 	return "", utils.ErrorWithTrace(err)
	// }
	outputFileName := os.TempDir() + "/" + fmt.Sprintf("%x", sum) + ".mp4"

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

var contextMeasures = []nba.VideoDetailsAssetContextMeasure{
	nba.VideoDetailsAssetContextMeasures.FGA,
	nba.VideoDetailsAssetContextMeasures.REB,
	nba.VideoDetailsAssetContextMeasures.AST,
	nba.VideoDetailsAssetContextMeasures.STL,
	nba.VideoDetailsAssetContextMeasures.TOV,
	nba.VideoDetailsAssetContextMeasures.BLK,
}

func getAssets(season string, gameIDs []string, playerIDs []string) ([]nba.VideoDetailAsset, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided :%s", season))
	}
	assetChan := make(chan nba.VideoDetailAsset, 1024)
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

	assetMap := map[float64]nba.VideoDetailAsset{}
	for a := range assetChan {
		if a.EventID == nil {
			continue
		}
		assetMap[*a.EventID] = a
	}
	assets := make([]nba.VideoDetailAsset, 0, len(assetMap))
	for _, v := range assetMap {
		assets = append(assets, v)
	}

	return assets, nil
}

type Scheduler struct {
	Id           int
	MaxWorkers   int
	PollInterval time.Duration
	Workers      []*Worker
}

func NewScheduler(id int, maxWorkers int, pollInterval time.Duration) *Scheduler {
	s := Scheduler{
		Id:           id,
		MaxWorkers:   maxWorkers,
		PollInterval: pollInterval,
		Workers:      make([]*Worker, 0, maxWorkers),
	}
	for i := range maxWorkers {
		s.Workers = append(s.Workers, NewWorker(i))
	}
	return &s
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()
	for range ticker.C {
		w := s.GetIdleWorker()
		if w == nil {
			continue
		}
		w.IsIdle = false

		job, err := db.SelectJobForUpdate()
		if err != nil && strings.Contains(err.Error(), "QUEUE EMPTY") {
			w.IsIdle = true
			continue
		} else if err != nil {
			w.IsIdle = true
			log.Println(utils.ErrorWithTrace(err))
			continue
		}
		go w.DoYourJob(job)
	}
}

func (s *Scheduler) GetIdleWorker() *Worker {
	for _, w := range s.Workers {
		if w.IsIdle {
			return w
		}
	}
	return nil
}

func StalledJobsJanitory(duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for range ticker.C {
		if err := db.ResetStaleJobs(); err != nil {
			log.Println(err)
		}
	}
}
