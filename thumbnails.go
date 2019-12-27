package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/edwvee/exiffix"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/net/context"
)

const fontSize = 75

func transformImage(day *GhtDay, file io.Reader) (io.Reader, error) {
	imgIn, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading image: %w", err)
	}
	imgBuffer := bytes.NewReader(imgIn)

	img, _, err := exiffix.Decode(imgBuffer)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	// 1280x720

	rgba := imaging.Fill(img, 1280, 720, imaging.Center, imaging.Lanczos)

	bold, err := getFont("./JosefinSans-Bold.ttf")
	if err != nil {
		return nil, err
	}
	regular, err := getFont("./JosefinSans-Regular.ttf")
	if err != nil {
		return nil, err
	}

	fg := image.White
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(bold)
	c.SetFontSize(fontSize)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)
	c.SetHinting(font.HintingNone) // font.HintingFull

	// Draw background
	draw.Draw(
		rgba,
		image.Rectangle{
			Min: image.Point{
				X: 280,
				Y: 90,
			},
			Max: image.Point{
				X: rgba.Bounds().Max.X,
				Y: 225,
			},
		},
		image.NewUniform(color.NRGBA{0, 0, 0, 128}),
		image.Point{},
		draw.Over,
	)
	// Draw the text.
	_, err = c.DrawString("The Great Himalaya Trail", freetype.Pt(320, 180))
	if err != nil {
		return nil, fmt.Errorf("drawing font: %w", err)
	}

	if day.Day > 0 {
		c.SetFont(regular)

		// calculate the size of the text by drawing it onto a blank image
		c.SetDst(image.NewRGBA(image.Rect(0, 0, 1280, 720)))
		pos, err := c.DrawString(fmt.Sprintf("Day %d: %s", day.Day, day.Short), freetype.Pt(0, 0))
		if err != nil {
			return nil, fmt.Errorf("drawing font: %w", err)
		}

		c.SetDst(rgba)

		draw.Draw(
			rgba,
			image.Rectangle{
				Min: image.Point{
					X: 0,
					Y: 500,
				},
				Max: image.Point{
					X: pos.X.Round() + 100,
					Y: 635,
				},
			},
			image.NewUniform(color.NRGBA{0, 0, 0, 128}),
			image.Point{},
			draw.Over,
		)

		_, err = c.DrawString(fmt.Sprintf("Day %d: %s", day.Day, day.Short), freetype.Pt(50, 590))
		if err != nil {
			return nil, fmt.Errorf("drawing font: %w", err)
		}
	}

	r, w := io.Pipe()

	go func() {
		err := jpeg.Encode(w, rgba, nil)
		if err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	return r, nil
}

func getFont(fname string) (*truetype.Font, error) {
	fontBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("reading font file: %w", err)
	}
	fontParsed, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing font file: %w", err)
	}
	return fontParsed, nil
}

func previewThumbnails(ctx context.Context) error {
	daysOrdered, err := getDays()
	if err != nil {
		return fmt.Errorf("can't load days: %w", err)
	}

	daysOrdered = []*GhtDay{
		{
			Day: 0,
		},
	}

	daysByIndex := map[int]*GhtDay{}
	for _, day := range daysOrdered {
		daysByIndex[day.Day] = day
	}

	files, err := ioutil.ReadDir(thumbnailTestingImportDir)
	if err != nil {
		return fmt.Errorf("getting files in folder: %w", err)
	}

	for _, f := range files {
		matches := filenameRegex.FindStringSubmatch(f.Name())
		if len(matches) != 2 {
			continue
		} else {
			dayNumber, err := strconv.Atoi(matches[1])
			if err != nil {
				return fmt.Errorf("parsing day number from %q: %w", f.Name, err)
			}
			day := daysByIndex[dayNumber]
			if day == nil {
				continue
			}
			day.ThumbnailTesting = f
		}
	}

	for _, day := range daysOrdered {

		if day.ThumbnailTesting == nil {
			continue
		}

		fmt.Println("Opening thumbnail", day.Day)
		input, err := os.Open(filepath.Join(thumbnailTestingImportDir, day.ThumbnailTesting.Name()))
		if err != nil {
			return fmt.Errorf("opening thumbnail: %w", err)
		}

		f, err := transformImage(day, input)
		if err != nil {
			input.Close()
			return fmt.Errorf("transforming thumbnail: %w", err)
		}
		input.Close()

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading thumbnail: %w", err)
		}

		err = ioutil.WriteFile(filepath.Join(thumbnailTestingOutputDir, day.ThumbnailTesting.Name()), b, 0666)
		if err != nil {
			return fmt.Errorf("writing thumbnail: %w", err)
		}
		//return nil
	}
	fmt.Print("Done!")
	return nil
}
