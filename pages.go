package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"text/template"
	"time"
)

func updatePages(ctx context.Context) error {

	data, err := getData()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	youtubeService, err := getYoutubeService(ctx)
	if err != nil {
		return fmt.Errorf("getting youtube service: %w", err)
	}

	if err := getVideos(youtubeService, data); err != nil {
		return fmt.Errorf("getting videos: %w", err)
	}

	if err := updateAllStrings(data); err != nil {
		return fmt.Errorf("updating all strings: %w", err)
	}

	for _, item := range data {
		if !item.HasVideo {
			continue
		}
		if item.Video == nil {
			continue
		}
		if item.Expedition != "ght" || item.Type != "day" {
			continue
		}
		data := pageTemplateData{}
		data.Day = item.Key
		data.ActualDate = item.Date
		data.PublishDate = item.LiveTime
		data.SocialDate = item.LiveTime
		data.DayPadded = fmt.Sprintf("%03d", item.Key)
		data.Title = item.Title
		data.Highlights = item.Highlights
		data.Image = ImageFilenames[item.Key]
		data.YouTubeId = item.Video.Id

		buf := bytes.NewBufferString("")

		if err := pageTemplate.Execute(buf, data); err != nil {
			return fmt.Errorf("executing page template: %w", err)
		}

		if err := ioutil.WriteFile(filepath.Join(PageOutputDir, fmt.Sprintf("day-%03d.en.md", item.Key)), buf.Bytes(), 0666); err != nil {
			return fmt.Errorf("writing page template file: %w", err)
		}
	}

	return nil
}

type pageTemplateData struct {
	ActualDate, PublishDate, SocialDate time.Time
	DayPadded                           string
	Day                                 int
	Title, Highlights, Image, YouTubeId string
}

var pageTemplate = template.Must(template.New("main").Parse(`---
type: report
date: {{ .ActualDate }}
publishDate: {{ .PublishDate }}
slug: day-{{ .DayPadded }}
translationKey: day-{{ .DayPadded }}
title: Key {{ .Key }} - {{ .Title }}
description: {{ .Highlights }}
image: "/v1553075075/{{ .Image }}.jpg"
keywords: []
author: dave
featured: false
social_posts: true
social_date: {{ .SocialDate }}
hashtags: "#vlog"
title_has_context: false
---

{{ .Highlights }}

<iframe class="youtube" src="https://www.youtube.com/embed/{{ .YouTubeId }}" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

`))
