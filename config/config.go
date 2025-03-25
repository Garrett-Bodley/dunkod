package config

import (
	"os"
	"path/filepath"
	"fmt"

	flag "github.com/spf13/pflag"
)

var DatabaseFile string
var SecretFile string
var TokenFile string
var ProdFlag *bool

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
	ProdFlag = flag.BoolP("prod", "p", false, "designates production")
	flag.Parse()
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Println(*ProdFlag)
	if *ProdFlag {
		DatabaseFile = "/sqlitedata/database.db"
		SecretFile = "/secrets/secret.json"
		TokenFile = "/secrets/token.json"
	}else{
		DatabaseFile = filepath.Join(filepath.Dir(binPath), "database.db")
		SecretFile = filepath.Join(filepath.Dir(binPath), "secret.json")
		TokenFile = filepath.Join(filepath.Dir(binPath), "token.json")
	}
	return nil
}