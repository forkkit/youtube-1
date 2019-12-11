package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

func getYoutubeService(ctx context.Context) *youtube.Service {
	//b, err := ioutil.ReadFile("/Users/dave/.credentials/client_secret_1059285243547-0gltuppaggdlain4qulbelnl2as4ip35.apps.googleusercontent.com.json")
	b, err := ioutil.ReadFile("/root/.credentials/youtube_secret.json")
	handleError(err, "Unable to read client secret file")

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/youtube-go-quickstart.json
	config, err := google.ConfigFromJSON(b, youtube.YoutubeScope)
	handleError(err, "Unable to parse client secret file to config")

	client := getYoutubeClient(ctx, config)
	service, err := youtube.New(client)
	handleError(err, "Error creating YouTube client")

	return service
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getYoutubeClient(ctx context.Context, config *oauth2.Config) *http.Client {
	fname := "youtube_token.json"
	tok, err := youtubeTokenFromFile(fname)
	if err != nil {
		tok = getYoutubeTokenFromWeb(config)
		saveYoutubeToken(fname, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getYoutubeTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func youtubeTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveYoutubeToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
