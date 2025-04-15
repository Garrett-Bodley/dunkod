package config

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
)

var DatabaseFile string
var SecretFile string
var TokenFile string
var ProdFlag *bool
var BigScrape *bool

// Sorted slice of all valid seasons
//
//	ValidSeasons[0] == most recent valid season
//	ValidSeasons[len(ValidSeasons)-1] == oldest valid season
var ValidSeasons = []string{
	"2024-25",
	"2023-24",
	"2022-23",
	"2021-22",
	"2020-21",
	"2019-20",
	"2018-19",
	"2017-18",
	"2016-17",
	"2015-16",
	"2014-15",
}

var SeasonTypes = []string{
	// "Pre+Season",
	"Regular+Season",
	// "All+Star",
	"PlayIn",
	"Playoffs",
}

var TeamIDs = []int{
	1610612737, // Atlanta Hawks
	1610612738, // Boston Celtics
	1610612751, // Brooklyn Nets
	1610612766, // Charlotte Hornets
	1610612741, // Chicago Bulls
	1610612739, // Cleveland Cavaliers
	1610612742, // Dallas Mavericks
	1610612743, // Denver Nuggets
	1610612765, // Detroit Pistons
	1610612744, // Golden State Warriors
	1610612745, // Houston Rockets
	1610612754, // Indiana Pacers
	1610612746, // Los Angeles Clippers
	1610612747, // Los Angeles Lakers
	1610612763, // Memphis Grizzlies
	1610612748, // Miami Heat
	1610612749, // Milwaukee Bucks
	1610612750, // Minnesota Timberwolves
	1610612740, // New Orleans Pelicans
	1610612752, // New York Knicks
	1610612760, // Oklahoma City Thunder
	1610612753, // Orlando Magic
	1610612755, // Philadelphia 76ers
	1610612756, // Phoenix Suns
	1610612757, // Portland Trail Blazers
	1610612758, // Sacramento Kings
	1610612759, // San Antonio Spurs
	1610612761, // Toronto Raptors
	1610612762, // Utah Jazz
	1610612764, // Washington Wizards
}

func LoadConfig() error {
	ProdFlag = flag.Bool("p", false, "designates production")
	BigScrape = flag.Bool("s", false, "do big scrape task and then die")
	flag.Parse()
	binPath, err := os.Executable()
	if err != nil {
		return err
	}

	if *ProdFlag {
		DatabaseFile = "/sqlitedata/database.db"
		SecretFile = "/secrets/secret.json"
		TokenFile = "/secrets/token.json"
	} else {
		DatabaseFile = filepath.Join(filepath.Dir(binPath), "database.db")
		SecretFile = filepath.Join(filepath.Dir(binPath), "secret.json")
		TokenFile = filepath.Join(filepath.Dir(binPath), "token.json")
	}
	slices.Sort(ValidSeasons)
	slices.Reverse(ValidSeasons)
	return nil
}
