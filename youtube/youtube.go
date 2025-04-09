// Sample Go code for user authorization

package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"dunkod/config"
	"dunkod/utils"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var tokenMu = sync.Mutex{}
var secretMu = sync.Mutex{}

var service *youtube.Service
var serviceMut = sync.RWMutex{}

func GetClient(ctx context.Context, oauthConfig *oauth2.Config) (*http.Client, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tok, err := tokenFromFile(config.TokenFile)
	if err != nil {
		tok, err := getTokenFromWeb(oauthConfig)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		err = saveToken(config.TokenFile, tok)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
	} else {
		tokenSource := oauthConfig.TokenSource(context.Background(), tok)
		newTok, err := tokenSource.Token()
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		if newTok.AccessToken != tok.AccessToken {
			saveToken(config.TokenFile, newTok)
			tok = newTok
		}
	}
	return oauthConfig.Client(ctx, tok), nil
}

func GetService() (*youtube.Service, error) {
	oauthConfig, err := OAuthConfig()
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	token, err := GetToken(oauthConfig)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	ctx := context.Background()
	client := oauthConfig.Client(ctx, token)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	return service, nil
}

func GetToken(oauthConfig *oauth2.Config) (*oauth2.Token, error) {
	token, err := tokenFromFile(config.TokenFile)
	if err != nil {
		token, err = getTokenFromWeb(oauthConfig)
		if err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
		if err := saveToken(config.TokenFile, token); err != nil {
			return nil, utils.ErrorWithTrace(err)
		}
	} else {
		tokenSource := oauthConfig.TokenSource(context.Background(), token)
		newTok, err := tokenSource.Token()
		if err != nil {
			token, err2 := getTokenFromWeb(oauthConfig)
			if err2 != nil {
				return nil, errors.Join(err, err2)
			}
			if err := saveToken(config.TokenFile, token); err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
		} else if newTok.AccessToken != token.AccessToken {
			if err := saveToken(config.TokenFile, token); err != nil {
				return nil, utils.ErrorWithTrace(err)
			}
			token = newTok
		}
	}
	return token, nil
}

func getTokenFromWeb(oauthConfig *oauth2.Config) (*oauth2.Token, error) {
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, utils.ErrorWithTrace(fmt.Errorf("unable to read authorization code %v", err))
	}

	tok, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, utils.ErrorWithTrace(fmt.Errorf("unable to retrieve token from web %v", err))
	}
	return tok, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}
	defer f.Close()

	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	return t, nil
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return utils.ErrorWithTrace(fmt.Errorf("unable to cache oauth token: %v", err))
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}

func OAuthConfig() (*oauth2.Config, error) {
	secretMu.Lock()
	defer secretMu.Unlock()
	b, err := os.ReadFile(config.SecretFile)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	oauthConfig, err := google.ConfigFromJSON(b, youtube.YoutubeUploadScope)
	if err != nil {
		return nil, utils.ErrorWithTrace(err)
	}

	return oauthConfig, nil
}

func UploadFile(filepath, title, description string, tags []string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}
	defer file.Close()
	snippet := &youtube.VideoSnippet{
		Title:       title,
		Description: description,
		CategoryId:  "17", // 17 => Sports
		Tags:        tags,
	}

	status := &youtube.VideoStatus{
		PrivacyStatus:           "public",
		MadeForKids:             false,
		SelfDeclaredMadeForKids: false,
	}

	upload := &youtube.Video{
		Snippet: snippet,
		Status:  status,
	}
	serviceMut.RLock()
	defer serviceMut.RUnlock()
	call := service.Videos.Insert([]string{"snippet", "status"}, upload)
	resp, err := call.Media(file, googleapi.ChunkSize(32*1024*1024)).Do()
	if err != nil {
		return "", utils.ErrorWithTrace(err)
	}
	return fmt.Sprintf("https://www.youtube.com/embed/%s", resp.Id), nil
}

func InitService() error {
	var err error
	serviceMut.Lock()
	defer serviceMut.Unlock()
	service, err = GetService()
	if err != nil {
		return utils.ErrorWithTrace(err)
	}
	return nil
}

func ServiceJanitor() {
	var err error

	ticker := time.NewTicker(8 * time.Hour)
	for range ticker.C {
		serviceMut.Lock()
		service, err = GetService()
		if err != nil {
			panic(err)
		}
		serviceMut.Unlock()
	}
}
