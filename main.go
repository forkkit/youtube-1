// Sample Go code for user authorization

package main

import (
	"fmt"
	_ "image/jpeg"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/youtube/v3"
)

const InsertVideos = false
const SingleType = ""
const SingleKey = 0
const UpdateThumbnails = false
const UpdateDetails = false
const ReorderPlaylist = true
const ApiPartsInsert = "snippet,localizations,status"
const ApiPartsUpdate = "snippet,localizations"
const ApiPartsRead = "snippet,localizations,status,fileDetails"
const PlaylistItemParts = "id,contentDetails,snippet"
const Playlist = "PLiM-TFJI81R_X4HUrRDjwSJmK-MpqC1dW"

var StartTime = time.Date(2020, 2, 1, 21, 0, 0, 0, time.UTC)

var filenameRegex = regexp.MustCompile(`^([A-Z])([0-9]{3}).*$`)

const thumbnailTestingImportDir = `/Users/dave/Downloads/thumbnails`
const thumbnailTestingOutputDir = `/Users/dave/Downloads/thumbnails-out`

const PageOutputDir = "/Users/dave/src/wildernessprime/content/expeditions/great-himalaya-trail"

func titleCase(s string) string {
	return strings.Replace(strings.Title(strings.ToLower(s)), "'S", "'s", -1)
}

func isLocal() bool {
	host, _ := os.Hostname()
	return host == "Davids-MacBook-Air.local"
}

func main() {
	var err error
	if isLocal() {
		err = updatePages(context.Background())
		//err = previewThumbnails(context.Background())
	} else {
		err = saveVideos(context.Background())
	}
	if err != nil {
		log.Fatal(err)
	}
}

func saveVideos(ctx context.Context) error {

	data, err := getData()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	driveService, err := getDriveService(ctx)
	if err != nil {
		return fmt.Errorf("can't get drive service: %w", err)
	}

	f := func(folder string, expected int, expedition string, action func(*VideoData, *drive.File)) error {
		files, err := getFilesInFolder(driveService, folder)
		if err != nil {
			return fmt.Errorf("getting files in folder: %w", err)
		}
		if len(files) != expected {
			return fmt.Errorf("should be %d files in folder, but found %d", expected, len(files))
		}

		for _, f := range files {
			matches := filenameRegex.FindStringSubmatch(f.Name)
			if len(matches) != 3 {
				return fmt.Errorf("found file with unknown filename %q", f.Name)
			} else {
				var itemType string
				fileType := matches[1]
				switch fileType {
				case "D":
					itemType = "day"
				case "T":
					itemType = "trailer"
				}
				keyNumber, err := strconv.Atoi(matches[2])
				if err != nil {
					return fmt.Errorf("parsing key number from %q: %w", f.Name, err)
				}
				var item *VideoData
				for _, itm := range data {
					if itm.Expedition == expedition && itm.Type == itemType && itm.Key == keyNumber {
						item = itm
						break
					}
				}
				if item == nil {
					return fmt.Errorf("no item for type %s and key %d for file %q", itemType, keyNumber, f.Name)
				}
				action(item, f)
			}
		}
		return nil
	}
	// Video files:
	if err := f("1SPRjcEw1nPhQbj05MejHEvWteM0pRVQD", 126, "ght", func(item *VideoData, file *drive.File) { item.File = file }); err != nil {
		return fmt.Errorf("getting video files from drive: %w", err)
	}
	// Thumbnails:
	if err := f("1xETuf-n2mRH0REoZp-eLXLn5bzRTe3pi", 126, "ght", func(item *VideoData, file *drive.File) { item.Thumbnail = file }); err != nil {
		return fmt.Errorf("getting thumbnail files from drive: %w", err)
	}

	youtubeService, err := getYoutubeService(ctx)
	if err != nil {
		return fmt.Errorf("getting youtube service: %w", err)
	}

	playlistItems, err := getPlaylist(youtubeService)
	if err != nil {
		return fmt.Errorf("getting playlist items: %w", err)
	}

	if err := getVideos(youtubeService, data); err != nil {
		return fmt.Errorf("getting videos: %w", err)
	}

	if err := updateAllStrings(data); err != nil {
		return fmt.Errorf("updating all strings: %w", err)
	}

	dataByVideoId := map[string]*VideoData{}
	for _, item := range data {
		if item.Video != nil {
			dataByVideoId[item.Video.Id] = item
		}
	}

	if ReorderPlaylist {
		for _, playlistItem := range playlistItems {
			item := dataByVideoId[playlistItem.ContentDetails.VideoId]
			item.PlaylistItem = playlistItem
			playlistItem.Snippet.Position = int64(item.Position)
		}
		for _, item := range data {
			if item.PlaylistItem == nil {
				continue
			}
			fmt.Printf("Updating playlist item for day %d\n", item.Key)
			_, err := youtubeService.PlaylistItems.Update(PlaylistItemParts, item.PlaylistItem).Do()
			if err != nil {
				return fmt.Errorf("updating playlist item: %w", err)
			}
		}
	}

	for _, item := range data {
		if !item.HasVideo {
			continue
		}

		if item.Video == nil {
			// create new video
			item.Video = &youtube.Video{}
		}

		// set the correct PublishAt date, but only for new videos (don't modify this on update)
		if item.Video.Id == "" && item.Video.Status == nil {
			// changing this after setting it breaks the "premiere" feature?
			// TODO: TEST THIS
			item.Video.Status = &youtube.VideoStatus{}
			if item.Type == "day" {
				item.Video.Status.PrivacyStatus = "private"
				item.Video.Status.PublishAt = strings.TrimSuffix(item.LiveTime.Format(time.RFC3339), "Z") + ".0Z"
			} else {
				item.Video.Status.PrivacyStatus = "private"
			}
		}

		// add basic data
		if item.Video.Snippet == nil {
			item.Video.Snippet = &youtube.VideoSnippet{}
		}
		item.Video.Snippet.CategoryId = "19"
		item.Video.Snippet.ChannelId = "UCFDggPICIlCHp3iOWMYt8cg"
		item.Video.Snippet.DefaultAudioLanguage = "en"
		item.Video.Snippet.DefaultLanguage = "en"
		item.Video.Snippet.LiveBroadcastContent = "none"
		item.Video.Snippet.Description = item.FullDescription
		item.Video.Snippet.Title = item.FullTitle

		// add the special USA localized title and description
		if item.Video.Localizations == nil {
			item.Video.Localizations = map[string]youtube.VideoLocalization{}
		}
		item.Video.Localizations["en_US"] = youtube.VideoLocalization{
			Title:       item.FullTitleUsa,
			Description: item.FullDescriptionUsa,
		}

	}

	for _, day := range data {
		if day.Video == nil {
			continue
		}
		if day.Video.Id == "" {

			if SingleKey > 0 && SingleKey != day.Key {
				continue
			}
			if SingleType != "" && SingleType != day.Type {
				continue
			}

			// insert video

			if InsertVideos {
				fmt.Printf("Inserting video: %q\n", day.Video.Snippet.Title)
				call := youtubeService.Videos.Insert(ApiPartsInsert, day.Video)

				fmt.Println("Downloading video", day.File.Id)
				download, err := driveService.Files.Get(day.File.Id).Download()
				if err != nil {
					return fmt.Errorf("downloading drive file: %w", err)
				}
				insertCall := call.Media(download.Body)

				filename, err := day.GetFilename()
				if err != nil {
					return fmt.Errorf("generating meta data filename: %w", err)
				}
				insertCall.Header().Add("Slug", filename)

				if _, err := insertCall.Do(); err != nil {
					download.Body.Close()
					return fmt.Errorf("inserting video: %w", err)
				}
				download.Body.Close()
			}

		} else {
			// update video

			if SingleKey > 0 && SingleKey != day.Key {
				continue
			}
			if SingleType != "" && SingleType != day.Type {
				continue
			}

			// clear FileDetails because it's not updatable
			day.Video.FileDetails = nil

			// clear Status because we dont want to update it
			day.Video.Status = nil

			if UpdateDetails {
				fmt.Printf("Updating video: %q\n", day.Video.Snippet.Title)
				_, err := youtubeService.Videos.Update(ApiPartsUpdate, day.Video).Do()
				if err != nil {
					return fmt.Errorf("updating video: %w", err)
				}
			}

			if UpdateThumbnails {
				fmt.Println("Downloading thumbnail", day.Thumbnail.Id)
				download, err := driveService.Files.Get(day.Thumbnail.Id).Download()
				if err != nil {
					return fmt.Errorf("downloading drive file: %w", err)
				}
				f, err := transformImage(day, download.Body)
				if err != nil {
					download.Body.Close()
					return fmt.Errorf("transforming thumbnail: %w", err)
				}
				download.Body.Close()
				if _, err := youtubeService.Thumbnails.Set(day.Video.Id).Media(f).Do(); err != nil {
					return fmt.Errorf("setting thumbnail: %w", err)
				}
			}
		}
	}
	return nil
}

type Meta struct {
	Version    int    `json:"v"`
	Expedition string `json:"e"`
	Type       string `json:"t"`
	Key        int    `json:"k"`
}

const missingClientSecretsMessage = `
Please configure OAuth 2.0
`

var suffixes = []string{"th", "st", "nd", "rd", "th", "th", "th", "th", "th", "th",
	"th", "th", "th", "th", "th", "th", "th", "th", "th", "th",
	"th", "st", "nd", "rd", "th", "th", "th", "th", "th", "th",
	"th", "st"}
