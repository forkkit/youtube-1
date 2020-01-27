package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
	drive "google.golang.org/api/drive/v3"
	youtube "google.golang.org/api/youtube/v3"
)

var titleTemplate = template.Must(template.New("main").Parse(`{{ .Title }} Great Himalaya Trail Key {{ .Key }}`))

var highlightsTemplate = template.Must(template.New("main").Parse(`{{ if .To }}Today {{ .Self }} {{ .Transport }} from {{ .From }} ({{ .FromLocal }}) to {{ .To }} ({{ .ToLocal }}){{ end -}}
{{- if .Pass }} via {{ .Pass }} ({{ .PassLocal }}){{ end -}}
{{- if .SecondPass }} and {{ .SecondPass }} ({{ .SecondLocal }}){{ end -}}
{{- if .End }} {{ .End }}{{ end -}}
{{- if .To }}.{{ end }}`))

var ghtDayDescriptionTemplate = template.Must(template.New("main").Parse(`{{ "" -}}
Key {{ .Key }} of the Great Himalaya Trail - {{ .DateString }} in the {{ .Section }} section. {{ .Highlights }}

🔽 The Great Himalaya Trail

Hi, I'm Dave Brophy. From April to September 2019 Mathi and I thru-hiked the Great Himalaya Trail across Nepal.

The concept of the Great Himalaya Trail is to follow the highest elevation continuous hiking route across the Himalayas. The Nepal section stretches for {{ .TotalLocal }} from Kanchenjunga in the east to Humla in the west. It winds through the mountains with an average elevation of {{ .AvgLocal }}, and up to {{ .MaxLocal }}, with an average elevation change of {{ .ChangeLocal }} per day. The route includes parts of the more commercialised treks, linking them together with sections that are so remote even the locals seldom hike there. 

🔽 Get Involved

If you're thinking about hiking the GHT yourself, join our WhatsApp group: https://chat.whatsapp.com/D5kC4kBc7SALDE8WctMmrH

More info about our preparation for the trek: https://www.wildernessprime.com/expeditions/great-himalaya-trail/ 

Our logistics were arranged by Narayan at Mac Trek: http://www.mactreks.com/

Music in this episode by Blue Dot Sessions: https://www.sessions.blue/

{{- .Index }}

`))

var ghtTrailerDescriptionTemplate = template.Must(template.New("main").Parse(`{{ "" -}}
Hi, I'm Dave Brophy. From April to September 2019 Mathi and I thru-hiked the Great Himalaya Trail across Nepal. This vlog follows our progress, with 125 episodes - one for each day of our hike.

The concept of the Great Himalaya Trail is to follow the highest elevation continuous hiking route across the Himalayas. The Nepal section stretches for {{ .TotalLocal }} from Kanchenjunga in the east to Humla in the west. It winds through the mountains with an average elevation of {{ .AvgLocal }}, and up to {{ .MaxLocal }}, with an average elevation change of {{ .ChangeLocal }} per day. The route includes parts of the more commercialised treks, linking them together with sections that are so remote even the locals seldom hike there. 

🔽 Get Involved

If you're thinking about hiking the GHT yourself, join our WhatsApp group: https://chat.whatsapp.com/D5kC4kBc7SALDE8WctMmrH

More info about our preparation for the trek: https://www.wildernessprime.com/expeditions/great-himalaya-trail/ 

Our logistics were arranged by Narayan at Mac Trek: http://www.mactreks.com/

Music in this episode by Blue Dot Sessions: https://www.sessions.blue/

{{- .Index }}

`))

func getData() ([]*VideoData, error) {
	var data []*VideoData
	raw, err := ioutil.ReadFile("./ght_data.json")
	if err != nil {
		return nil, fmt.Errorf("unable to read ght data: %w", err)
	}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ght data json: %w", err)
	}
	var i int
	for _, item := range data {
		if !item.HasVideo {
			continue
		}
		item.Position = i
		item.LiveTime = StartTime.Add(time.Duration(i*24) * time.Hour)
		i++
	}
	return data, nil
}

func getIndexGht(pointer int, data []*VideoData, usa bool, typ string) string {

	type sectionData struct {
		name         string
		min, max     int
		firstVideoId string
	}

	var sectionsOrdered []*sectionData
	sections := map[string]*sectionData{}
	var currentSection string

	var sb strings.Builder

	{
		var sectionName string
		for _, item := range data {
			if item.Section == "" {
				continue
			}
			if item.Key == pointer {
				currentSection = item.Section
			}
			if item.Section != sectionName {
				sectionName = item.Section
				s := &sectionData{
					name: item.Section,
				}
				if item.Video != nil {
					s.firstVideoId = item.Video.Id
				}
				sections[item.Section] = s
				sectionsOrdered = append(sectionsOrdered, s)
			}

			section := sections[item.Section]
			if item.Key < section.min || section.min == 0 {
				section.min = item.Key
			}
			if item.Key > section.max || section.max == 0 {
				section.max = item.Key
			}
		}
	}

	if typ == "day" {

		sb.WriteString(fmt.Sprintf("\n\n🔽 %s Section\n", currentSection))

		for _, item := range data {
			if item.Section != currentSection {
				continue
			}
			sb.WriteString(fmt.Sprintf("\nKey %d - ", item.Key))
			if item.From == "" {
				switch item.Rest {
				case "ADMIN":
					sb.WriteString("Admin day")
				case "ALT":
					sb.WriteString("Acclimatisation day")
				case "REST":
					sb.WriteString("Rest day")
				case "SICK":
					sb.WriteString("Sick day")
				case "WEATHER":
					sb.WriteString("Waiting for the weather")
				}
			} else {
				if item.Pass != "" {
					pass := item.Pass
					passM := item.PassM
					passFt := item.PassFt
					if item.Key == 117 {
						// special case for Mesokanto La
						pass = item.SecondPass
						passM = item.SecondPassM
						passFt = item.SecondPassFt
					}

					if usa {
						sb.WriteString(fmt.Sprintf("%s via %s %s ft", titleCase(item.To), titleCase(pass), humanize.Comma(int64(passFt))))
					} else {
						sb.WriteString(fmt.Sprintf("%s via %s %s m", titleCase(item.To), titleCase(pass), humanize.Comma(int64(passM))))
					}
				} else {
					if item.To != "" {
						sb.WriteString(titleCase(item.To))
					} else {
						sb.WriteString(titleCase(item.From))
					}
				}
				if item.End != "" {
					sb.WriteString(fmt.Sprintf(" %s", item.End))
				}
				if item.Video != nil {
					sb.WriteString(fmt.Sprintf(" - https://youtu.be/%s", item.Video.Id))
				}
				if item.Key == pointer {
					sb.WriteString("  ⬅️ THIS EPISODE")
				}
			}
		}
	}

	sb.WriteString("\n\n🔽 Sections\n")

	for _, section := range sectionsOrdered {
		sb.WriteString(fmt.Sprintf("\nDay %d to %d - %s Section", section.min, section.max, section.name))
		if section.firstVideoId != "" {
			sb.WriteString(fmt.Sprintf(" - https://youtu.be/%s", section.firstVideoId))
		}
		if typ == "day" && section.name == currentSection {
			sb.WriteString("  ⬅️ THIS SECTION")
		}
	}
	return sb.String()
}

func updateAllStrings(data []*VideoData) error {
	for _, item := range data {
		if !item.HasVideo {
			continue
		}
		if item.Expedition == "ght" && item.Type == "day" {
			if err := updateStringsGhtDay(item, true, getIndexGht(item.Key, data, true, "day")); err != nil {
				return fmt.Errorf("updating strings: %w", err)
			}
			if err := updateStringsGhtDay(item, false, getIndexGht(item.Key, data, false, "day")); err != nil {
				return fmt.Errorf("updating strings: %w", err)
			}
		}
		if item.Expedition == "ght" && item.Type == "trailer" {
			if err := updateStringsGhtTrailer(item, true, getIndexGht(0, data, true, "trailer")); err != nil {
				return fmt.Errorf("updating strings: %w", err)
			}
			if err := updateStringsGhtTrailer(item, false, getIndexGht(0, data, false, "trailer")); err != nil {
				return fmt.Errorf("updating strings: %w", err)
			}
		}
	}
	return nil
}

func updateStringsGhtTrailer(item *VideoData, usa bool, index string) error {
	v := struct {
		*VideoData
		TotalLocal  string
		MaxLocal    string
		AvgLocal    string
		Index       string
		ChangeLocal string
	}{
		VideoData: item,
		Index:     index,
	}

	if usa {
		v.TotalLocal = "900 miles"
		v.MaxLocal = "20,300 ft"
		v.AvgLocal = "12,300 ft"
		v.ChangeLocal = "5,200 ft"
	} else {
		v.TotalLocal = "1,400 km"
		v.MaxLocal = "6,200 m"
		v.AvgLocal = "3,750 m"
		v.ChangeLocal = "1,600 m"
	}

	if usa {
		v.FullTitleUsa = "The Great Himalaya Trail"
	} else {
		v.FullTitle = "The Great Himalaya Trail"
	}

	buf := bytes.NewBufferString("")

	if err := ghtTrailerDescriptionTemplate.Execute(buf, v); err != nil {
		return fmt.Errorf("executing description template: %w", err)
	}

	if usa {
		v.FullDescriptionUsa = buf.String()
	} else {
		v.FullDescription = buf.String()
	}

	return nil
}

func updateStringsGhtDay(item *VideoData, usa bool, index string) error {

	v := struct {
		*VideoData
		TotalLocal  string
		MaxLocal    string
		AvgLocal    string
		Index       string
		ChangeLocal string
		Self        string
		Transport   string
	}{
		VideoData: item,
		Index:     index,
	}

	if item.Key < 31 {
		v.Self = "I"
	} else {
		v.Self = "we"
	}

	if item.Key == 30 {
		v.Transport = "flew"
	} else {
		v.Transport = "hiked"
	}

	v.From = titleCase(v.From)
	v.To = titleCase(v.To)
	v.Pass = titleCase(v.Pass)
	v.SecondPass = titleCase(v.SecondPass)
	v.DateString = fmt.Sprintf("%d%s %s", v.Date.Day(), suffixes[v.Date.Day()], v.Date.Format("January"))

	if usa {
		v.FromLocal = fmt.Sprintf("%s ft", humanize.Comma(int64(v.FromFt)))
		v.ToLocal = fmt.Sprintf("%s ft", humanize.Comma(int64(v.ToFt)))
		v.FromLocal = fmt.Sprintf("%s ft", humanize.Comma(int64(v.FromFt)))
		v.PassLocal = fmt.Sprintf("%s ft", humanize.Comma(int64(v.PassFt)))
		v.SecondLocal = fmt.Sprintf("%s ft", humanize.Comma(int64(v.SecondPassFt)))
		v.TotalLocal = "900 miles"
		v.MaxLocal = "20,300 ft"
		v.AvgLocal = "12,300 ft"
		v.ChangeLocal = "5,200 ft"
	} else {
		v.FromLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.FromM)))
		v.ToLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.ToM)))
		v.FromLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.FromM)))
		v.PassLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.PassM)))
		v.SecondLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.SecondPassM)))
		v.TotalLocal = "1,400 km"
		v.MaxLocal = "6,200 m"
		v.AvgLocal = "3,750 m"
		v.ChangeLocal = "1,600 m"
	}

	buf := bytes.NewBufferString("")

	if err := titleTemplate.Execute(buf, v); err != nil {
		return fmt.Errorf("executing title template: %w", err)
	}

	if usa {
		v.FullTitleUsa = buf.String()
	} else {
		v.FullTitle = buf.String()
	}

	buf = bytes.NewBufferString("")

	if err := highlightsTemplate.Execute(buf, v); err != nil {
		return fmt.Errorf("executing description template: %w", err)
	}

	item.Highlights = buf.String()

	buf = bytes.NewBufferString("")

	if err := ghtDayDescriptionTemplate.Execute(buf, v); err != nil {
		return fmt.Errorf("executing description template: %w", err)
	}

	if usa {
		v.FullDescriptionUsa = buf.String()
	} else {
		v.FullDescription = buf.String()
	}

	return nil
}

type VideoData struct {
	Expedition         string
	Type               string
	Key                int
	Date               time.Time
	HasVideo           bool
	From               string
	FromM              int
	FromFt             int
	FromLocal          string
	To                 string
	ToM                int
	ToFt               int
	ToLocal            string
	Pass               string
	PassM              int
	PassFt             int
	PassLocal          string
	SecondPass         string
	SecondPassM        int
	SecondPassFt       int
	SecondLocal        string
	End                string
	Title              string
	Short              string
	Section            string
	Rest               string
	DayAndDate         string
	Desc               string
	Special            bool
	File               *drive.File
	Thumbnail          *drive.File
	ThumbnailTesting   os.FileInfo
	Video              *youtube.Video
	DateString         string
	FullTitle          string
	FullDescription    string
	FullTitleUsa       string
	FullDescriptionUsa string
	Position           int
	LiveTime           time.Time
	PlaylistItem       *youtube.PlaylistItem
	Highlights         string
}

func (item VideoData) GetFilename() (string, error) {
	metaData := Meta{
		Version:    1,
		Expedition: item.Expedition,
		Type:       item.Type,
		Key:        item.Key,
	}
	metaDataBytes, err := json.Marshal(metaData)
	if err != nil {
		return "", fmt.Errorf("encoding youtube meta data json: %w", err)
	}
	return base64.StdEncoding.EncodeToString(metaDataBytes), nil
}
