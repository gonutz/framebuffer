package framebuffer

/*
#include <sys/ioctl.h>
#include <linux/fb.h>

struct fb_fix_screeninfo getFixScreenInfo(int fd) {
	struct fb_fix_screeninfo info;
	ioctl(fd, FBIOGET_FSCREENINFO, &info);
	return info;
}

struct fb_var_screeninfo getVarScreenInfo(int fd) {
	struct fb_var_screeninfo info;
	ioctl(fd, FBIOGET_VSCREENINFO, &info);
	return info;
}
*/
import "C"
import (
	"errors"
	"image"
	"image/color"
	"os"
	"syscall"
)

// Open expects a framebuffer device as its argument. It is opened and its
// memory is memory-mapped to a local buffer. Writing to this buffer changes
// the screen output. The returned Device implements the draw.Image interface.
// This means that you can use it to copy to and from other images.
// Currently only the RGB565 color model is supported.
// After you are done using the Device, call Close on it to unmap the memory
// and close the framebuffer file.
func Open(device string) (*Device, error) {
	file, err := os.OpenFile(device, os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}

	fixInfo := C.getFixScreenInfo(C.int(file.Fd()))
	varInfo := C.getVarScreenInfo(C.int(file.Fd()))

	pixels, err := syscall.Mmap(
		int(file.Fd()),
		0, int(varInfo.xres*varInfo.yres*varInfo.bits_per_pixel/8),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		file.Close()
		return nil, err
	}

	var colorModel color.Model
	if varInfo.red.offset == 11 && varInfo.red.length == 5 && varInfo.red.msb_right == 0 &&
		varInfo.green.offset == 5 && varInfo.green.length == 6 && varInfo.green.msb_right == 0 &&
		varInfo.blue.offset == 0 && varInfo.blue.length == 5 && varInfo.blue.msb_right == 0 {
		colorModel = rgb565ColorModel{}
	} else {
		return nil, errors.New("unsupported color model")
	}

	return &Device{
		file,
		pixels,
		int(fixInfo.line_length),
		image.Rect(0, 0, int(varInfo.xres), int(varInfo.yres)),
		colorModel,
	}, nil
}

// Device represents the frame buffer. It implements the draw.Image interface.
type Device struct {
	file       *os.File
	pixels     []byte
	pitch      int
	bounds     image.Rectangle
	colorModel color.Model
}

// Close unmaps the framebuffer memory and closes the device file. Call this
// function when you are done using the frame buffer.
func (d *Device) Close() {
	syscall.Munmap(d.pixels)
	d.file.Close()
}

// Bounds implements the image.Image (and draw.Image) interface.
func (d *Device) Bounds() image.Rectangle {
	return d.bounds
}

// ColorModel implements the image.Image (and draw.Image) interface.
func (d *Device) ColorModel() color.Model {
	return d.colorModel
}

// At implements the image.Image (and draw.Image) interface.
func (d *Device) At(x, y int) color.Color {
	if x < d.bounds.Min.X || x >= d.bounds.Max.X ||
		y < d.bounds.Min.Y || y >= d.bounds.Max.Y {
		return rgb565(0)
	}
	i := y*d.pitch + 2*x
	return rgb565(d.pixels[i+1])<<8 | rgb565(d.pixels[i])
}

// Set implements the draw.Image interface.
func (d *Device) Set(x, y int, c color.Color) {
	// the min bounds are at 0,0 (see Open)
	if x >= 0 && x < d.bounds.Max.X &&
		y >= 0 && y < d.bounds.Max.Y {
		r, g, b, a := c.RGBA()
		if a > 0 {
			rgb := toRGB565(r, g, b)
			i := y*d.pitch + 2*x
			// This assumes a little endian system which is the default for
			// Raspbian. The d.pixels indices have to be swapped if the target
			// system is big endian.
			d.pixels[i+1] = byte(rgb >> 8)
			d.pixels[i] = byte(rgb & 0xFF)
		}
	}
}

// The default color model under the Raspberry Pi is RGB 565. Each pixel is
// represented by two bytes, with 5 bits for red, 6 bits for green and 5 bits
// for blue. There is no alpha channel, so alpha is assumed to always be 100%
// opaque.
// This shows the memory layout of a pixel:
//
//    bit 76543210  76543210
//        RRRRRGGG  GGGBBBBB
//       high byte  low byte
type rgb565ColorModel struct{}

func (rgb565ColorModel) Convert(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	return toRGB565(r, g, b)
}

// toRGB565 helps convert a color.Color to rgb565. In a color.Color each
// channel is represented as a uint32 but the maximum value is 0xFFFF. This
// simply uses the highest 5 or 6 bits of each channel as the RGB values.
func toRGB565(r, g, b uint32) rgb565 {
	return rgb565((r & 0xF800) +
		((g & 0xFC00) >> 5) +
		((b & 0xF800) >> 11))
}

// rgb565 implements the color.Color interface.
type rgb565 uint16

// RGBA implements the color.Color interface.
func (c rgb565) RGBA() (r, g, b, a uint32) {
	// The conversion from 5 and 6 bit channels to 16 bits is lossy. It simply
	// shifts each channel's bits to the highest position which means that the
	// maximum value of 0xFFFF can never be reached for example. The low 15 or
	// 16 bits of the channels will always be 0.
	// Alpha is always 100% opaque since this model does not support it.
	r = uint32(c & 0xF800)
	g = uint32((c & 0x7E0) << 5)
	b = uint32((c & 0x1F) << 11)
	a = 0xFFFF
	return
}
