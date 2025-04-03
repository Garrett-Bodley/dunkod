package config

import (
	"flag"
	"os"
	"path/filepath"
)

var DatabaseFile string
var SecretFile string
var TokenFile string
var ProdFlag *bool
var BigScrape *bool

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
	return nil
}
