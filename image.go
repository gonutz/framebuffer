package framebuffer

import (
	"errors"
	"image"
	"image/color"
	"os"
	"syscall"
)

type Image struct {
	fb            *FrameBuffer
	pixels        []byte
	lineLength    int
	bytesPerPixel int
	xRes, yRes    int
	color         colorModel
}

var _ image.Image = (*Image)(nil)

// Open opens the frame buffer device (such as "/dev/fb0") as a drawable image.
func Open(name string) (_ *Image, err error) {
	// Open the frame buffer
	fb, err := OpenFrameBuffer(name, os.O_RDWR)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = fb.File().Close()
		}
	}()

	// Get screen info
	fi, err := fb.FixScreenInfo()
	if err != nil {
		return nil, err
	}
	vi, err := fb.VarScreenInfo()
	if err != nil {
		return nil, err
	}

	switch {
	// Check MSB right
	case vi.Red.MsbRight != 0,
		vi.Green.MsbRight != 0,
		vi.Blue.MsbRight != 0,
		vi.Alpha.MsbRight != 0:
		return nil, errors.New("unsupported msb_right")

	// Check length
	case vi.Red.Length > 16,
		vi.Green.Length > 16,
		vi.Blue.Length > 16,
		vi.Alpha.Length > 16:
		return nil, errors.New("unsupported color model: each component must be 16-bits or less")

	// Check pixel size
	case vi.BitsPerPixel == 0:
		return nil, errors.New("pixel size is zero!")
	case vi.BitsPerPixel%8 != 0:
		return nil, errors.New("unsupported color model: pixel size is not a multiple of 8")
	case vi.BitsPerPixel > 32:
		return nil, errors.New("unsupported color model: pixel size is greater than 32 bits")
	}

	// Memory map the pixels
	pix, err := fb.Pixels()
	if err != nil {
		return nil, err
	}

	return &Image{
		fb:            fb,
		pixels:        pix,
		lineLength:    int(fi.LineLength),
		bytesPerPixel: int(vi.BitsPerPixel) / 8,
		xRes:          int(vi.XRes),
		yRes:          int(vi.YRes),
		color: colorModel{
			r: colorChannel{uint32(vi.Red.Length), uint32(vi.Red.Offset)},
			g: colorChannel{uint32(vi.Green.Length), uint32(vi.Green.Offset)},
			b: colorChannel{uint32(vi.Blue.Length), uint32(vi.Blue.Offset)},
			a: colorChannel{uint32(vi.Alpha.Length), uint32(vi.Alpha.Offset)}},
	}, nil
}

// Close closes the frame buffer.
func (m *Image) Close() error {
	return errors.Join(
		// Unmap and close the frame buffer
		syscall.Munmap(m.pixels),
		m.fb.File().Close(),
	)
}

// Width returns the width of the frame buffer.
func (m *Image) Width() int { return m.xRes }

// Height returns the height of the frame buffer.
func (m *Image) Height() int { return m.yRes }

// Bounds returns the bounds of the frame buffer.
func (m *Image) Bounds() image.Rectangle {
	return image.Rect(
		0, 0,
		int(m.xRes),
		int(m.yRes))
}

// ColorModel returns the color model of the frame buffer.
func (m *Image) ColorModel() color.Model { return &m.color }

// At returns the color at the given coordinates.
func (m *Image) At(x, y int) color.Color {
	if x < 0 || x >= m.xRes ||
		y < 0 || y >= m.yRes {
		return &colorValue{0, m.color}
	}
	return &colorValue{m.read(x, y), m.color}
}

// Set sets the color at the given coordinates.
func (m *Image) Set(x, y int, c color.Color) {
	// the min bounds are at 0,0 (see Open)
	if x >= 0 && x < m.xRes &&
		y >= 0 && y < m.yRes {
		m.write(x, y, m.color.convert(c))
	}
}

// This assumes a little endian system. The m.pixels indices have to be swapped
// if the target system is big endian.

//go:inline
func (m *Image) read(x, y int) uint32 {
	i := y*m.lineLength + m.bytesPerPixel*x
	var val uint32
	for j := 0; j < m.bytesPerPixel; j++ {
		val |= uint32(m.pixels[i+j] << (8 * j))
	}
	return val
}

//go:inline
func (m *Image) write(x, y int, v uint32) {
	i := y*m.lineLength + m.bytesPerPixel*x
	for j := 0; j < m.bytesPerPixel; j++ {
		m.pixels[i+j] = byte(v >> (8 * j))
	}
}
