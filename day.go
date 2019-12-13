package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/dustin/go-humanize"
	drive "google.golang.org/api/drive/v3"
	youtube "google.golang.org/api/youtube/v3"
)

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
	return days, nil
}

func updateAllStrings(days []*GhtDay) {
	var index string
	var section string
	for _, day := range days {
		if day.Section != section {
			index = fmt.Sprintf("%s\n\n--- %s Section ---\n\n", index, day.Section)
			section = day.Section
		}
		if day.From == "" {
			index = fmt.Sprintf("%s\nDay %d - %s", index, day.Day, titleCase(day.Rest))
		} else {
			index = fmt.Sprintf("%s\nDay %d - %s", index, day.Day, titleCase(day.To))
			if day.Video != nil {
				index = fmt.Sprintf("%s https://youtu.be/%s", index, day.Video.Id)
			}
		}
	}

	for _, v := range days {
		if v.From == "" {
			continue
		}
		updateStrings(v, true, index)
		updateStrings(v, false, index)
	}
}

func updateStrings(day *GhtDay, usa bool, index string) error {

	v := struct {
		*GhtDay
		TotalLocal string
		Index      string
	}{
		GhtDay: day,
		Index:  index,
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
	} else {
		v.FromLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.FromM)))
		v.ToLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.ToM)))
		v.FromLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.FromM)))
		v.PassLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.PassM)))
		v.SecondLocal = fmt.Sprintf("%s m", humanize.Comma(int64(v.SecondPassM)))
		v.TotalLocal = "1,400 km"
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
	Section            string
	Rest               string
	DayAndDate         string
	Desc               string
	Special            bool
	DriveFileId        string
	DriveFile          *drive.File
	YoutubeVideoId     string
	Video              *youtube.Video
	DateString         string
	FullTitle          string
	FullDescription    string
	FullTitleUsa       string
	FullDescriptionUsa string
}
