package igo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io"
)

// Serialize GIF image into Kitty terminal in-band format.
func KittyWriteGIF(out io.Writer, gifReader io.Reader, opts KittyImgOpts) error {
	frames, delays := DecodeGIF(gifReader)
	if frames == nil {
		return errors.New("failed to decode GIF")
	}

	opts.SrcHeight = uint32(frames[0].Bounds().Dx())
	opts.SrcWidth = uint32(frames[0].Bounds().Dy())

	// Use the first frame as the base image
	if err := KittyWriteImage(out, frames[0], opts); err != nil {
		return err
	}

	// Write each frame with appropriate frame numbers
	for i, frame := range frames[1:] {
		if err := KittyWriteFrame(out, frame, opts, delays[i]); err != nil {
			return err
		}
	}

	return nil

	// // Control the animation
	// return KittyControlAnimation(out, opts, delays)
}

func DecodeGIF(r io.Reader) ([]image.Image, []int) {
	img, err := gif.DecodeAll(r)
	if err != nil {
		fmt.Println("Error decoding GIF:", err)
		return nil, nil
	}

	frames := make([]image.Image, len(img.Image))
	delays := make([]int, len(img.Delay))
	for i, frame := range img.Image {
		frames[i] = frame
		delays[i] = img.Delay[i]
	}

	return frames, delays
}

func KittyWriteFrame(out io.Writer, frame image.Image, opts KittyImgOpts, delay int) error {
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
		fmt.Sprintf("z=%d", delay*10),
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

func KittyControlAnimation(out io.Writer, opts KittyImgOpts, delays []int) error {
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
