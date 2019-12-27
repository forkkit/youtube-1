package main

import (
	"bytes"
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

var titleTemplate = template.Must(template.New("main").Parse(`{{ .Title }} Great Himalaya Trail Day {{ .Day }}`))

var descriptionTemplate = template.Must(template.New("main").Parse(`{{ "" -}}
Great Himalaya Trail - Day {{ .Day }} - {{ .DateString }} in the {{ .Section }} section.

{{ if .To }}Today {{ .Self }} {{ .Transport }} from {{ .From }} ({{ .FromLocal }}) to {{ .To }} ({{ .ToLocal }}){{ end -}}
{{- if .Pass }} via {{ .Pass }} ({{ .PassLocal }}){{ end -}}
{{- if .SecondPass }} and {{ .SecondPass }} ({{ .SecondLocal }}){{ end -}}
{{- if .End }} {{ .End }}{{ end -}}
{{- if .To }}.{{ end }}

=== The Great Himalaya Trail ===

From April to September 2019 Mathi and Dave thru-hiked the Great Himalaya Trail.

The concept of the Great Himalaya Trail is to follow the highest elevation continuous hiking route across the Himalayas. The Nepal section stretches for {{ .TotalLocal }} from Kanchenjunga in the east to Humla in the west. It winds through the mountains with an average elevation of {{ .AvgLocal }}, and up to {{ .MaxLocal }}, with an average elevation change of {{ .ChangeLocal }} per day. The route includes parts of the more commercialised treks, linking them together with sections that are so remote even the locals seldom hike there. 

=== Get Involved ===

If you're thinking about hiking the GHT yourself, join our WhatsApp group: https://chat.whatsapp.com/D5kC4kBc7SALDE8WctMmrH

More info about our preparation for the trek: https://www.wildernessprime.com/expeditions/great-himalaya-trail/ 

Our logistics were arranged by Narayan at Mac Trek: http://www.mactreks.com/

Music in this episode by Blue Dot Sessions: https://www.sessions.blue/

{{- .Index }}

`))

func getDays() ([]*GhtDay, error) {
	var days []*GhtDay
	raw, err := ioutil.ReadFile("./ght_data.json")
	if err != nil {
		return nil, fmt.Errorf("unable to read ght data: %w", err)
	}
	err = json.Unmarshal(raw, &days)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ght data json: %w", err)
	}
	var i int
	for _, day := range days {
		if day.From == "" {
			continue
		}
		day.Position = i
		day.LiveTime = StartTime.Add(time.Duration(i*24) * time.Hour)
		i++
	}
	return days, nil
}

func getIndex(current int, days []*GhtDay, usa bool) string {

	type sectionData struct {
		name           string
		dayFrom, dayTo int
		firstVideoId   string
	}

	var sectionsOrdered []*sectionData
	sections := map[string]*sectionData{}
	var currentSection string

	{
		var section string
		for i, day := range days {
			if day.Day == current {
				currentSection = day.Section
			}
			if day.Section != section {
				section = day.Section
				s := &sectionData{
					name:    day.Section,
					dayFrom: day.Day,
				}
				if day.Video != nil {
					s.firstVideoId = day.Video.Id
				}
				sections[day.Section] = s
				sectionsOrdered = append(sectionsOrdered, s)
				if i > 0 {
					previous := days[i-1]
					sections[previous.Section].dayTo = previous.Day
				}
			}
			if i == len(days)-1 {
				sections[day.Section].dayTo = day.Day
			}
		}
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n\n=== %s Section ===\n", currentSection))

	for _, day := range days {
		if day.Section != currentSection {
			continue
		}
		sb.WriteString(fmt.Sprintf("\nDay %d - ", day.Day))
		if day.From == "" {
			switch day.Rest {
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
			if day.Pass != "" {
				pass := day.Pass
				passM := day.PassM
				passFt := day.PassFt
				if day.Day == 117 {
					// special case for Mesokanto La
					pass = day.SecondPass
					passM = day.SecondPassM
					passFt = day.SecondPassFt
				}

				if usa {
					sb.WriteString(fmt.Sprintf("%s via %s %s ft", titleCase(day.To), titleCase(pass), humanize.Comma(int64(passFt))))
				} else {
					sb.WriteString(fmt.Sprintf("%s via %s %s m", titleCase(day.To), titleCase(pass), humanize.Comma(int64(passM))))
				}
			} else {
				if day.To != "" {
					sb.WriteString(titleCase(day.To))
				} else {
					sb.WriteString(titleCase(day.From))
				}
			}
			if day.End != "" {
				sb.WriteString(fmt.Sprintf(" %s", day.End))
			}
			if day.Video != nil {
				sb.WriteString(fmt.Sprintf(" - https://youtu.be/%s", day.Video.Id))
			}
			if day.Day == current {
				sb.WriteString(" - THIS EPISODE")
			}
		}
	}

	sb.WriteString("\n\n=== Sections ===\n")

	for _, section := range sectionsOrdered {
		sb.WriteString(fmt.Sprintf("\nDay %d to %d - %s Section", section.dayFrom, section.dayTo, section.name))
		if section.firstVideoId != "" {
			sb.WriteString(fmt.Sprintf(" - https://youtu.be/%s", section.firstVideoId))
		}
		if section.name == currentSection {
			sb.WriteString(" - THIS SECTION")
		}
	}
	return sb.String()
}

func updateAllStrings(days []*GhtDay) error {
	for _, v := range days {
		if v.From == "" {
			continue
		}
		if err := updateStrings(v, true, getIndex(v.Day, days, true)); err != nil {
			return fmt.Errorf("updating strings: %w", err)
		}
		if err := updateStrings(v, false, getIndex(v.Day, days, false)); err != nil {
			return fmt.Errorf("updating strings: %w", err)
		}
	}
	return nil
}

func updateStrings(day *GhtDay, usa bool, index string) error {

	v := struct {
		*GhtDay
		TotalLocal  string
		MaxLocal    string
		AvgLocal    string
		Index       string
		ChangeLocal string
		Self        string
		Transport   string
	}{
		GhtDay: day,
		Index:  index,
	}

	if day.Day < 31 {
		v.Self = "I"
	} else {
		v.Self = "we"
	}

	if day.Day == 30 {
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

	if err := descriptionTemplate.Execute(buf, v); err != nil {
		return fmt.Errorf("executing description template: %w", err)
	}

	if usa {
		v.FullDescriptionUsa = buf.String()
	} else {
		v.FullDescription = buf.String()
	}

	return nil
}

type GhtDay struct {
	Day                int
	Date               time.Time
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
}
