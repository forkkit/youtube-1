package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"

	"github.com/disintegration/imaging"
)

func transformImage(day *GhtDay, file io.Reader) (io.Reader, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	// 1280x720

	imgCropped := imaging.Fill(img, 1280, 720, imaging.Center, imaging.Lanczos)

	r, w := io.Pipe()

	go func() {
		err := jpeg.Encode(w, imgCropped, nil)
		if err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	return r, nil
}
