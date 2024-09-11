package igo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
)

// Serialize GIF image into Kitty terminal in-band format.
func KittyWriteGIF(out io.Writer, gifReader io.Reader, opts KittyImgOpts) error {
	frames, x, y, err := splitAnimatedGIF(gifReader)
	if err != nil {
		return errors.New("failed to decode GIF")
	}

	opts.SrcHeight = y
	opts.SrcWidth = x

	// Use the first frame as the base image
	if err := KittyWriteImage(out, &frames[0], opts); err != nil {
		return err
	}

	// Write each frame with appropriate frame numbers
	for _, frame := range frames[1:] {
		if err := kittyWriteFrame(out, &frame, opts); err != nil {
			return err
		}
	}

	return nil

	// // Control the animation
	// return KittyControlAnimation(out, opts, delays)
}

// Decode reads and analyzes the given reader as a GIF image
func splitAnimatedGIF(reader io.Reader) ([]image.RGBA, uint32, uint32, error) {
	gif, err := gif.DecodeAll(reader)

	if err != nil {
		return nil, 0, 0, err
	}

	imgs := make([]image.RGBA, len(gif.Image))

	imgWidth, imgHeight := getGifDimensions(gif)

	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), gif.Image[0], image.ZP, draw.Src)

	for _, srcImg := range gif.Image {
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		imgs = append(imgs, *overpaintImage)
	}

	return imgs, uint32(imgWidth), uint32(imgHeight), nil
}

func getGifDimensions(gif *gif.GIF) (x, y int) {
	var lowestX int
	var lowestY int
	var highestX int
	var highestY int

	for _, img := range gif.Image {
		if img.Rect.Min.X < lowestX {
			lowestX = img.Rect.Min.X
		}
		if img.Rect.Min.Y < lowestY {
			lowestY = img.Rect.Min.Y
		}
		if img.Rect.Max.X > highestX {
			highestX = img.Rect.Max.X
		}
		if img.Rect.Max.Y > highestY {
			highestY = img.Rect.Max.Y
		}
	}

	return highestX - lowestX, highestY - lowestY
}

func kittyWriteFrame(out io.Writer, frame image.Image, opts KittyImgOpts) error {
	// Ensure width and height are properly set
	if frame.Bounds().Dx() == 0 || frame.Bounds().Dy() == 0 {
		return errors.New("frame width or height is zero")
	}

	headerOpts := []string{
		"a=f",
		"t=d",
		"m=1",
		fmt.Sprintf("s=%d", opts.SrcWidth),
		fmt.Sprintf("v=%d", opts.SrcHeight),
		fmt.Sprintf("i=%d", opts.ImageId),
		fmt.Sprintf("z=%d", 1000),
	}
	fmt.Println(headerOpts)

	_, err := fmt.Fprint(out, opts.ToHeader(headerOpts...))
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, frame); err != nil {
		return err
	}

	chunk := kittyChunk{
		chunkSize: 4096,
		writer:    out,
	}

	enc64 := base64.NewEncoder(base64.StdEncoding, &chunk)
	_, err = io.Copy(enc64, buf)
	return errors.Join(
		err,
		enc64.Close(),
		chunk.Close(),
	)
}

func kittyControlAnimation(out io.Writer, opts KittyImgOpts, delays []int) error {
	// Set animation control options, such as looping
	headersOpts := []string{
		"a=a",
		fmt.Sprintf("i=%d", opts.ImageId),
		fmt.Sprintf("s=%d", opts.SrcWidth),
		fmt.Sprintf("v=%d", opts.SrcHeight),
	}
	_, err := fmt.Fprint(out, opts.ToHeader(headersOpts...), KITTY_IMG_FTR)
	return err
}
