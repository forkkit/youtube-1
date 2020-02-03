package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"google.golang.org/api/youtube/v3"
)

func updatePages(ctx context.Context) error {

	data, err := getData()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	for _, item := range data {
		if id := videoIds[item.MustGetFilename()]; id != "" {
			item.Video = &youtube.Video{Id: id}
		}
	}

	/*
		youtubeService, err := getYoutubeService(ctx)
		if err != nil {
			return fmt.Errorf("getting youtube service: %w", err)
		}

		if err := getVideos(youtubeService, data); err != nil {
			return fmt.Errorf("getting videos: %w", err)
		}
	*/

	/*
		code := jen.Id("videos").Op(":=").Map(jen.String()).String().Values(jen.DictFunc(func(d jen.Dict) {
			for _, item := range data {
				if item.HasVideo {
					s, err := item.GetFilename()
					if err != nil {
						panic(err)
					}
					d[jen.Lit(s)] = jen.Lit(item.Video.Id)
				}
			}
		}))
		fmt.Printf("%#v\n", code)
		return nil
	*/

	if err := updateAllStrings(data); err != nil {
		return fmt.Errorf("updating all strings: %w", err)
	}

	var count int
	var summary summaryTemplateData
	summaryCount := 1
	for i, item := range data {
		if item.Expedition != "ght" || item.Type != "day" {
			continue
		}
		if !item.HasVideo || item.Video == nil {
			// non video day -> add placeholder to summary.Days
			dayData := pageTemplateData{}
			dayData.HasVideo = false
			dayData.Day = item.Key
			dayData.ActualDate = item.Date
			dayData.DayPadded = fmt.Sprintf("%03d", item.Key)
			dayData.Item = item
			dayData.NoVideoDescription = item.ZeroDayDescription()
			summary.Days = append(summary.Days, dayData)
			continue
		}

		dayData := pageTemplateData{}
		dayData.HasVideo = true
		dayData.Day = item.Key
		dayData.ActualDate = item.Date
		dayData.PublishDate = item.LiveTime
		dayData.SocialDate = item.LiveTime
		dayData.DayPadded = fmt.Sprintf("%03d", item.Key)
		dayData.Title = strings.TrimSuffix(item.Title, ".")
		dayData.Highlights = item.Highlights
		dayData.Image = imageFilenamesNoText[item.Key]
		dayData.YouTubeId = item.Video.Id
		dayData.Item = item
		summary.Days = append(summary.Days, dayData)
		if summary.Image == "" {
			summary.Image = imageFilenamesNoText[dayData.Day]
		}

		buf := bytes.NewBufferString("")

		if err := pageTemplate.Execute(buf, dayData); err != nil {
			return fmt.Errorf("executing page template: %w", err)
		}

		if err := ioutil.WriteFile(filepath.Join(PageOutputDir, fmt.Sprintf("day-%03d.en.md", item.Key)), buf.Bytes(), 0666); err != nil {
			return fmt.Errorf("writing page template file: %w", err)
		}

		count++

		if count%7 == 0 || i == len(data)-1 {
			//...
			summary.ActualDate = item.Date.Add(time.Minute * 30)
			summary.PublishDate = item.LiveTime.Add(time.Minute * 30)
			summary.SocialDate = item.LiveTime.Add(time.Minute * 30)
			summary.Week = summaryCount
			summary.WeekPadded = fmt.Sprintf("%02d", summaryCount)
			summary.DayStart = summary.Days[0].Day
			summary.DayEnd = summary.Days[len(summary.Days)-1].Day

			buf = bytes.NewBufferString("")

			if err := pageWeekTemplate.Execute(buf, summary); err != nil {
				return fmt.Errorf("executing page template: %w", err)
			}

			if err := ioutil.WriteFile(filepath.Join(PageOutputDir, fmt.Sprintf("week-%02d.en.md", summaryCount)), buf.Bytes(), 0666); err != nil {
				return fmt.Errorf("writing page template file: %w", err)
			}

			summaryCount++
			summary = summaryTemplateData{
				Week:       summaryCount,
				WeekPadded: fmt.Sprintf("%02d", summaryCount),
			}
		}
	}

	return nil
}

type pageTemplateData struct {
	ActualDate, PublishDate, SocialDate time.Time
	DayPadded                           string
	Day                                 int
	Title, Highlights, Image, YouTubeId string
	Item                                *VideoData
	HasVideo                            bool
	NoVideoDescription                  string
}

var pageTemplate = template.Must(template.New("main").Parse(`---
type: report
date: {{ .ActualDate }}
publishDate: {{ .PublishDate }}
slug: day-{{ .DayPadded }}
translationKey: day-{{ .DayPadded }}
title: Day {{ .Day }} - {{ .Title }}
description: {{ .Highlights }}
image: "/v1553075075/{{ .Image }}.jpg"
keywords: []
author: dave
featured: true
social_posts: true
social_date: {{ .SocialDate }}
hashtags: "#vlog"
title_has_context: false
---

{{ .Highlights }}

<iframe class="youtube75" src="https://www.youtube.com/embed/{{ .YouTubeId }}" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

`))

type summaryTemplateData struct {
	ActualDate, PublishDate, SocialDate time.Time
	WeekPadded                          string
	Week                                int
	Image                               string
	DayStart, DayEnd                    int
	Days                                []pageTemplateData
}

var pageWeekTemplate = template.Must(template.New("main").Parse(`---
type: report
date: {{ .ActualDate }}
publishDate: {{ .PublishDate }}
slug: week-{{ .WeekPadded }}
translationKey: week-{{ .WeekPadded }}
title: "Weekly summary #{{ .Week }}"
description: A summary of the vlog episodes from week {{ .Week }}
image: "/v1553075075/{{ .Image }}.jpg"
keywords: []
author: dave
featured: false
social_posts: true
social_date: {{ .SocialDate }}
hashtags: "#vlog"
title_has_context: false
---

This is a weekly summary of the trek from day {{ .DayStart }} to {{ .DayEnd }}.

{{ range .Days }}
## Day {{ .Day }}

{{ if .HasVideo }}
{{ .Highlights }}

<iframe class="youtube75" src="https://www.youtube.com/embed/{{ .YouTubeId }}" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
{{ else }}

{{ .NoVideoDescription }}

{{ end }}
{{ end }}
`))
