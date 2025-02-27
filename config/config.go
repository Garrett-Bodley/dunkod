package config

import (
	"os"
	"path/filepath"
)

var DatabaseFile string
var SecretFile string
var TokenFile string

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
	"Regular+Season",
	// "Pre+Season",
	"Playoffs",
	// "All+Star",
}

func LoadConfig() error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	DatabaseFile = filepath.Join(filepath.Dir(binPath), "database.db")
	SecretFile = filepath.Join(filepath.Dir(binPath), "secret.json")
	TokenFile = filepath.Join(filepath.Dir(binPath), "token.json")
	return nil
}
