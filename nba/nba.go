package nba

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"dunkod/utils"

	"golang.org/x/time/rate"
)

var playerCacheMu = sync.Mutex{}
var playerCache = map[string][]CommonAllPlayer{}

type CommonAllPlayersResp struct {
	ResultSets []struct {
		Headers []string        `json:"headers"`
		RowSet  [][]interface{} `json:"rowSet"`
	} `json:"resultSets"`
}

type CommonAllPlayer struct {
	PersonID                *float64
	DisplayLastFirst        *string
	DisplayFirstLast        *string
	RosterStatus            *float64
	FromYear                *string
	ToYear                  *string
	PlayerCode              *string
	PlayerSlug              *string
	TeamID                  *float64
	TeamCity                *string
	TeamName                *string
	TeamAbbreviation        *string
	TeamCode                *string
	TeamSlug                *string
	GamesPlayedFlag         *string
	OtherLeagueExperienceCh *string
}

type CommonAllPlayerJSON struct {
	PersonID                float64 `json:"personID,omitempty"`
	DisplayLastFirst        string  `json:"displayLastFirst,omitempty"`
	DisplayFirstLast        string  `json:"displayFirstLast,omitempty"`
	RosterStatus            float64 `json:"rosterStatus,omitempty"`
	FromYear                string  `json:"fromYear,omitempty"`
	ToYear                  string  `json:"toYear,omitempty"`
	PlayerCode              string  `json:"playerCode,omitempty"`
	PlayerSlug              string  `json:"playerSlug,omitempty"`
	TeamID                  float64 `json:"teamID,omitempty"`
	TeamCity                string  `json:"teamCity,omitempty"`
	TeamName                string  `json:"teamName,omitempty"`
	TeamAbbreviation        string  `json:"teamAbbreviation,omitempty"`
	TeamCode                string  `json:"teamCode,omitempty"`
	TeamSlug                string  `json:"teamSlug,omitempty"`
	GamesPlayedFlag         string  `json:"gamesPlayedFlag,omitempty"`
	OtherLeagueExperienceCh string  `json:"otherLeagueExperienceCh,omitempty"`
}

func (p CommonAllPlayer) LogNilFields() {
	errs := []error{}
	if p.PersonID == nil {
		errs = append(errs, fmt.Errorf("nil field 'PersonID'"))
	}
	if p.DisplayLastFirst == nil {
		errs = append(errs, fmt.Errorf("nil field 'DisplayLastFirst'"))
	}
	if p.DisplayFirstLast == nil {
		errs = append(errs, fmt.Errorf("nil field 'DisplayFirstLast'"))
	}
	if p.RosterStatus == nil {
		errs = append(errs, fmt.Errorf("nil field 'RosterStatus'"))
	}
	if p.FromYear == nil {
		errs = append(errs, fmt.Errorf("nil field 'FromYear'"))
	}
	if p.ToYear == nil {
		errs = append(errs, fmt.Errorf("nil field 'ToYear'"))
	}
	if p.PlayerCode == nil {
		errs = append(errs, fmt.Errorf("nil field 'PlayerCode'"))
	}
	if p.PlayerSlug == nil {
		errs = append(errs, fmt.Errorf("nil field 'PlayerSlug'"))
	}
	if p.TeamID == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamID'"))
	}
	if p.TeamCity == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamCity'"))
	}
	if p.TeamName == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamName'"))
	}
	if p.TeamAbbreviation == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamAbbreviation'"))
	}
	if p.TeamCode == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamCode'"))
	}
	if p.TeamSlug == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamSlug'"))
	}
	if p.GamesPlayedFlag == nil {
		errs = append(errs, fmt.Errorf("nil field 'GamesPlayedFlag'"))
	}
	if p.OtherLeagueExperienceCh == nil {
		errs = append(errs, fmt.Errorf("nil field 'OtherLeagueExperienceCh'"))
	}
	if len(errs) == 0 {
		return
	}

	if p.DisplayFirstLast != nil {
		log.Printf("%s:\n\t%v", *p.DisplayFirstLast, errors.Join(errs...))
	} else if p.DisplayLastFirst != nil {
		log.Printf("%s:\n\t%v", *p.DisplayLastFirst, errors.Join(errs...))
	} else if p.PlayerSlug != nil {
		log.Printf("%s:\n\t%v", *p.PlayerSlug, errors.Join(errs...))
	} else if p.PersonID != nil {
		log.Printf("MISSING NAME. ID: %s:\n\t%v", *p.PlayerCode, errors.Join(errs...))
	} else {
		log.Printf("MISSING ALL IDENTIFYING INFORMATION LOL\n\t%v", errors.Join(errs...))
	}
}

func (p CommonAllPlayer) ToJSON() CommonAllPlayerJSON {
	json := CommonAllPlayerJSON{}
	if p.PersonID != nil {
		json.PersonID = *p.PersonID
	}
	if p.DisplayLastFirst != nil {
		json.DisplayLastFirst = *p.DisplayLastFirst
	}
	if p.DisplayFirstLast != nil {
		json.DisplayFirstLast = *p.DisplayFirstLast
	}
	if p.RosterStatus != nil {
		json.RosterStatus = *p.RosterStatus
	}
	if p.FromYear != nil {
		json.FromYear = *p.FromYear
	}
	if p.ToYear != nil {
		json.ToYear = *p.ToYear
	}
	if p.PlayerCode != nil {
		json.PlayerCode = *p.PlayerCode
	}
	if p.PlayerSlug != nil {
		json.PlayerSlug = *p.PlayerSlug
	}
	if p.TeamID != nil {
		json.TeamID = *p.TeamID
	}
	if p.TeamCity != nil {
		json.TeamCity = *p.TeamCity
	}
	if p.TeamName != nil {
		json.TeamName = *p.TeamName
	}
	if p.TeamAbbreviation != nil {
		json.TeamAbbreviation = *p.TeamAbbreviation
	}
	if p.TeamCode != nil {
		json.TeamCode = *p.TeamCode
	}
	if p.TeamSlug != nil {
		json.TeamSlug = *p.TeamSlug
	}
	if p.GamesPlayedFlag != nil {
		json.GamesPlayedFlag = *p.GamesPlayedFlag
	}
	if p.OtherLeagueExperienceCh != nil {
		json.OtherLeagueExperienceCh = *p.OtherLeagueExperienceCh
	}
	return json
}

func CommonAllPlayerAllSeasons() ([]CommonAllPlayer, error) {
	url := "https://stats.nba.com/stats/commonallplayers?LeagueID=00&Season=2024-25&IsOnlyCurrentSeason=0"
	body, err := curl(url)
	if err != nil {
		return nil, err
	}
	unmarshalledBody := CommonAllPlayersResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	expectedHeaders := []string{
		"PERSON_ID",
		"DISPLAY_LAST_COMMA_FIRST",
		"DISPLAY_FIRST_LAST",
		"ROSTERSTATUS",
		"FROM_YEAR",
		"TO_YEAR",
		"PLAYERCODE",
		"PLAYER_SLUG",
		"TEAM_ID",
		"TEAM_CITY",
		"TEAM_NAME",
		"TEAM_ABBREVIATION",
		"TEAM_CODE",
		"TEAM_SLUG",
		"GAMES_PLAYED_FLAG",
		"OTHERLEAGUE_EXPERIENCE_CH",
	}

	receivedHeaders := unmarshalledBody.ResultSets[0].Headers
	if err := validateHeaders(expectedHeaders, receivedHeaders); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	players := make([]CommonAllPlayer, len(unmarshalledBody.ResultSets[0].RowSet))
	for i, raw := range unmarshalledBody.ResultSets[0].RowSet {
		player := CommonAllPlayer{
			PersonID:                maybe[float64](raw[0]),
			DisplayLastFirst:        maybe[string](raw[1]),
			DisplayFirstLast:        maybe[string](raw[2]),
			RosterStatus:            maybe[float64](raw[3]),
			FromYear:                maybe[string](raw[4]),
			ToYear:                  maybe[string](raw[5]),
			PlayerCode:              maybe[string](raw[6]),
			PlayerSlug:              maybe[string](raw[7]),
			TeamID:                  maybe[float64](raw[8]),
			TeamCity:                maybe[string](raw[9]),
			TeamName:                maybe[string](raw[10]),
			TeamAbbreviation:        maybe[string](raw[11]),
			TeamCode:                maybe[string](raw[12]),
			TeamSlug:                maybe[string](raw[13]),
			GamesPlayedFlag:         maybe[string](raw[14]),
			OtherLeagueExperienceCh: maybe[string](raw[15]),
		}
		// player.LogNilFields()
		players[i] = player
	}
	return players, nil
}

func CommonAllPlayersBySeason(season string) ([]CommonAllPlayer, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}

	url := fmt.Sprintf("https://stats.nba.com/stats/commonallplayers?LeagueID=00&Season=%s&IsOnlyCurrentSeason=1", season)
	body, err := curl(url)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	unmarshalledBody := CommonAllPlayersResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	players := make([]CommonAllPlayer, len(unmarshalledBody.ResultSets[0].RowSet))
	for i, raw := range unmarshalledBody.ResultSets[0].RowSet {
		player := CommonAllPlayer{
			PersonID:                maybe[float64](raw[0]),
			DisplayLastFirst:        maybe[string](raw[1]),
			DisplayFirstLast:        maybe[string](raw[2]),
			RosterStatus:            maybe[float64](raw[3]),
			FromYear:                maybe[string](raw[4]),
			ToYear:                  maybe[string](raw[5]),
			PlayerCode:              maybe[string](raw[6]),
			PlayerSlug:              maybe[string](raw[7]),
			TeamID:                  maybe[float64](raw[8]),
			TeamCity:                maybe[string](raw[9]),
			TeamName:                maybe[string](raw[10]),
			TeamAbbreviation:        maybe[string](raw[11]),
			TeamCode:                maybe[string](raw[12]),
			TeamSlug:                maybe[string](raw[13]),
			GamesPlayedFlag:         maybe[string](raw[14]),
			OtherLeagueExperienceCh: maybe[string](raw[15]),
		}
		// player.LogNilFields()
		players[i] = player
	}
	return players, nil
}

// https://stats.nba.com/stats/leaguegamelog?Counter=0&Direction=DESC&LeagueID=00&PlayerOrTeam=T&Season=2024-25&SeasonType=Regular+Season&Sorter=DATE

type LeagueGameLogResp struct {
	ResultSets []struct {
		Headers []string        `json:"headers"`
		RowSet  [][]interface{} `json:"rowSet"`
	} `json:"resultSets"`
}

type LeagueGameLogGame struct {
	SeasonID         *string
	TeamID           *float64
	TeamAbbreviation *string
	TeamName         *string
	GameID           *string
	GameDate         *string
	Matchup          *string
	WL               *string
	MIN              *float64
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TOV              *float64
	PF               *float64
	PTS              *float64
	PlusMinus        *float64
	VideoAvailable   *float64 // 1 is NOT AVAILABLE, 0 is AVAILABLE
}

func (g *LeagueGameLogGame) LogNilFields() {
	errs := []error{}
	if g.SeasonID == nil {
		errs = append(errs, fmt.Errorf("nil field 'SeasonID'"))
	}
	if g.TeamID == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamID'"))
	}
	if g.TeamAbbreviation == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamAbbreviation'"))
	}
	if g.TeamName == nil {
		errs = append(errs, fmt.Errorf("nil field 'TeamName'"))
	}
	if g.GameID == nil {
		errs = append(errs, fmt.Errorf("nil field 'GameID'"))
	}
	if g.GameDate == nil {
		errs = append(errs, fmt.Errorf("nil field 'GameDate'"))
	}
	if g.Matchup == nil {
		errs = append(errs, fmt.Errorf("nil field 'Matchup'"))
	}
	if g.WL == nil {
		errs = append(errs, fmt.Errorf("nil field 'WL'"))
	}
	if g.MIN == nil {
		errs = append(errs, fmt.Errorf("nil field 'MIN'"))
	}
	if g.FGM == nil {
		errs = append(errs, fmt.Errorf("nil field 'FGM'"))
	}
	if g.FGA == nil {
		errs = append(errs, fmt.Errorf("nil field 'FGA'"))
	}
	if g.FG_PCT == nil {
		errs = append(errs, fmt.Errorf("nil field 'FG_PCT'"))
	}
	if g.FG3M == nil {
		errs = append(errs, fmt.Errorf("nil field 'FG3M'"))
	}
	if g.FG3A == nil {
		errs = append(errs, fmt.Errorf("nil field 'FG3A'"))
	}
	if g.FG3_PCT == nil {
		errs = append(errs, fmt.Errorf("nil field 'FG3_PCT'"))
	}
	if g.FTM == nil {
		errs = append(errs, fmt.Errorf("nil field 'FTM'"))
	}
	if g.FTA == nil {
		errs = append(errs, fmt.Errorf("nil field 'FTA'"))
	}
	if g.FT_PCT == nil {
		errs = append(errs, fmt.Errorf("nil field 'FT_PCT'"))
	}
	if g.OREB == nil {
		errs = append(errs, fmt.Errorf("nil field 'OREB'"))
	}
	if g.DREB == nil {
		errs = append(errs, fmt.Errorf("nil field 'DREB'"))
	}
	if g.REB == nil {
		errs = append(errs, fmt.Errorf("nil field 'REB'"))
	}
	if g.AST == nil {
		errs = append(errs, fmt.Errorf("nil field 'AST'"))
	}
	if g.STL == nil {
		errs = append(errs, fmt.Errorf("nil field 'STL'"))
	}
	if g.BLK == nil {
		errs = append(errs, fmt.Errorf("nil field 'BLK'"))
	}
	if g.TOV == nil {
		errs = append(errs, fmt.Errorf("nil field 'TOV'"))
	}
	if g.PF == nil {
		errs = append(errs, fmt.Errorf("nil field 'PF'"))
	}
	if g.PTS == nil {
		errs = append(errs, fmt.Errorf("nil field 'PTS'"))
	}
	if g.PlusMinus == nil {
		errs = append(errs, fmt.Errorf("nil field 'PlusMinus'"))
	}
	if g.VideoAvailable == nil {
		errs = append(errs, fmt.Errorf("nil field 'VideoAvailable'"))
	}

	if len(errs) == 0 {
		return
	}

	if g.Matchup != nil && g.GameDate != nil {
		log.Printf("%s %s:\n\t%v", *g.Matchup, *g.GameDate, errors.Join(errs...))
	} else if g.Matchup != nil {
		log.Printf("%s:\n\t%v", *g.Matchup, errors.Join(errs...))
	} else if g.GameID != nil {
		log.Printf("GameID: %s\n\t%v", *g.GameID, errors.Join(errs...))
	} else {
		log.Printf("NO IDENTIFYING INFO:\n\t%v", errors.Join(errs...))
	}
}

func LeagueGameLog(season string, seasonType string) ([]LeagueGameLogGame, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}

	url := fmt.Sprintf("https://stats.nba.com/stats/leaguegamelog?Counter=0&Direction=DESC&LeagueID=00&PlayerOrTeam=T&Season=%s&SeasonType=%s&Sorter=DATE", season, seasonType)
	body, err := curl(url)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	unmarshalledBody := LeagueGameLogResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	expectedHeaders := []string{
		"SEASON_ID",
		"TEAM_ID",
		"TEAM_ABBREVIATION",
		"TEAM_NAME",
		"GAME_ID",
		"GAME_DATE",
		"MATCHUP",
		"WL",
		"MIN",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TOV",
		"PF",
		"PTS",
		"PLUS_MINUS",
		"VIDEO_AVAILABLE",
	}

	receivedHeaders := unmarshalledBody.ResultSets[0].Headers
	if err := validateHeaders(expectedHeaders, receivedHeaders); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	leagueGameLogGames := make([]LeagueGameLogGame, len(unmarshalledBody.ResultSets[0].RowSet))
	for i, raw := range unmarshalledBody.ResultSets[0].RowSet {
		leagueGameLogGames[i] = LeagueGameLogGame{
			SeasonID:         maybe[string](raw[0]),
			TeamID:           maybe[float64](raw[1]),
			TeamAbbreviation: maybe[string](raw[2]),
			TeamName:         maybe[string](raw[3]),
			GameID:           maybe[string](raw[4]),
			GameDate:         maybe[string](raw[5]),
			Matchup:          maybe[string](raw[6]),
			WL:               maybe[string](raw[7]),
			MIN:              maybe[float64](raw[8]),
			FGM:              maybe[float64](raw[9]),
			FGA:              maybe[float64](raw[10]),
			FG_PCT:           maybe[float64](raw[11]),
			FG3M:             maybe[float64](raw[12]),
			FG3A:             maybe[float64](raw[13]),
			FG3_PCT:          maybe[float64](raw[14]),
			FTM:              maybe[float64](raw[15]),
			FTA:              maybe[float64](raw[16]),
			FT_PCT:           maybe[float64](raw[17]),
			OREB:             maybe[float64](raw[18]),
			DREB:             maybe[float64](raw[19]),
			REB:              maybe[float64](raw[20]),
			AST:              maybe[float64](raw[21]),
			STL:              maybe[float64](raw[22]),
			BLK:              maybe[float64](raw[23]),
			TOV:              maybe[float64](raw[24]),
			PF:               maybe[float64](raw[25]),
			PTS:              maybe[float64](raw[26]),
			PlusMinus:        maybe[float64](raw[27]),
			VideoAvailable:   maybe[float64](raw[28]),
		}
		// leagueGameLogGames[i].LogNilFields()
	}

	return leagueGameLogGames, nil
}

func DedupeLeagueGameLogGames(games []LeagueGameLogGame) ([]LeagueGameLogGame, error) {
	set := map[string]LeagueGameLogGame{}
	for _, g := range games {
		if _, exists := set[*g.GameID]; exists {
			continue
		}
		set[*g.GameID] = g
	}

	res := make([]LeagueGameLogGame, 0, len(set))
	for _, v := range set {
		res = append(res, v)
	}

	slices.SortStableFunc(res, func(a, b LeagueGameLogGame) int {
		if *a.GameDate < *b.GameDate {
			return 1
		} else {
			return -1
		}
	})
	return res, nil
}

type VideoDetailAsset struct {
	GameID      *string
	EventID     *float64
	Year        *float64
	Month       *string
	Day         *string
	Description *string
	Uuid        *string
	LargeUrl    *string
	MedUrl      *string
	SmallUrl    *string
}

type VideoDetailsAssetResp struct {
	ResultSets struct {
		Meta struct {
			VideoUrls []VideoDetailsAssetURLEntry `json:"videoUrls"`
		} `json:"Meta"`
		Playlist []VideoDetailsAssetPlaylistEntry `json:"playlist"`
	} `json:"resultSets"`
}

type VideoDetailsAssetURLEntry struct {
	Uuid           *string  `json:"uuid"`
	SmallDur       *float64 `json:"sdur"`
	SmallUrl       *string  `json:"surl"`
	SmallThumbnail *string  `json:"sth"`
	MedDur         *float64 `json:"mdur"`
	MedUrl         *string  `json:"murl"`
	MedThumbnail   *string  `json:"mth"`
	LargeDur       *float64 `json:"ldur"`
	LargeUrl       *string  `json:"lurl"`
	LargeThumbnail *string  `json:"lth"`
	Vtt            *string  `json:"vtt"`
	Scc            *string  `json:"scc"`
	Srt            *string  `json:"srt"`
}

type VideoDetailsAssetPlaylistEntry struct {
	GameID               *string  `json:"gi"`
	EventID              *float64 `json:"ei"`
	Year                 *float64 `json:"y"`
	Month                *string  `json:"m"`
	Day                  *string  `json:"d"`
	GameCode             *string  `json:"gc"`
	Period               *float64 `json:"p"`
	Description          *string  `json:"dsc"`
	HomeAbbreviation     *string  `json:"ha"`
	HomeID               *float64 `json:"hid"`
	VisitingAbbreviation *string  `json:"va"`
	VisitingID           *float64 `json:"vid"`
	HomePointsBefore     *float64 `json:"hpb"`
	HomePointsAfter      *float64 `json:"hpa"`
	VisitingPointsBefore *float64 `json:"vpb"`
	VisitingPointsAfter  *float64 `json:"vpa"`
	IdkWhatThisDoes      *float64 `json:"pta"`
}

type VideoDetailsAssetContextMeasure string

var VideoDetailsAssetContextMeasures = struct {
	FGM                VideoDetailsAssetContextMeasure
	FGA                VideoDetailsAssetContextMeasure
	FG_PCT             VideoDetailsAssetContextMeasure
	FG3M               VideoDetailsAssetContextMeasure
	FG3A               VideoDetailsAssetContextMeasure
	FG3_PCT            VideoDetailsAssetContextMeasure
	FTM                VideoDetailsAssetContextMeasure
	FTA                VideoDetailsAssetContextMeasure
	OREB               VideoDetailsAssetContextMeasure
	DREB               VideoDetailsAssetContextMeasure
	AST                VideoDetailsAssetContextMeasure
	FGM_AST            VideoDetailsAssetContextMeasure
	FG3_AST            VideoDetailsAssetContextMeasure
	STL                VideoDetailsAssetContextMeasure
	BLK                VideoDetailsAssetContextMeasure
	BLKA               VideoDetailsAssetContextMeasure
	TOV                VideoDetailsAssetContextMeasure
	PF                 VideoDetailsAssetContextMeasure
	PFD                VideoDetailsAssetContextMeasure
	POSS_END_FT        VideoDetailsAssetContextMeasure
	PTS_PAINT          VideoDetailsAssetContextMeasure
	PTS_FB             VideoDetailsAssetContextMeasure
	PTS_OFF_TOV        VideoDetailsAssetContextMeasure
	PTS_2ND_CHANCE     VideoDetailsAssetContextMeasure
	REB                VideoDetailsAssetContextMeasure
	TM_FGM             VideoDetailsAssetContextMeasure
	TM_FGA             VideoDetailsAssetContextMeasure
	TM_FG3M            VideoDetailsAssetContextMeasure
	TM_FG3A            VideoDetailsAssetContextMeasure
	TM_FTM             VideoDetailsAssetContextMeasure
	TM_FTA             VideoDetailsAssetContextMeasure
	TM_OREB            VideoDetailsAssetContextMeasure
	TM_DREB            VideoDetailsAssetContextMeasure
	TM_REB             VideoDetailsAssetContextMeasure
	TM_TEAM_REB        VideoDetailsAssetContextMeasure
	TM_AST             VideoDetailsAssetContextMeasure
	TM_STL             VideoDetailsAssetContextMeasure
	TM_BLK             VideoDetailsAssetContextMeasure
	TM_BLKA            VideoDetailsAssetContextMeasure
	TM_TOV             VideoDetailsAssetContextMeasure
	TM_TEAM_TOV        VideoDetailsAssetContextMeasure
	TM_PF              VideoDetailsAssetContextMeasure
	TM_PFD             VideoDetailsAssetContextMeasure
	TM_PTS             VideoDetailsAssetContextMeasure
	TM_PTS_PAINT       VideoDetailsAssetContextMeasure
	TM_PTS_FB          VideoDetailsAssetContextMeasure
	TM_PTS_OFF_TOV     VideoDetailsAssetContextMeasure
	TM_PTS_2ND_CHANCE  VideoDetailsAssetContextMeasure
	TM_FGM_AST         VideoDetailsAssetContextMeasure
	TM_FG3_AST         VideoDetailsAssetContextMeasure
	TM_POSS_END_FT     VideoDetailsAssetContextMeasure
	OPP_FGM            VideoDetailsAssetContextMeasure
	OPP_FGA            VideoDetailsAssetContextMeasure
	OPP_FG3M           VideoDetailsAssetContextMeasure
	OPP_FG3A           VideoDetailsAssetContextMeasure
	OPP_FTM            VideoDetailsAssetContextMeasure
	OPP_FTA            VideoDetailsAssetContextMeasure
	OPP_OREB           VideoDetailsAssetContextMeasure
	OPP_DREB           VideoDetailsAssetContextMeasure
	OPP_REB            VideoDetailsAssetContextMeasure
	OPP_TEAM_REB       VideoDetailsAssetContextMeasure
	OPP_AST            VideoDetailsAssetContextMeasure
	OPP_STL            VideoDetailsAssetContextMeasure
	OPP_BLK            VideoDetailsAssetContextMeasure
	OPP_BLKA           VideoDetailsAssetContextMeasure
	OPP_TOV            VideoDetailsAssetContextMeasure
	OPP_TEAM_TOV       VideoDetailsAssetContextMeasure
	OPP_PF             VideoDetailsAssetContextMeasure
	OPP_PFD            VideoDetailsAssetContextMeasure
	OPP_PTS            VideoDetailsAssetContextMeasure
	OPP_PTS_PAINT      VideoDetailsAssetContextMeasure
	OPP_PTS_FB         VideoDetailsAssetContextMeasure
	OPP_PTS_OFF_TOV    VideoDetailsAssetContextMeasure
	OPP_PTS_2ND_CHANCE VideoDetailsAssetContextMeasure
	OPP_FGM_AST        VideoDetailsAssetContextMeasure
	OPP_FG3_AST        VideoDetailsAssetContextMeasure
	OPP_POSS_END_FT    VideoDetailsAssetContextMeasure
	PTS                VideoDetailsAssetContextMeasure
}{
	FGM:                "FGM",
	FGA:                "FGA",
	FG_PCT:             "FG_PCT",
	FG3M:               "FG3M",
	FG3A:               "FG3A",
	FG3_PCT:            "FG3_PCT",
	FTM:                "FTM",
	FTA:                "FTA",
	OREB:               "OREB",
	DREB:               "DREB",
	AST:                "AST",
	FGM_AST:            "FGM_AST",
	FG3_AST:            "FG3_AST",
	STL:                "STL",
	BLK:                "BLK",
	BLKA:               "BLKA",
	TOV:                "TOV",
	PF:                 "PF",
	PFD:                "PFD",
	POSS_END_FT:        "POSS_END_FT",
	PTS_PAINT:          "PTS_PAINT",
	PTS_FB:             "PTS_FB",
	PTS_OFF_TOV:        "PTS_OFF_TOV",
	PTS_2ND_CHANCE:     "PTS_2ND_CHANCE",
	REB:                "REB",
	TM_FGM:             "TM_FGM",
	TM_FGA:             "TM_FGA",
	TM_FG3M:            "TM_FG3M",
	TM_FG3A:            "TM_FG3A",
	TM_FTM:             "TM_FTM",
	TM_FTA:             "TM_FTA",
	TM_OREB:            "TM_OREB",
	TM_DREB:            "TM_DREB",
	TM_REB:             "TM_REB",
	TM_TEAM_REB:        "TM_TEAM_REB",
	TM_AST:             "TM_AST",
	TM_STL:             "TM_STL",
	TM_BLK:             "TM_BLK",
	TM_BLKA:            "TM_BLKA",
	TM_TOV:             "TM_TOV",
	TM_TEAM_TOV:        "TM_TEAM_TOV",
	TM_PF:              "TM_PF",
	TM_PFD:             "TM_PFD",
	TM_PTS:             "TM_PTS",
	TM_PTS_PAINT:       "TM_PTS_PAINT",
	TM_PTS_FB:          "TM_PTS_FB",
	TM_PTS_OFF_TOV:     "TM_PTS_OFF_TOV",
	TM_PTS_2ND_CHANCE:  "TM_PTS_2ND_CHANCE",
	TM_FGM_AST:         "TM_FGM_AST",
	TM_FG3_AST:         "TM_FG3_AST",
	TM_POSS_END_FT:     "TM_POSS_END_FT",
	OPP_FGM:            "OPP_FGM",
	OPP_FGA:            "OPP_FGA",
	OPP_FG3M:           "OPP_FG3M",
	OPP_FG3A:           "OPP_FG3A",
	OPP_FTM:            "OPP_FTM",
	OPP_FTA:            "OPP_FTA",
	OPP_OREB:           "OPP_OREB",
	OPP_DREB:           "OPP_DREB",
	OPP_REB:            "OPP_REB",
	OPP_TEAM_REB:       "OPP_TEAM_REB",
	OPP_AST:            "OPP_AST",
	OPP_STL:            "OPP_STL",
	OPP_BLK:            "OPP_BLK",
	OPP_BLKA:           "OPP_BLKA",
	OPP_TOV:            "OPP_TOV",
	OPP_TEAM_TOV:       "OPP_TEAM_TOV",
	OPP_PF:             "OPP_PF",
	OPP_PFD:            "OPP_PFD",
	OPP_PTS:            "OPP_PTS",
	OPP_PTS_PAINT:      "OPP_PTS_PAINT",
	OPP_PTS_FB:         "OPP_PTS_FB",
	OPP_PTS_OFF_TOV:    "OPP_PTS_OFF_TOV",
	OPP_PTS_2ND_CHANCE: "OPP_PTS_2ND_CHANCE",
	OPP_FGM_AST:        "OPP_FGM_AST",
	OPP_FG3_AST:        "OPP_FG3_AST",
	OPP_POSS_END_FT:    "OPP_POSS_END_FT",
	PTS:                "PTS",
}

func VideoDetailsAsset(gameID, playerID string, contextMeasure VideoDetailsAssetContextMeasure) ([]VideoDetailAsset, error) {
	url := fmt.Sprintf("https://stats.nba.com/stats/videodetailsasset?AheadBehind=&ClutchTime=&ContextFilter=&ContextMeasure=%s&DateFrom=&DateTo=&EndPeriod=&EndRange=&GameID=%s&GameSegment=&LastNGames=0&LeagueID=&Location=&Month=0&OpponentTeamID=0&Outcome=&Period=0&PlayerID=%s&PointDiff=&Position=&RangeType=&RookieYear=&Season=2024-25&SeasonSegment=&SeasonType=Regular+Season&StartPeriod=&StartRange=&TeamID=0&VsConference=&VsDivision=", contextMeasure, gameID, playerID)
	body, err := curl(url)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	unmarshalledBody := VideoDetailsAssetResp{}
	err = json.Unmarshal(body, &unmarshalledBody)
	if err != nil && strings.Contains(err.Error(), "invalid character '<'") {
		return nil, utils.ErrorWithTrace(fmt.Errorf("received html response, expected json when querying for %s", contextMeasure))
	} else if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	Playlist := unmarshalledBody.ResultSets.Playlist
	VideoUrls := unmarshalledBody.ResultSets.Meta.VideoUrls

	if len(Playlist) != len(VideoUrls) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("playlist array and urls array lengths do not match (╯°□°)╯︵ ɹoɹɹƎ"))
	}

	res := make([]VideoDetailAsset, 0, len(Playlist))
	for i := range Playlist {
		entry := VideoDetailAsset{
			GameID:      Playlist[i].GameID,
			EventID:     Playlist[i].EventID,
			Year:        Playlist[i].Year,
			Month:       Playlist[i].Month,
			Day:         Playlist[i].Day,
			Description: Playlist[i].Description,
			Uuid:        VideoUrls[i].Uuid,
			SmallUrl:    VideoUrls[i].SmallUrl,
			MedUrl:      VideoUrls[i].MedUrl,
			LargeUrl:    VideoUrls[i].LargeUrl,
		}
		if entry.LargeUrl == nil && entry.MedUrl == nil && entry.SmallUrl == nil {
			continue
		}
		res = append(res, entry)
	}
	return res, nil
}

type LeagueGameFinderByPlayerIDResp struct {
	ResultsSet []struct {
		Headers []string        `json:"headers"`
		RowSet  [][]interface{} `json:"rowSet"`
	} `json:"resultSets"`
}

type LeagueGameFinderGame struct {
	SeasonID         *string
	PlayerId         *float64
	PlayerName       *string
	TeamID           *float64
	TeamAbbreviation *string
	TeamName         *string
	GameID           *string
	GameDate         *string
	Matchup          *string
	WL               *string
	MIN              *float64
	PTS              *float64
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TOV              *float64
	PF               *float64
	PlusMinus        *float64
}

func (g *LeagueGameFinderGame) IsPreSeason() bool {
	return strings.HasPrefix(*g.SeasonID, "1")
}

func (g *LeagueGameFinderGame) IsRegularSeason() bool {
	return strings.HasPrefix(*g.SeasonID, "2")
}

func (g *LeagueGameFinderGame) IsAllStar() bool {
	return strings.HasPrefix(*g.SeasonID, "3")
}

func (g *LeagueGameFinderGame) IsPlayoffs() bool {
	return strings.HasPrefix(*g.SeasonID, "4")
}

func (g *LeagueGameFinderGame) IsPlayIn() bool {
	return strings.HasPrefix(*g.SeasonID, "5")
}

func (g *LeagueGameFinderGame) IsIST() bool {
	return strings.HasPrefix(*g.SeasonID, "6")
}

func LeagueGameFinderByPlayerIDAndSeason(playerID int, season string) ([]LeagueGameFinderGame, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}
	url := fmt.Sprintf("https://stats.nba.com/stats/leaguegamefinder?PlayerOrTeam=P&LeagueID=00&PlayerID=%d&Season=%s", playerID, season)
	return leagueGameFinder(url)
}

func LeagueGameFinderBySeason(season string) ([]LeagueGameFinderGame, error) {
	if utils.IsInvalidSeason(season) {
		return nil, utils.ErrorWithTrace(fmt.Errorf("invalid season provided: %s", season))
	}
	url := fmt.Sprintf("https://stats.nba.com/stats/leaguegamefinder?PlayerOrTeam=P&LeagueID=00&Season=%s", season)
	return leagueGameFinder(url)
}

func leagueGameFinder(url string) ([]LeagueGameFinderGame, error) {
	body, err := curl(url)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	unmarshalledBody := LeagueGameFinderByPlayerIDResp{}
	if err := json.Unmarshal(body, &unmarshalledBody); err != nil {
		return []LeagueGameFinderGame{}, err
	}

	expectedHeaders := []string{
		"SEASON_ID",
		"PLAYER_ID",
		"PLAYER_NAME",
		"TEAM_ID",
		"TEAM_ABBREVIATION",
		"TEAM_NAME",
		"GAME_ID",
		"GAME_DATE",
		"MATCHUP",
		"WL",
		"MIN",
		"PTS",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TOV",
		"PF",
		"PLUS_MINUS",
	}
	receivedHeaders := unmarshalledBody.ResultsSet[0].Headers
	if err := validateHeaders(expectedHeaders, receivedHeaders); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	res := make([]LeagueGameFinderGame, len(unmarshalledBody.ResultsSet[0].RowSet))
	for i, raw := range unmarshalledBody.ResultsSet[0].RowSet {
		game := LeagueGameFinderGame{
			SeasonID:         maybe[string](raw[0]),
			PlayerId:         maybe[float64](raw[1]),
			PlayerName:       maybe[string](raw[2]),
			TeamID:           maybe[float64](raw[3]),
			TeamAbbreviation: maybe[string](raw[4]),
			TeamName:         maybe[string](raw[5]),
			GameID:           maybe[string](raw[6]),
			GameDate:         maybe[string](raw[7]),
			Matchup:          maybe[string](raw[8]),
			WL:               maybe[string](raw[9]),
			MIN:              maybe[float64](raw[10]),
			PTS:              maybe[float64](raw[11]),
			FGM:              maybe[float64](raw[12]),
			FGA:              maybe[float64](raw[13]),
			FG_PCT:           maybe[float64](raw[14]),
			FG3M:             maybe[float64](raw[15]),
			FG3A:             maybe[float64](raw[16]),
			FG3_PCT:          maybe[float64](raw[17]),
			FTM:              maybe[float64](raw[18]),
			FTA:              maybe[float64](raw[19]),
			FT_PCT:           maybe[float64](raw[20]),
			OREB:             maybe[float64](raw[21]),
			DREB:             maybe[float64](raw[22]),
			REB:              maybe[float64](raw[23]),
			AST:              maybe[float64](raw[24]),
			STL:              maybe[float64](raw[25]),
			BLK:              maybe[float64](raw[26]),
			TOV:              maybe[float64](raw[27]),
			PF:               maybe[float64](raw[28]),
			PlusMinus:        maybe[float64](raw[29]),
		}
		res[i] = game
	}
	return res, nil
}

type BoxScoreTraditionalV2Resp struct {
	ResultsSet []BoxScoreTraditionalV2ResultsSet `json:"resultSets"`
}

type BoxScoreTraditionalV2ResultsSet struct {
	Name    string          `json:"name"`
	Headers []string        `json:"headers"`
	RowSet  [][]interface{} `json:"rowSet"`
}

type BoxScoreTraditionalV2Data struct {
	PlayerStats           []BoxScoreTraditionalV2PlayerStats
	TeamStats             []BoxScoreTraditionalV2TeamStats
	TeamStarterBenchStats []BoxScoreTraditionalV2TeamStarterBenchStats
}

type BoxScoreTraditionalV2PlayerStats struct {
	GameID           *string
	TeamId           *float64
	TeamAbbreviation *string
	TeamCity         *string
	PlayerId         *float64
	PlayerName       *string
	Nickname         *string
	StartPosition    *string
	Comment          *string
	MIN              *string
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TO               *float64
	PF               *float64
	PTS              *float64
	PlusMinus        *float64
}

func (p *BoxScoreTraditionalV2PlayerStats) Minutes() (int, error) {
	if p.MIN == nil {
		return 0, nil
	}
	minString := strings.Split(*p.MIN, ":")[0]
	minFloat, err := strconv.ParseFloat(minString, 64)
	if err != nil {
		return 0, utils.ErrorWithTrace(err)
	}
	return int(minFloat), nil
}

func (p *BoxScoreTraditionalV2PlayerStats) Seconds() (int, error) {
	if p.MIN == nil {
		return 0, nil
	}
	secString := strings.Split(*p.MIN, ":")[1]
	secFloat, err := strconv.ParseFloat(secString, 64)
	if err != nil {
		return 0, utils.ErrorWithTrace(err)
	}
	return int(secFloat), nil
}

func (p *BoxScoreTraditionalV2PlayerStats) DidNotPlay() (bool, error) {
	if p.MIN == nil {
		return true, nil
	}
	splitMins := strings.Split(*p.MIN, ":")
	if len(splitMins) != 2 {
		return true, utils.ErrorWithTrace(fmt.Errorf("malformed MIN string: %s", *p.MIN))
	}
	minString, secString := splitMins[0], splitMins[1]
	secFloat, err := strconv.ParseFloat(secString, 64)
	if err != nil {
		return true, utils.ErrorWithTrace(err)
	}
	minFloat, err := strconv.ParseFloat(minString, 64)
	if err != nil {
		return true, utils.ErrorWithTrace(err)
	}
	return secFloat == 0 && minFloat == 0, nil
}

type BoxScoreTraditionalV2TeamStats struct {
	GameID           *string
	TeamID           *float64
	TeamName         *string
	TeamAbbreviation *string
	TeamCity         *string
	MIN              *float64
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TO               *float64
	PF               *float64
	PTS              *float64
	PlusMinus        *float64
}

type BoxScoreTraditionalV2TeamStarterBenchStats struct {
	GameID           *string
	TeamID           *float64
	TeamName         *string
	TeamAbbreviation *string
	TeamCity         *string
	StartersBench    *string
	MIN              *float64
	FGM              *float64
	FGA              *float64
	FG_PCT           *float64
	FG3M             *float64
	FG3A             *float64
	FG3_PCT          *float64
	FTM              *float64
	FTA              *float64
	FT_PCT           *float64
	OREB             *float64
	DREB             *float64
	REB              *float64
	AST              *float64
	STL              *float64
	BLK              *float64
	TO               *float64
	PF               *float64
	PTS              *float64
}

func BoxScoreTraditionalV2(gameID string) (*BoxScoreTraditionalV2Data, error) {
	url := fmt.Sprintf("https://stats.nba.com/stats/boxscoretraditionalv2?GameID=%s", gameID)
	body, err := curl(url)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	if strings.Contains(string(body), "html") {
		return nil, utils.ErrorWithTrace(fmt.Errorf("received response body containing html"))
	}

	unmarshalledBody := BoxScoreTraditionalV2Resp{}
	if err := json.Unmarshal(body, &unmarshalledBody); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	boxScore := BoxScoreTraditionalV2Data{}

	for _, set := range unmarshalledBody.ResultsSet {
		switch set.Name {
		case "PlayerStats":
			playerStats, err := unmarshalBoxScorePlayerStats(set)
			if err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
			boxScore.PlayerStats = playerStats
		case "TeamStats":
			teamStats, err := unmarshalBoxScoreTeamStats(set)
			if err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
			boxScore.TeamStats = teamStats
		case "TeamStarterBenchStats":
			teamStarterBenchStats, err := unmarshalTeamStarterBenchStats(set)
			if err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
			boxScore.TeamStarterBenchStats = teamStarterBenchStats
		default:
			return nil, utils.ErrorWithTrace(fmt.Errorf("invalid ResultsSet found: %v", set.Name))
		}
	}
	return &boxScore, nil
}

func unmarshalBoxScorePlayerStats(set BoxScoreTraditionalV2ResultsSet) ([]BoxScoreTraditionalV2PlayerStats, error) {
	expectedHeaders := []string{
		"GAME_ID",
		"TEAM_ID",
		"TEAM_ABBREVIATION",
		"TEAM_CITY",
		"PLAYER_ID",
		"PLAYER_NAME",
		"NICKNAME",
		"START_POSITION",
		"COMMENT",
		"MIN",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TO",
		"PF",
		"PTS",
		"PLUS_MINUS",
	}
	if err := validateHeaders(expectedHeaders, set.Headers); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	playerStats := make([]BoxScoreTraditionalV2PlayerStats, len(set.RowSet))
	for i, raw := range set.RowSet {
		stats := BoxScoreTraditionalV2PlayerStats{
			GameID:           maybe[string](raw[0]),
			TeamId:           maybe[float64](raw[1]),
			TeamAbbreviation: maybe[string](raw[2]),
			TeamCity:         maybe[string](raw[3]),
			PlayerId:         maybe[float64](raw[4]),
			PlayerName:       maybe[string](raw[5]),
			Nickname:         maybe[string](raw[6]),
			StartPosition:    maybe[string](raw[7]),
			Comment:          maybe[string](raw[8]),
			MIN:              maybe[string](raw[9]),
			FGM:              maybe[float64](raw[10]),
			FGA:              maybe[float64](raw[11]),
			FG_PCT:           maybe[float64](raw[12]),
			FG3M:             maybe[float64](raw[13]),
			FG3A:             maybe[float64](raw[14]),
			FG3_PCT:          maybe[float64](raw[15]),
			FTM:              maybe[float64](raw[16]),
			FTA:              maybe[float64](raw[17]),
			FT_PCT:           maybe[float64](raw[18]),
			OREB:             maybe[float64](raw[19]),
			DREB:             maybe[float64](raw[20]),
			REB:              maybe[float64](raw[21]),
			AST:              maybe[float64](raw[22]),
			STL:              maybe[float64](raw[23]),
			BLK:              maybe[float64](raw[24]),
			TO:               maybe[float64](raw[25]),
			PF:               maybe[float64](raw[26]),
			PTS:              maybe[float64](raw[27]),
			PlusMinus:        maybe[float64](raw[28]),
		}
		playerStats[i] = stats
	}
	return playerStats, nil
}

func unmarshalBoxScoreTeamStats(set BoxScoreTraditionalV2ResultsSet) ([]BoxScoreTraditionalV2TeamStats, error) {
	expectedHeaders := []string{
		"GAME_ID",
		"TEAM_ID",
		"TEAM_NAME",
		"TEAM_ABBREVIATION",
		"TEAM_CITY",
		"MIN",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TO",
		"PF",
		"PTS",
		"PLUS_MINUS",
	}
	if err := validateHeaders(expectedHeaders, set.Headers); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	teamStats := make([]BoxScoreTraditionalV2TeamStats, len(set.RowSet))
	for i, raw := range set.RowSet {
		stats := BoxScoreTraditionalV2TeamStats{
			GameID:           maybe[string](raw[0]),
			TeamID:           maybe[float64](raw[1]),
			TeamName:         maybe[string](raw[2]),
			TeamAbbreviation: maybe[string](raw[3]),
			TeamCity:         maybe[string](raw[4]),
			MIN:              maybe[float64](raw[5]),
			FGM:              maybe[float64](raw[6]),
			FGA:              maybe[float64](raw[7]),
			FG_PCT:           maybe[float64](raw[8]),
			FG3M:             maybe[float64](raw[9]),
			FG3A:             maybe[float64](raw[10]),
			FG3_PCT:          maybe[float64](raw[11]),
			FTM:              maybe[float64](raw[12]),
			FTA:              maybe[float64](raw[13]),
			FT_PCT:           maybe[float64](raw[14]),
			OREB:             maybe[float64](raw[15]),
			DREB:             maybe[float64](raw[16]),
			REB:              maybe[float64](raw[17]),
			AST:              maybe[float64](raw[18]),
			STL:              maybe[float64](raw[19]),
			BLK:              maybe[float64](raw[20]),
			TO:               maybe[float64](raw[21]),
			PF:               maybe[float64](raw[22]),
			PTS:              maybe[float64](raw[23]),
			PlusMinus:        maybe[float64](raw[24]),
		}
		teamStats[i] = stats
	}
	return teamStats, nil
}

func unmarshalTeamStarterBenchStats(set BoxScoreTraditionalV2ResultsSet) ([]BoxScoreTraditionalV2TeamStarterBenchStats, error) {
	expectedHeaders := []string{
		"GAME_ID",
		"TEAM_ID",
		"TEAM_NAME",
		"TEAM_ABBREVIATION",
		"TEAM_CITY",
		"STARTERS_BENCH",
		"MIN",
		"FGM",
		"FGA",
		"FG_PCT",
		"FG3M",
		"FG3A",
		"FG3_PCT",
		"FTM",
		"FTA",
		"FT_PCT",
		"OREB",
		"DREB",
		"REB",
		"AST",
		"STL",
		"BLK",
		"TO",
		"PF",
		"PTS",
	}
	if err := validateHeaders(expectedHeaders, set.Headers); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	teamStarterBenchStats := make([]BoxScoreTraditionalV2TeamStarterBenchStats, len(set.RowSet))
	for i, raw := range set.RowSet {
		stats := BoxScoreTraditionalV2TeamStarterBenchStats{
			GameID:           maybe[string](raw[0]),
			TeamID:           maybe[float64](raw[1]),
			TeamName:         maybe[string](raw[2]),
			TeamAbbreviation: maybe[string](raw[3]),
			TeamCity:         maybe[string](raw[4]),
			StartersBench:    maybe[string](raw[5]),
			MIN:              maybe[float64](raw[6]),
			FGM:              maybe[float64](raw[7]),
			FGA:              maybe[float64](raw[8]),
			FG_PCT:           maybe[float64](raw[9]),
			FG3M:             maybe[float64](raw[10]),
			FG3A:             maybe[float64](raw[11]),
			FG3_PCT:          maybe[float64](raw[12]),
			FTM:              maybe[float64](raw[13]),
			FTA:              maybe[float64](raw[14]),
			FT_PCT:           maybe[float64](raw[15]),
			OREB:             maybe[float64](raw[16]),
			DREB:             maybe[float64](raw[17]),
			REB:              maybe[float64](raw[18]),
			AST:              maybe[float64](raw[19]),
			STL:              maybe[float64](raw[20]),
			BLK:              maybe[float64](raw[21]),
			TO:               maybe[float64](raw[22]),
			PF:               maybe[float64](raw[23]),
			PTS:              maybe[float64](raw[24]),
		}
		teamStarterBenchStats[i] = stats
	}
	return teamStarterBenchStats, nil
}

func validateHeaders(expected, received []string) error {
	if len(expected) != len(received) {
		return (fmt.Errorf("expected headers to be of length %d, found %d", len(expected), len(received)))
	}
	for i := range expected {
		if expected[i] != received[i] {
			return (fmt.Errorf("uh oh! mismatched headers! expected %s, found %s", expected[i], received[i]))
		}
	}
	return nil
}

func GetPlayersBySeason(season string) ([]CommonAllPlayer, error) {
	playerCacheMu.Lock()
	defer playerCacheMu.Unlock()
	if cached, exists := playerCache[season]; exists {
		return cached, nil
	}

	players, err := CommonAllPlayersBySeason(season)
	if err != nil {
		return nil, err
	}
	playerCache[season] = players
	return players, nil
}

func GetPlayersByIds(season string, playerIDs []string) ([]CommonAllPlayer, error) {
	pidMap := map[string]bool{}
	for _, pid := range playerIDs {
		pidMap[pid] = true
	}

	seasonPlayers, err := GetPlayersBySeason(season)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	players := make([]CommonAllPlayer, 0, len(playerIDs))
	for _, p := range seasonPlayers {
		if pidMap[fmt.Sprintf("%d", int(*p.PersonID))] {
			players = append(players, p)
		}
	}
	return players, nil
}

func PlayerCacheJanitor() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		playerCacheMu.Lock()
		playerCache = make(map[string][]CommonAllPlayer)
		playerCacheMu.Unlock()
	}
}

var nbaClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        25,
		MaxIdleConnsPerHost: 25,
		IdleConnTimeout:     90 * time.Second,
	},
}

var sem = make(chan int, 25)
var rateLimiter = rate.NewLimiter(rate.Limit(25), 3)

func curl(url string) ([]byte, error) {
	sem <- 1
	defer func() { <-sem }()
	if err := rateLimiter.Wait(context.Background()); err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Referer", "https://www.nba.com/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := nbaClient.Do(req)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return body, nil
}

func maybe[T any](x any) *T {
	if x, ok := x.(T); ok {
		return &x
	}
	// log.Printf("Type assertion failed: expected type %T, got type %v\n", *new(T), reflect.TypeOf(x))
	return nil
}
