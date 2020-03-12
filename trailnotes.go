package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"text/template"

	"github.com/dustin/go-humanize"
)

var trailNotesTemplate = template.Must(template.New("main").Funcs(functions).Parse(`---
type: report
date: 2020-02-28 00:00:00 +0000 UTC
publishDate: 2020-02-28 00:00:00 +0000 UTC
slug: trail-notes{{ if not .Maps }}-no-maps{{ end }}
translationKey: trail-notes{{ if not .Maps }}-no-maps{{ end }}
title: Trail notes{{ if not .Maps }} (no maps){{ end }}
description: Comprehensive trail notes for the Great Himalaya Trail{{ if not .Maps }} (no maps){{ end }}.
image: "/v1553075075/{{ if .Maps }}compass-390054_1920_hz27dl.jpg{{ else }}compass-1753659_1920_h82a3n.jpg{{ end }}"
keywords: [trail-notes]
author: dave
featured: false
social_posts: false
social_date: 2020-02-28 00:00:00 +0000 UTC
hashtags: "#trail-notes"
title_has_context: false
---

<style>
.print-only {
	display: none;
}
@media print {
	#header, #nav, #breadcrumbs, header.major, #author, #next-section, #prev-section, #footer, #copyright, #navPanel {
		display:none!important;
	}
	body, #wrapper, #main, section.post, div.content {
		margin:0!important;
		padding:0!important;
		width:100%!important;
	}
	* {
		font-size: 100%!important;
		line-height: 130%!important;
	}
	p, td {
		font-size: 1rem!important;
	}
	h2 {
		font-size: 1.5rem!important;
	}
	h4 {
		font-size: 1rem!important;
	}
	table td {
		padding: 0.25rem 0.25rem
	}
	h1, h2 {
		margin: 2rem 0 1rem 0;
	}
	h3, h4 {
    	margin: 0 0 1rem 0;
    }
	p {
		margin: 0 0 1rem 0;
	}
	.page-break {
		page-break-after: always;
	}
	.no-page-break {
		page-break-inside: avoid;
	}
	.print-only {
		display: initial;
	}
	.no-print {
		display: none;
	}
	img[src*="#elev028"] {
	   max-width: 80%;
	}
}
</style>

<div class="no-print">

GPS routes for these trail notes are available here: 

* **[GPX ROUTES](https://www.dropbox.com/s/2gnn6isfuq63syq/routes-v3.gpx?dl=1)** (recommended for most apps)  

I would recommend using the GPX file above if you can, but some apps need KML format:

* [KML](https://www.dropbox.com/s/ndw8kplfp6yrlui/routes-v3.kml?dl=1) (if gpx files don't work)  
* [KML](https://www.dropbox.com/s/7uzulf1chxtat7j/routes-for-maps-me-v3.kml?dl=1) (for the maps.me app - route descriptions aren't visible in maps.me, so I've included a waypoint with the description at the start of each leg)  

You can find a printable PDF version [with maps](https://www.dropbox.com/s/elstjct83yw8hxf/trail-notes-v3.pdf?dl=1) or [with no maps](https://www.dropbox.com/s/sn5iwgh20641dgi/trail-notes-no-maps-v3.pdf?dl=1). 

There are versions of this page [with maps](/expeditions/great-himalaya-trail/trail-notes/) or [with no maps](/expeditions/great-himalaya-trail/trail-notes-no-maps/).

# Trail notes

</div>

{{ range .Legs }}

<div class="no-page-break">

## Leg {{ .Leg }}: {{ .From }} to {{ .To }}

{{ .Notes }}

</div>

{{ if gt (len .Waypoints) 0 }}

<div class="no-page-break">

#### Waypoints <span class="print-only">(leg {{ .Leg }})</span>

{{ range $i, $w := .Waypoints }}

<div class="no-page-break">

**L{{ printf "%.03d" $w.Leg }} {{ $w.Name }} ({{ comma (round $w.Elevation) }} m / {{ comma (round (feet $w.Elevation)) }} ft)**: {{ $w.Notes }}

</div>

{{ if eq $i 0 }}

</div>

{{ end }}

{{ end }}

{{ end }}

<div class="no-page-break">

#### Ratings <span class="print-only">(leg {{ .Leg }})</span>

Trail: {{ .TrailString }}  
Route: {{ .RouteString }}  
Accommodation: {{ .LodgeString }} - {{ .QualityString }}  

</div>

<div class="no-page-break">

#### Stats <span class="print-only">(leg {{ .Leg }})</span>

|   |   |  |
| - | - |- |
| Length | {{ printf "%.1f" .Length }} km | {{ printf "%.1f" (miles .Length) }} miles |
| Climb / descent | {{ comma (round .Climb) }} / {{ comma (round .Descent) }} m | {{ comma (round (feet .Climb)) }} / {{ comma (round (feet .Descent)) }} ft |
<!--| Start / end |  {{ comma (round .Start) }} / {{ comma (round .End) }} m |  {{ comma (round (feet .Start)) }} / {{ comma (round (feet .End)) }} ft |
| Top / bottom |  {{ comma (round .Top) }} / {{ comma (round .Bottom) }} m  |  {{ comma (round (feet .Top)) }} / {{ comma (round (feet .Bottom)) }} ft |-->

</div>

<div class="no-page-break">

#### Elevation <span class="print-only">(leg {{ .Leg }})</span>

![](https://storage.googleapis.com/wilderness-prime-static/elev3/E{{ printf "%.03d" .Leg }}.png#elev{{ printf "%.03d" .Leg }})

</div>

{{ if $.Maps }}

<div class="no-page-break">

#### Map <span class="print-only">(leg {{ .Leg }})</span>

![](https://storage.googleapis.com/wilderness-prime-static/maps3/L{{ printf "%.03d" .Leg }}.jpg)

</div>

<div class="page-break"></div>

{{ if or (or (eq .Leg 87) (eq .Leg 62)) (eq .Leg 22) }}

<div class="print-only">

This page is intentionally left blank.

<div class="page-break"></div>

</div>

{{ end }}

{{ end }}

{{ end }}
`))

var functions = template.FuncMap{
	"comma": func(i interface{}) string {
		switch j := i.(type) {
		case float64:
			return humanize.Comma(int64(math.Round(j)))
		case int:
			return humanize.Comma(int64(j))
		default:
			return "0"
		}
	},
	"miles": func(i ...interface{}) float64 {
		if len(i) == 0 {
			return 0
		}
		switch j := i[0].(type) {
		case float64:
			return j / 1.60934
		case int:
			return float64(j) / 1.60934
		default:
			return 0
		}
	},
	"feet": func(i ...interface{}) float64 {
		if len(i) == 0 {
			return 0
		}
		switch j := i[0].(type) {
		case float64:
			return j * 3.28084
		case int:
			return float64(j) * 3.28084
		default:
			return 0
		}
	},
	"round": func(in ...interface{}) float64 {
		if len(in) == 0 {
			return 0
		}
		var i float64
		switch j := in[0].(type) {
		case float64:
			i = j
		case int:
			i = float64(j)
		}
		if i >= 10000 {
			// round to the nearest 100
			return math.Round(i/100.0) * 100.0
		} else {
			return math.Round(i/10.0) * 10.0
		}
	},
}

func CreateTrailNotes() error {
	b, err := ioutil.ReadFile(`/Users/dave/src/youtube/trailnotes.json`)
	if err != nil {
		return err
	}
	var notes TrailNotesSheetStruct
	if err := json.Unmarshal(b, &notes); err != nil {
		return err
	}
	legs := notes.Legs
	for i, leg := range legs {
		if i == 0 {
			leg.From = "Taplejung"
		} else {
			leg.From = legs[i-1].To
		}
		for _, waypoint := range notes.Waypoints {
			if waypoint.Leg == leg.Leg {
				leg.Waypoints = append(leg.Waypoints, waypoint)
			}
		}
		for _, pass := range notes.Passes {
			if pass.Leg == leg.Leg {
				leg.Passes = append(leg.Passes, pass)
			}
		}
		if leg.Vlog != nil {
			days := strings.Split(fmt.Sprint(leg.Vlog), ",")
			for _, day := range days {
				d, err := strconv.Atoi(day)
				if err != nil {
					return err
				}
				leg.Days = append(leg.Days, d)
			}
		}

		qualityString := func(i int, t string) string {
			switch i {
			case 1:
				switch t {
				case "T", "R":
					// trail or route
					return "1/5 (major problems)"
				case "C", "S":
					// campsite, shelter
					return "1/5 (awful)"
				case "G", "H":
					// guesthouse or homestay
					return "1/5 (basic)"
				default:
					return "1/5"
				}
			case 2:
				return "2/5 (below average)"
			case 3:
				return "3/5 (average)"
			case 4:
				return "4/5 (above average)"
			case 5:
				return "5/5 (excellent)"
			default:
				return "(unknown)"
			}
		}

		leg.TrailString = qualityString(leg.Trail, "T")
		leg.RouteString = qualityString(leg.Route, "R")
		leg.QualityString = qualityString(leg.Quality, leg.Lodge)

		switch leg.Lodge {
		case "C":
			leg.LodgeString = "campsite"
		case "S":
			leg.LodgeString = "shelter"
		case "H":
			leg.LodgeString = "homestay"
		case "G":
			leg.LodgeString = "guesthouse"
		default:
			leg.LodgeString = "unknown"
		}
	}
	var out bytes.Buffer

	data := struct {
		Maps bool
		Legs []*LegStruct
	}{
		Maps: true,
		Legs: legs,
	}

	if err := trailNotesTemplate.Execute(&out, data); err != nil {
		return err
	}
	if err := ioutil.WriteFile("/Users/dave/src/wildernessprime/content/expeditions/great-himalaya-trail/trail-notes.en.md", out.Bytes(), 0777); err != nil {
		return err
	}

	out = bytes.Buffer{}
	data.Maps = false
	if err := trailNotesTemplate.Execute(&out, data); err != nil {
		return err
	}
	if err := ioutil.WriteFile("/Users/dave/src/wildernessprime/content/expeditions/great-himalaya-trail/trail-notes-no-maps.en.md", out.Bytes(), 0777); err != nil {
		return err
	}

	return nil
}

type TrailNotesSheetStruct struct {
	Legs      []*LegStruct
	Waypoints []*WaypointStruct
	Passes    []*PassStruct
}

type LegStruct struct {
	Leg  int
	Vlog interface{}

	To                                              string
	Length, Climb, Descent, Start, End, Top, Bottom float64
	Route, Trail, Quality                           int
	Lodge                                           string
	Notes                                           string

	From      string
	Waypoints []*WaypointStruct
	Passes    []*PassStruct
	Days      []int

	RouteString, TrailString, LodgeString, QualityString string
}

type WaypointStruct struct {
	Leg         int
	Name, Notes string
	Elevation   int
}

type PassStruct struct {
	Leg    int
	Pass   string
	Height float64
}
