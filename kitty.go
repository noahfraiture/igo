package igo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"
)

// See https://sw.kovidgoyal.net/kitty/graphics-protocol.html for more details.

const (
	KITTY_IMG_HDR = "\x1b_G"
	KITTY_IMG_FTR = "\x1b\\"
)

type KittyImgOpts struct {
	SrcX        uint32 // x=
	SrcY        uint32 // y=
	SrcWidth    uint32 // w=
	SrcHeight   uint32 // h=
	CellOffsetX uint32 // X= (pixel x-offset inside terminal cell)
	CellOffsetY uint32 // Y= (pixel y-offset inside terminal cell)
	DstCols     uint32 // c= (display width in terminal columns)
	DstRows     uint32 // r= (display height in terminal rows)
	ZIndex      int32  // z=
	ImageId     uint32 // i=
	ImageNo     uint32 // I=
	PlacementId uint32 // p=
}

func (o KittyImgOpts) ToHeader(opts ...string) string {

	type fldmap struct {
		pv   *uint32
		code rune
	}
	sFld := []fldmap{
		{&o.SrcX, 'x'},
		{&o.SrcY, 'y'},
		{&o.SrcWidth, 'w'},
		{&o.SrcHeight, 'h'},
		{&o.CellOffsetX, 'X'},
		{&o.CellOffsetY, 'Y'},
		{&o.DstCols, 'c'},
		{&o.DstRows, 'r'},
		{&o.ImageId, 'i'},
		{&o.ImageNo, 'I'},
		{&o.PlacementId, 'p'},
	}

	for _, f := range sFld {
		if *f.pv != 0 {
			opts = append(opts, fmt.Sprintf("%c=%d", f.code, *f.pv))
		}
	}

	if o.ZIndex != 0 {
		opts = append(opts, fmt.Sprintf("z=%d", o.ZIndex))
	}

	return KITTY_IMG_HDR + strings.Join(opts, ",") + ";"
}

// checks if terminal supports kitty image protocols
func IsKittyCapable() bool {

	// TODO: more rigorous check
	V := GetEnvIdentifiers()
	return (len(V["KITTY_WINDOW_ID"]) > 0) || (V["TERM_PROGRAM"] == "wezterm")
}

// Display local PNG file
// - pngFileName must be directly accesssible from Kitty instance
// - pngFileName must be an absolute path
func KittyWritePNGLocal(out io.Writer, pngFileName string, opts KittyImgOpts) error {

	_, e := fmt.Fprint(out, opts.ToHeader("a=T", "f=100", "t=f"))
	if e != nil {
		return e
	}

	enc64 := base64.NewEncoder(base64.StdEncoding, out)

	_, e = fmt.Fprint(enc64, pngFileName)
	if e != nil {
		return e
	}

	e = enc64.Close()
	if e != nil {
		return e
	}

	_, e = fmt.Fprint(out, KITTY_IMG_FTR)
	return e
}

// Serialize image.Image into Kitty terminal in-band format.
func KittyWriteImage(out io.Writer, iImg image.Image, opts KittyImgOpts) error {

	pBuf := new(bytes.Buffer)
	if E := png.Encode(pBuf, iImg); E != nil {
		return E
	}

	return KittyWritePngReader(out, pBuf, opts)
}

// Serialize PNG image from io.Reader into Kitty terminal in-band format.
func KittyWritePngReader(out io.Writer, in io.Reader, opts KittyImgOpts) error {

	_, err := fmt.Fprint(out, opts.ToHeader("a=T", "f=100", "t=d", "m=1"), KITTY_IMG_FTR)
	if err != nil {
		return err
	}

	// PIPELINE: PNG (io.Reader) -> B64 -> CHUNKER -> (io.Writer)
	// SEND IN 4K CHUNKS
	chunk := kittyChunk{
		chunkSize: 4096,
		writer:    out,
	}

	enc64 := base64.NewEncoder(base64.StdEncoding, &chunk)
	_, err = io.Copy(enc64, in)
	return errors.Join(
		err,
		enc64.Close(),
		chunk.Close(),
	)
}

func KittyClean(out io.Writer, opts KittyImgOpts) error {
	// Remove the image
	_, err := fmt.Fprint(out, opts.ToHeader("a=d", "d=a"), KITTY_IMG_FTR)
	if err != nil {
		return err
	}

	// Clear the affected terminal lines
	_, err = fmt.Fprint(out, "\033[2J") // Clear the entire screen
	if err != nil {
		return err
	}

	// Optionally, reset the cursor position if needed
	_, err = fmt.Fprint(out, "\033[H") // Move cursor to home position (top-left)
	return err
}
