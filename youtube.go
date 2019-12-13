package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

func getVideos(srv *youtube.Service) (map[int]*youtube.Video, error) {

	var ids []string

	{
		var done bool
		var pageToken string

		for !done {

			// Search for all the videos in this channel and make a list of their IDs
			response, err := srv.Search.List("id").Type("video").ForMine(true).PageToken(pageToken).Do()
			if err != nil {
				return nil, fmt.Errorf("youtube search list call: %w", err)
			}

			for _, v := range response.Items {
				ids = append(ids, v.Id.VideoId)
			}

			pageToken = response.NextPageToken
			if pageToken == "" {
				done = true
			}
		}
	}

	videos := map[int]*youtube.Video{}

	{
		var done bool
		var pageToken string

		for !done {
			// Get all these videos with the Videos API
			response, err := srv.Videos.List(ApiParts).Id(strings.Join(ids, ",")).PageToken(pageToken).Do()
			if err != nil {
				return nil, fmt.Errorf("youtube videos list call: %w", err)
			}

			// Check for meta data on each video and ignore those without meta data
			for _, v := range response.Items {
				if v.Localizations == nil || v.Localizations["eo"].Title != "youtube-tool-meta-data" {
					continue
				}

				var meta Meta
				err := json.Unmarshal([]byte(v.Localizations["eo"].Description), &meta)
				if err != nil {
					return nil, fmt.Errorf("unmarshaling youtube meta data for ID %s: %w", v.Id, err)
				}

				if meta.Expedition == "ght" && meta.Type == "day" {
					videos[meta.Key] = v
				}
			}

			pageToken = response.NextPageToken
			if pageToken == "" {
				done = true
			}
		}
	}

	return videos, nil
}

func getYoutubeService(ctx context.Context) (*youtube.Service, error) {

	var filename string
	if isLocal() {
		filename = "/Users/dave/.credentials/youtube_secret.json"
	} else {
		filename = "/root/.credentials/youtube_secret.json"
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading client secret file: %w", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/youtube-go-quickstart.json
	config, err := google.ConfigFromJSON(b, youtube.YoutubeScope)
	if err != nil {
		return nil, fmt.Errorf("parsing client secret file to config: %w", err)
	}

	client, err := getYoutubeClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("getting youtube client: %w", err)
	}

	service, err := youtube.New(client)
	if err != nil {
		return nil, fmt.Errorf("creating youtube service: %w", err)
	}

	return service, nil
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getYoutubeClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	fname := "youtube_token.json"
	tok, err := youtubeTokenFromFile(fname)
	if err != nil {
		tok, err = getYoutubeTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("getting youtube token from web: %w", err)
		}
		err = saveYoutubeToken(fname, tok)
		if err != nil {
			return nil, fmt.Errorf("saving youtube token: %w", err)
		}
	}
	return config.Client(ctx, tok), nil
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getYoutubeTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("reading authorization code: %w", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("retrieve token from web: %w", err)
	}
	return tok, nil
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func youtubeTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening youtube token file: %w", err)
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	if err != nil {
		return nil, fmt.Errorf("decoding token from json: %w", err)
	}
	defer f.Close()
	return t, nil
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveYoutubeToken(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating youtube token file %q: %w", file, err)
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("encoding youtube token: %w", err)
	}
	return nil
}
