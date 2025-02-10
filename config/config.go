package config

import (
	"os"
	"path/filepath"
)

var DatabaseFile string

func LoadConfig() error {
	binPath, err := os.Executable()
	if err != nil {
		return err
	}
	DatabaseFile = filepath.Join(filepath.Dir(binPath), "database.db")
	return nil
}