package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func getFilesInFolder(srv *drive.Service, folderId string) ([]*drive.File, error) {
	var done bool
	var page string
	var files []*drive.File

	for !done {
		query := fmt.Sprintf("'%s' in parents", folderId)
		response, err := srv.Files.List().Q(query).PageSize(50).Fields("nextPageToken, files(id, name)").PageToken(page).Do()
		if err != nil {
			return nil, fmt.Errorf("list files from drive: %w", err)
		}
		for _, file := range response.Files {
			files = append(files, file)
		}
		page = response.NextPageToken
		if page == "" {
			done = true
		}
	}
	return files, nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getDriveClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "drive_token.json"
	tok, err := driveTokenFromFile(tokFile)
	if err != nil {
		tok, err = getDriveTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("can't get drive token from web: %w", err)
		}
		err = saveDriveToken(tokFile, tok)
		if err != nil {
			return nil, fmt.Errorf("can't save drive token: %w", err)
		}
	}
	return config.Client(ctx, tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getDriveTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code %w", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web %w", err)
	}
	return tok, nil
}

// Retrieves a token from a local file.
func driveTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("can't open drive token %q from file: %w", file, err)
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("can't decode drive token from json: %w", err)
	}
	return tok, nil
}

// Saves a token to a file path.
func saveDriveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to create token file %q: %w", path, err)
	}
	defer f.Close()
	if err = json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("unable to encode token file: %w", err)
	}
	return nil
}

func getDriveService(ctx context.Context) (*drive.Service, error) {

	var fname string
	if isLocal() {
		fname = "/Users/dave/.credentials/drive_secret.json"
	} else {
		fname = "/root/.credentials/drive_secret.json"
	}

	b, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file %q: %w", fname, err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	client, err := getDriveClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to get drive client: %w", err)
	}

	srv, err := drive.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %w", err)
	}

	return srv, nil
}
