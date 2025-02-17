package utils

import (
	"dunkod/config"

	"fmt"
	"runtime"
)

func ErrorWithTrace(e error) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("%s:%d\n\t%v", file, line, e)
}

func IsInvalidSeason(season string) bool {
	for _, s := range config.ValidSeasons {
		if season == s {
			return false
		}
	}
	return true
}