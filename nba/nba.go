package nba

import (
	"dunkod/utils"
	"encoding/json"
	"io"
	"net/http"
)

func initNBAReq(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Referer", "https://www.nba.com/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	return req
}

type CommonAllPlayersResp struct {
	ResultSets []struct {
		RowSet [][]interface{} `json:"rowSet"`
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

func CommonAllPlayers() ([]CommonAllPlayer, error) {
	url := "https://stats.nba.com/stats/commonallplayers?LeagueID=00&Season=2023-24&IsOnlyCurrentSeason=0"
	req := initNBAReq(url)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
		players[i] = player
	}
	return players, nil
}

func maybe[T any](x any) *T {
	if x, ok := x.(T); ok {
		return &x
	}
	return nil
}