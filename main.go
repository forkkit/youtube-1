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
const UpdateDetails = true
const ReorderPlaylist = false
const ApiPartsInsert = "snippet,localizations,status"
const ApiPartsUpdate = "snippet,localizations"
const ApiPartsRead = "snippet,localizations,status,fileDetails"
const PlaylistItemParts = "id,contentDetails,snippet"
const Playlist = "PLiM-TFJI81R_X4HUrRDjwSJmK-MpqC1dW"

var StartTime = time.Date(2020, 2, 1, 21, 0, 0, 0, time.UTC)

var filenameRegex = regexp.MustCompile(`^([A-Z])([0-9]{3}).*$`)

const thumbnailTestingImportDir = `/Users/dave/Downloads/thumbnails`
const thumbnailTestingOutputDir = `/Users/dave/Downloads/thumbnails-compressed`

const PageOutputDir = "/Users/dave/src/wildernessprime/content/expeditions/great-himalaya-trail"

func titleCase(s string) string {
	return strings.Replace(strings.Title(strings.ToLower(s)), "'S", "'s", -1)
}

func isLocal() bool {
	host, _ := os.Hostname()
	fmt.Println(host)
	return host == "Davids-Air" || host == "Davids-MacBook-Air.local"
}

func main() {
	var err error
	if isLocal() {
		err = CreateTrailNotes()
		//err = updatePages(context.Background())
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
				f, err := transformImage(day, download.Body, false)
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

var videoIds = map[string]string{
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0M30=":     "PmTkw3Vpdj0",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0MH0=":     "zHrEqF_X2iw",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0MX0=":     "5V2lKigQ1cY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0Mn0=":     "ps7iIKmZArw",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0N30=":     "bVEtYoZk0zw",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0NX0=":     "ZdqsppgGGZ4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0Nn0=":     "BBwY2-VmJpE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0OH0=":     "CtCG3lDLRGQ",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0OX0=":     "-CbR6OtN4Io",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE0fQ==":     "gcsrJiPMhg8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE1M30=":     "gLVD1fM_c94",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE1MH0=":     "qfEvWnCPG5s",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE1Mn0=":     "rqLcXZS6Jnk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE1NH0=":     "JVyyUHrKKTA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE2fQ==":     "3FMRlDxrMs8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE3fQ==":     "39ERHaSx49g",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjE4fQ==":     "p6EsuXAbAbk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwM30=":     "PC82sgB-3A4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwMH0=":     "KbDIetDuXG4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwMX0=":     "QM-rSdmnggE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwN30=":     "WP9v5NSqyvE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwNH0=":     "v8feU2RDc-c",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwNX0=":     "wetAVNbVr7A",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwNn0=":     "DhfSBAARiYY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwOH0=":     "LfT6s7WWvOA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwOX0=":     "Ew1s5IS0o4Y",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEwfQ==":     "AjspmuCvdHg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExM30=":     "CT202zIQkRE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExMH0=":     "1yvu6bIR3w8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExMX0=":     "4NsGzDPwmg4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExMn0=":     "RmUXluPd6LY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExN30=":     "Ma-z_b1OBPI",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExNX0=":     "0xqMCN7Hx00",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExNn0=":     "t-RfkWHwDAY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjExOH0=":     "1jN3eg5InHk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyM30=":     "7VQN2RXk0Rs",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyMH0=":     "DZrMlmAe1HU",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyMX0=":     "SCOARPF6Mw4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyMn0=":     "RhbSsEqAHzQ",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyN30=":     "tuuxURdQgyo",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyNH0=":     "NHI6K03SakY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyNX0=":     "hPrVUAGBsc4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyNn0=":     "G_AolyXnJt0",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyOX0=":     "Wc5t0xTAZHk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEyfQ==":     "9ZMzGSCEG5U",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzM30=":     "4O1Uivigybw",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzMH0=":     "Ylho6eQtg2k",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzMX0=":     "Sh2v8t6Du1g",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzMn0=":     "ahWXUmYsX1Q",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzN30=":     "6Y-K_zIcjws",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzNH0=":     "yp6j2pzMhLQ",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzNn0=":     "mT772ftT5kc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzOH0=":     "MklEMxu65cc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzOX0=":     "GISnGwKfT-Y",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjEzfQ==":     "hPm_dzVlk1o",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjF9":         "M7EAxcwILRQ",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjI0fQ==":     "1RmBNQu49Hs",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjI1fQ==":     "hrtTp8KKduE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjI3fQ==":     "cqfGYhrwwCc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjI4fQ==":     "GouSwlIUj5Y",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjI5fQ==":     "TYq1Pcgg3Ao",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjIxfQ==":     "q0fgLOMZUXY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjIyfQ==":     "C8200QB91yw",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjIzfQ==":     "TuyshY0fZ94",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjJ9":         "KDEIibvNGXE",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjM0fQ==":     "Yxc-P05aA68",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjM2fQ==":     "KYMvAWZfkQ8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjM3fQ==":     "57hbe-EIWn4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjM4fQ==":     "mYFZSLiRZSA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjM5fQ==":     "_yb0PJCsFe4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjMwfQ==":     "bcWvmnM5OAM",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjMxfQ==":     "W4N9L1LIDO4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjMyfQ==":     "HEsO_xwUu4Q",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjMzfQ==":     "og0CchPzNdU",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjN9":         "hRM0UJkTOmA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQ1fQ==":     "riabbR2kpkc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQ2fQ==":     "yFvaOQoHKnU",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQ3fQ==":     "Jn5XhBHZUi4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQ4fQ==":     "j6-H5rIYdks",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQ5fQ==":     "5miRtup7ByI",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQwfQ==":     "NyFhlsCLiVo",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQxfQ==":     "AHYZnfJ_LLg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQyfQ==":     "uiwApDAKG0w",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjQzfQ==":     "5S4oU9O40Jk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjR9":         "KRifKfUb64k",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjU0fQ==":     "58aF1Nds-xA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjU1fQ==":     "4NxdtOzA118",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjU2fQ==":     "q2VTtop1Ztk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjU3fQ==":     "uhxF-5uZeNc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjU4fQ==":     "-4YD18xAnC4",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjUwfQ==":     "moo05ITrwBQ",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjUxfQ==":     "2TRh-UyaEkc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjUzfQ==":     "JH_znHQmDJ8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjV9":         "clH-Rc-hZtY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjY0fQ==":     "HUjLH9tvjvY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjY1fQ==":     "-WgR1CJzDsg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjY3fQ==":     "_j412bYPNF8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjY4fQ==":     "GFqabwd5Jw0",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjY5fQ==":     "wPGE4zxE8Xg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjYwfQ==":     "QjSQTp0189E",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjYxfQ==":     "InTnIhGbn1o",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjYyfQ==":     "b7YFO5CToos",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjYzfQ==":     "8doRylwr6cU",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc0fQ==":     "VAsscbx72ag",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc1fQ==":     "va6SuOZAuaM",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc2fQ==":     "b5DU_jJHkrY",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc3fQ==":     "-UKM9dI_mMI",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc4fQ==":     "sjUmTpijkxc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjc5fQ==":     "1lSAOfTWwH8",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjcxfQ==":     "MBkcMHw8VJc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjcyfQ==":     "D84mLB_ERto",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjczfQ==":     "mMQMxfOvt48",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjd9":         "R7qSra0aNGo",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjg4fQ==":     "vVl3Qv9kDvA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjg5fQ==":     "eOtGZB-s0UA",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjgwfQ==":     "N4AyLCkcEKU",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjgyfQ==":     "AsaEERtNiOk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjgzfQ==":     "BSuyeFUMqyc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjh9":         "LUfig_9DEd0",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjk1fQ==":     "9SMSQ-CS7us",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjk3fQ==":     "IJiSyas2FJk",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjk4fQ==":     "QeQeKRvrTBc",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjkwfQ==":     "jh3pgVKMdzg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjkxfQ==":     "A9r8K-5o-0U",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6ImRheSIsImsiOjkyfQ==":     "1ddzooRv-Cg",
	"eyJ2IjoxLCJlIjoiZ2h0IiwidCI6InRyYWlsZXIiLCJrIjoxfQ==": "POHhwrogJ8U",
}
