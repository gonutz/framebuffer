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

// Open expects a framebuffer device as its argument (such as "/dev/fb0"). The
// device will be memory-mapped to a local buffer. Writing to the device changes
// the screen output.
// The returned Device implements the draw.Image interface. This means that you
// can use it to copy to and from other images.
// The only supported color model for the specified frame buffer is RGB565.
// After you are done using the Device, call Close on it to unmap the memory and
// close the framebuffer file.
func Open(device string) (*Device, error) {
	file, err := os.OpenFile(device, os.O_RDWR, os.ModeDevice)
	if err != nil {
		return nil, err
	}

	fixInfo := C.getFixScreenInfo(C.int(file.Fd()))
	varInfo := C.getVarScreenInfo(C.int(file.Fd()))

	if varInfo.red.msb_right != 0 ||
		varInfo.green.msb_right != 0 ||
		varInfo.blue.msb_right != 0 ||
		varInfo.transp.msb_right != 0 {
		return nil, errors.New("unsupported msb_right")
	}

	if varInfo.red.length > 16 ||
		varInfo.green.length > 16 ||
		varInfo.blue.length > 16 ||
		varInfo.transp.length > 16 {
		return nil, errors.New("unsupported color model: each component must be 16-bits or less")
	}

	if varInfo.bits_per_pixel%8 != 0 {
		return nil, errors.New("unsupported color model: total size is not a multiple of 8")
	}
	if varInfo.bits_per_pixel > 32 {
		return nil, errors.New("unsupported color model: total size is greater than 32 bits")
	}

	pixels, err := syscall.Mmap(
		int(file.Fd()),
		0, int(varInfo.xres*varInfo.yres*varInfo.bits_per_pixel/8),
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED,
	)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &Device{
		file:   file,
		pixels: pixels,
		pitch:  int(fixInfo.line_length),
		bytes:  int(varInfo.bits_per_pixel) / 8,
		bounds: image.Rect(0, 0, int(varInfo.xres), int(varInfo.yres)),
		color: colorModel{
			r: colorChannel{uint32(varInfo.red.length), uint32(varInfo.red.offset)},
			g: colorChannel{uint32(varInfo.green.length), uint32(varInfo.green.offset)},
			b: colorChannel{uint32(varInfo.blue.length), uint32(varInfo.blue.offset)},
			a: colorChannel{uint32(varInfo.transp.length), uint32(varInfo.transp.offset)},
		},
	}, nil
}

// Device represents the frame buffer. It implements the draw.Image interface.
type Device struct {
	file   *os.File
	pixels []byte
	pitch  int
	bytes  int
	bounds image.Rectangle
	color  colorModel
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
	return &d.color
}

// At implements the image.Image (and draw.Image) interface.
func (d *Device) At(x, y int) color.Color {
	if x < d.bounds.Min.X || x >= d.bounds.Max.X ||
		y < d.bounds.Min.Y || y >= d.bounds.Max.Y {
		return &colorValue{0, d.color}
	}
	return &colorValue{d.read(x, y), d.color}
}

// Set implements the draw.Image interface.
func (d *Device) Set(x, y int, c color.Color) {
	// the min bounds are at 0,0 (see Open)
	if x >= 0 && x < d.bounds.Max.X &&
		y >= 0 && y < d.bounds.Max.Y {
		d.write(x, y, d.color.convert(c))
	}
}

// This assumes a little endian system which is the default for Raspbian. The
// d.pixels indices have to be swapped if the target system is big endian.

func (d *Device) read(x, y int) uint32 {
	i := y*d.pitch + d.bytes*x
	var val uint32
	for j := 0; j < d.bytes; j++ {
		val |= uint32(d.pixels[i+j] << (8 * j))
	}
	return val
}

func (d *Device) write(x, y int, v uint32) {
	i := y*d.pitch + d.bytes*x
	for j := 0; j < d.bytes; j++ {
		d.pixels[i+j] = byte(v >> (8 * j))
	}
}
