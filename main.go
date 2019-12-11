// Sample Go code for user authorization

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/api/youtube/v3"
)

const ApiLimit = 10
const ApiParts = "snippet,localizations,status"

var filenameRegex = regexp.MustCompile(`^D([0-9]{3}).*$`)

func main() {

	ctx := context.Background()

	driveService := getDriveService()

	driveFiles := map[int]string{}

	filesInFolder, err := driveService.Files.List().Q("'1SPRjcEw1nPhQbj05MejHEvWteM0pRVQD' in parents").PageSize(200).Fields("nextPageToken, files(id, name)").Do()
	handleError(err, "Unable to retrieve files")

	fmt.Println("Files:")
	if len(filesInFolder.Files) == 0 {
		log.Fatalf("No files found")
	} else {
		for _, f := range filesInFolder.Files {
			matches := filenameRegex.FindStringSubmatch(f.Name)
			if len(matches) != 2 {
				log.Fatalf("File with unknown filename: %v", f.Name)
			} else {
				day, err := strconv.Atoi(matches[1])
				handleError(err, "Can't parse day number from "+f.Name)
				fmt.Printf("Day: %d %s\n", day, f.Id)
				driveFiles[day] = f.Id
			}
		}
	}

	ghtDataRaw, err := ioutil.ReadFile("./ght_data.json")
	handleError(err, "Unable to read ght data")
	var ghtData GhtData
	err = json.Unmarshal(ghtDataRaw, &ghtData)
	handleError(err, "Unable to parse ght data json")

	for _, v := range ghtData {
		if v.From == "" {
			continue
		}
		if driveFiles[v.Day] != "" {
			v.DriveFileId = driveFiles[v.Day]
		} else {
			log.Fatalf("Cant't find drive file for day %d", v.Day)
		}
		fmt.Printf("Day %d - %s %s\n", v.Day, v.DriveFileId, v.Title)
	}

	var videos []string

	youtubeService := getYoutubeService(ctx)

	var done bool
	var pageToken string

	for !done {

		// Search for all the videos in this channel and make a list of their IDs
		channelResponse, err := youtubeService.Search.List("id").Type("video").ForMine(true).PageToken(pageToken).Do()
		handleError(err, "")

		for _, v := range channelResponse.Items {
			videos = append(videos, v.Id.VideoId)
		}

		pageToken = channelResponse.NextPageToken
		if pageToken == "" {
			done = true
		}
	}

	fmt.Println("Videos:", len(videos))

	ghtVideos := map[int]*youtube.Video{}

	done, pageToken = false, ""

	for !done {
		// Get all these videos with the Videos API
		videosResponse, err := youtubeService.Videos.List(ApiParts).Id(strings.Join(videos, ",")).PageToken(pageToken).Do()
		handleError(err, "")

		// Check for meta data on each video and ignore those without meta data
		for _, v := range videosResponse.Items {
			if v.Localizations == nil || v.Localizations["eo"].Title != "youtube-tool-meta-data" {
				fmt.Println(v.Id, "skipped")
				continue
			}
			fmt.Println(v.Id, "meta data:", v.Localizations["eo"].Title, v.Localizations["eo"].Description)
			var meta Meta

			err := json.Unmarshal([]byte(v.Localizations["eo"].Description), &meta)
			handleError(err, fmt.Sprintf("Unable to unmarshal meta data for ID %s", v.Id))

			if meta.Expedition == "ght" && meta.Type == "day" {
				ghtVideos[meta.Key] = v
			}
		}

		pageToken = videosResponse.NextPageToken
		if pageToken == "" {
			done = true
		}
	}

	type videosToUpdateItem struct {
		*youtube.Video
		Data *GhtDay
	}
	var videosToUpdate []videosToUpdateItem

	for k, dataItem := range ghtData {

		if k >= ApiLimit {
			break
		}

		var v *youtube.Video

		if ghtVideos[dataItem.Day] != nil {
			// use existing video
			v = ghtVideos[dataItem.Day]
		} else {
			// create new video
			v = &youtube.Video{}
		}

		// add data
		if v.Snippet == nil {
			v.Snippet = &youtube.VideoSnippet{}
		}
		v.Snippet.CategoryId = "19"
		v.Snippet.ChannelId = "UCFDggPICIlCHp3iOWMYt8cg"
		v.Snippet.DefaultAudioLanguage = "en"
		v.Snippet.DefaultLanguage = "en"
		v.Snippet.Description = "UPDATED"
		v.Snippet.LiveBroadcastContent = "none"
		v.Snippet.Title = dataItem.Title

		if v.Localizations == nil {
			v.Localizations = map[string]youtube.VideoLocalization{}
		}
		metaData := Meta{
			Version:    1,
			Expedition: "ght",
			Type:       "day",
			Key:        dataItem.Day,
		}
		b, err := json.Marshal(metaData)
		handleError(err, "")
		v.Localizations["eo"] = youtube.VideoLocalization{
			Title:       "youtube-tool-meta-data",
			Description: string(b),
		}
		if v.Status == nil {
			v.Status = &youtube.VideoStatus{}
			v.Status.PrivacyStatus = "private"
		}

		videosToUpdate = append(videosToUpdate, videosToUpdateItem{
			Video: v,
			Data:  dataItem,
		})
	}

	for _, v := range videosToUpdate {
		if v.Id == "" {
			// add video
			fmt.Printf("Inserting video: %q\n", v.Snippet.Title)
			call := youtubeService.Videos.Insert(ApiParts, v.Video)

			fmt.Println("downloading video", v.Data.DriveFileId)
			download, err := driveService.Files.Get(v.Data.DriveFileId).Download()
			handleError(err, "Unable to download video")

			//file, err := os.Open("/Users/dave/Downloads/74354ACC-E7B0-4D9E-9B24-E293FEE166A7.MP4")
			defer download.Body.Close()
			handleError(err, "Unable to open video file")
			_, err = call.Media(download.Body).Do()
			handleError(err, "Unable to insert video")

		} else {
			// update video
			fmt.Printf("Updating video: %q\n", v.Snippet.Title)
			_, err := youtubeService.Videos.Update(ApiParts, v.Video).Do()
			handleError(err, "Unable to update video")
		}
	}

	//v1 := vr.Items[0]
	//fmt.Printf("%#v\n", v1.Snippet)
	//v1.Snippet.Title = "Footpath demo"
	//fmt.Printf("Localizations: %#v\n", v1.Localizations)
	//if v1.Localizations == nil {
	//	v1.Localizations = map[string]youtube.VideoLocalization{}
	//}

	//v1.Snippet.DefaultLanguage = "en"
	//v1.Localizations["eo"] = youtube.VideoLocalization{Title: "foo", Description: "bar"}

	//v2, err := service.Videos.Update("snippet,localizations", v1).Do()
	//handleError(err, "")
	//fmt.Printf("After update: %#v\n", v2.Snippet)
	//fmt.Printf("After update loc: %#v\n", v2.Localizations)

}

type GhtData []*GhtDay

type GhtDay struct {
	Day          int
	Date         time.Time
	From         string
	FromM        int
	FromFt       int
	To           string
	ToM          int
	ToFt         int
	Pass         string
	PassM        int
	PassFt       int
	SecondPass   string
	SecondPassM  int
	SecondPassFt int
	End          string
	Title        string
	DayAndDate   string
	Desc         string
	Special      bool
	DriveFileId  string
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

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}
