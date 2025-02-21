package utils

import (
	"dunkod/config"

	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
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

var sem = make(chan int, 50)

func CurlToFile(url, filepath string) error {
	sem <- 1
	defer func() { <-sem }()
	client := &http.Client{
		Timeout: time.Minute,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ErrorWithTrace(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return ErrorWithTrace(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return ErrorWithTrace(err)
	}
	return nil
}

func Curl(req *http.Request) ([]byte, error) {
	sem <- 1
	defer func() { <-sem }()
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrorWithTrace(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrorWithTrace(err)
	}
	return body, nil
}
