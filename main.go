// Sample Go code for user authorization

package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/youtube/v3"
)

const InsertVideos = false
const UpdateOne = 0
const UpdateThumbnails = false
const UpdateDetails = true

const ApiParts = "snippet,localizations,status"

var filenameRegex = regexp.MustCompile(`^D([0-9]{3}).*$`)

func titleCase(s string) string {
	return strings.Replace(strings.Title(strings.ToLower(s)), "'S", "'s", -1)
}

func isLocal() bool {
	host, _ := os.Hostname()
	return host == "Davids-MacBook.local"
}

func main() {
	err := saveVideos(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func saveVideos(ctx context.Context) error {

	daysOrdered, err := getDays()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	daysByIndex := map[int]*GhtDay{}
	for _, day := range daysOrdered {
		daysByIndex[day.Day] = day
	}

	driveService, err := getDriveService(ctx)
	if err != nil {
		return fmt.Errorf("can't get drive service: %w", err)
	}

	f := func(folder string, expected int, action func(day *GhtDay, file *drive.File)) error {
		files, err := getFilesInFolder(driveService, folder)
		if err != nil {
			return fmt.Errorf("getting files in folder: %w", err)
		}
		if len(files) != expected {
			return fmt.Errorf("should be %d files in folder, but found %d", expected, len(files))
		}

		for _, f := range files {
			matches := filenameRegex.FindStringSubmatch(f.Name)
			if len(matches) != 2 {
				return fmt.Errorf("found file with unknown filename %q", f.Name)
			} else {
				dayNumber, err := strconv.Atoi(matches[1])
				if err != nil {
					return fmt.Errorf("parsing day number from %q: %w", f.Name, err)
				}
				if dayNumber == 0 {
					// special case for trailer
					continue
				}
				day := daysByIndex[dayNumber]
				if day == nil {
					return fmt.Errorf("no day number %d for file %q", dayNumber, f.Name)
				}
				action(day, f)
			}
		}
		return nil
	}
	if err := f("1SPRjcEw1nPhQbj05MejHEvWteM0pRVQD", 125, func(day *GhtDay, file *drive.File) { day.File = file }); err != nil {
		return fmt.Errorf("getting video files from drive: %w", err)
	}
	if err := f("1xETuf-n2mRH0REoZp-eLXLn5bzRTe3pi", 126, func(day *GhtDay, file *drive.File) { day.Thumbnail = file }); err != nil {
		return fmt.Errorf("getting thumbnail files from drive: %w", err)
	}

	youtubeService, err := getYoutubeService(ctx)
	if err != nil {
		return fmt.Errorf("getting youtube service: %w", err)
	}

	videos, err := getVideos(youtubeService)
	if err != nil {
		return fmt.Errorf("getting videos: %w", err)
	}

	for _, day := range daysOrdered {
		day.Video = videos[day.Day]
	}

	if err := updateAllStrings(daysOrdered); err != nil {
		return fmt.Errorf("updating all strings: %w", err)
	}

	for _, day := range daysOrdered {
		if day.From == "" {
			continue
		}

		if day.Video == nil {
			// create new video
			day.Video = &youtube.Video{}
		}

		// add data
		if day.Video.Snippet == nil {
			day.Video.Snippet = &youtube.VideoSnippet{}
		}
		day.Video.Snippet.CategoryId = "19"
		day.Video.Snippet.ChannelId = "UCFDggPICIlCHp3iOWMYt8cg"
		day.Video.Snippet.DefaultAudioLanguage = "en"
		day.Video.Snippet.DefaultLanguage = "en"
		day.Video.Snippet.LiveBroadcastContent = "none"
		day.Video.Snippet.Description = day.FullDescription
		day.Video.Snippet.Title = day.FullTitle

		if day.Video.Localizations == nil {
			day.Video.Localizations = map[string]youtube.VideoLocalization{}
		}

		metaData := Meta{
			Version:    1,
			Expedition: "ght",
			Type:       "day",
			Key:        day.Day,
		}
		metaDataBytes, err := json.Marshal(metaData)
		if err != nil {
			return fmt.Errorf("encoding youtube meta data json: %w", err)
		}
		day.Video.Localizations["eo"] = youtube.VideoLocalization{
			Title:       "youtube-tool-meta-data",
			Description: string(metaDataBytes),
		}

		day.Video.Localizations["en_US"] = youtube.VideoLocalization{
			Title:       day.FullTitleUsa,
			Description: day.FullDescriptionUsa,
		}

		if day.Video.Status == nil {
			day.Video.Status = &youtube.VideoStatus{}
			day.Video.Status.PrivacyStatus = "private"
		}

	}

	if isLocal() {
		fmt.Println("Length:", len(daysByIndex[31].FullDescriptionUsa))
		fmt.Println(daysByIndex[31].FullDescriptionUsa)
		fmt.Println("Skipping youtube operations because of local execution")
		return nil
	}

	for _, day := range daysOrdered {
		if day.Video == nil {
			continue
		}
		if day.Video.Id == "" {

			// insert video

			if InsertVideos {
				fmt.Printf("Inserting video: %q\n", day.Video.Snippet.Title)
				call := youtubeService.Videos.Insert(ApiParts, day.Video)

				fmt.Println("Downloading video", day.File.Id)
				download, err := driveService.Files.Get(day.File.Id).Download()
				if err != nil {
					return fmt.Errorf("downloading drive file: %w", err)
				}
				if _, err := call.Media(download.Body).Do(); err != nil {
					download.Body.Close()
					return fmt.Errorf("inserting video: %w", err)
				}
				download.Body.Close()
			}

		} else {
			// update video

			if UpdateOne > 0 && UpdateOne != day.Day {
				continue
			}

			if UpdateDetails {
				fmt.Printf("Updating video: %q\n", day.Video.Snippet.Title)
				_, err := youtubeService.Videos.Update(ApiParts, day.Video).Do()
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
				//b, err := ioutil.ReadAll(f)
				//if err != nil {
				//	return fmt.Errorf("reading transformed thumbnail: %w", err)
				//}
				//fmt.Println(len(b))
				//return nil
				if _, err := youtubeService.Thumbnails.Set(day.Video.Id).Media(f).Do(); err != nil {
					return fmt.Errorf("setting thumbnail: %w", err)
				}
			}
		}
	}
	return nil
}

type Meta struct {
	Version    int
	Expedition string
	Type       string
	Key        int
}

const missingClientSecretsMessage = `
Please configure OAuth 2.0
`

var suffixes = []string{"th", "st", "nd", "rd", "th", "th", "th", "th", "th", "th",
	"th", "th", "th", "th", "th", "th", "th", "th", "th", "th",
	"th", "st", "nd", "rd", "th", "th", "th", "th", "th", "th",
	"th", "st"}
