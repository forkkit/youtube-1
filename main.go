// Sample Go code for user authorization

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"golang.org/x/net/context"
	"google.golang.org/api/youtube/v3"
)

const ApiParts = "snippet,localizations,status"

var filenameRegex = regexp.MustCompile(`^D([0-9]{3}).*$`)

var titleTemplate = template.Must(template.New("main").Parse(`{{ .Title }} | Great Himalaya Trail | Day {{ .Day }}`))

var descriptionTemplate = template.Must(template.New("main").Parse(`Great Himalaya Trail - Day {{ .Day }} - {{ .DateString }} in the {{ .Section }} Section.

{{ if .To }}Today we hiked from {{ .From }} ({{ .FromLocal }}) to {{ .To }} ({{ .ToLocal }}){{ end -}}
{{- if .Pass }} via {{ .Pass }} ({{ .PassLocal }}){{ end -}}
{{- if .SecondPass }} and {{ .SecondPass }} ({{ .SecondLocal }}){{ end -}}
{{- if .End }} {{ .End }}{{ end -}}
{{- if .To }}.{{ end }}

=== The Great Himalaya Trail ===

From April to September 2019 Mathi and Dave thru-hiked the {{ .TotalLocal }} Great Himalaya Trail.

The concept of the Great Himalaya Trail is to follow the highest elevation continuous hiking route across the Himalayas. The Nepal section stretches for 1,400km from Kanchenjunga in the east to Humla in the west. It winds through the mountains with an average elevation of 3,750m, and up to 6,200m. Originally conceived and mapped by Robin Boustead, the route includes parts of the more commercialised treks, linking them together with sections that are so remote even the locals seldom hike there. 

=== Get Involved ===

If you would like to do the GHT yourself, join my WhatsApp group: https://chat.whatsapp.com/D5kC4kBc7SALDE8WctMmrH

More info about my preparation for the trek: https://www.wildernessprime.com/expeditions/great-himalaya-trail/ 

Our logistics were arranged by Narayan at Mac Trek: http://www.mactreks.com/

Music in this episode by Blue Dot Sessions: https://www.sessions.blue/

=== Index ===

{{ .Index }}

`))

func titleCase(s string) string {
	return strings.Replace(strings.Title(strings.ToLower(s)), "'S", "'s", -1)
}

func isLocal() bool {
	host, _ := os.Hostname()
	return host == "Davids-MacBook.local"
}

func renderImages() {

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

	files, err := getFilesInFolder(driveService, "1SPRjcEw1nPhQbj05MejHEvWteM0pRVQD")
	if err != nil {
		return fmt.Errorf("getting files in folder: %w", err)
	}
	if len(files) != 125 {
		return fmt.Errorf("should be 125 files in folder, but found %d", len(files))
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
			day := daysByIndex[dayNumber]
			if day == nil {
				return fmt.Errorf("no day number %d for file %q", dayNumber, f.Name)
			}
			day.DriveFile = f
			day.DriveFileId = f.Id
		}
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

	updateAllStrings(daysOrdered)

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

	for _, day := range daysOrdered {
		if day.Video == nil {
			continue
		}
		if day.Video.Id == "" {

			if isLocal() {
				fmt.Printf("Skipping video insert (day %d) because of local execution\n", day.Day)
				continue
			}

			// add video
			fmt.Printf("Inserting video: %q\n", day.Video.Snippet.Title)
			call := youtubeService.Videos.Insert(ApiParts, day.Video)

			fmt.Println("Downloading video", day.DriveFileId)
			download, err := driveService.Files.Get(day.DriveFileId).Download()
			if err != nil {
				return fmt.Errorf("downloading drive file: %w", err)
			}
			defer download.Body.Close()
			_, err = call.Media(download.Body).Do()
			if err != nil {
				return fmt.Errorf("inserting video: %w", err)
			}

		} else {
			// update video
			fmt.Printf("Updating video: %q\n", day.Video.Snippet.Title)
			_, err := youtubeService.Videos.Update(ApiParts, day.Video).Do()
			if err != nil {
				return fmt.Errorf("updating video: %w", err)
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
