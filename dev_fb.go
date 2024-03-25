package framebuffer

import (
	"os"
	"syscall"
)

// FrameBuffer is a wrapper around [os.File] that provides convenience methods
// for interacting with a frame buffer device.
type FrameBuffer os.File

// OpenFrameBuffer opens a frame buffer device.
func OpenFrameBuffer(name string, flags int) (*FrameBuffer, error) {
	f, err := os.OpenFile(name, flags, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	return (*FrameBuffer)(f), nil
}

// File converts the frame buffer back to a file.
func (fb *FrameBuffer) File() *os.File {
	return (*os.File)(fb)
}

// Pixels returns a byte slice memory mapped to the frame buffer.
func (fb *FrameBuffer) Pixels() ([]byte, error) {
	vi, err := fb.VarScreenInfo()
	if err != nil {
		return nil, err
	}

	fd := fb.File().Fd()
	length := vi.XRes * vi.YRes * vi.BitsPerPixel / 8
	flags := syscall.PROT_READ | syscall.PROT_WRITE
	return syscall.Mmap(int(fd), 0, int(length), flags, syscall.MAP_SHARED)
}

// FixScreenInfo returns the fixed properties of the screen.
func (fb *FrameBuffer) FixScreenInfo() (FbFixScreenInfo, error) {
	return ioctlGet[FbFixScreenInfo](fb.File(), kFBIOGET_FSCREENINFO)
}

// VarScreenInfo returns the variable properties of the screen.
func (fb *FrameBuffer) VarScreenInfo() (FbVarScreenInfo, error) {
	return ioctlGet[FbVarScreenInfo](fb.File(), kFBIOGET_VSCREENINFO)
}

// <linux/fb.h> ioctls
//
// 0x46 is 'F'
const (
	kFBIOGET_VSCREENINFO = 0x4600
	kFBIOPUT_VSCREENINFO = 0x4601
	kFBIOGET_FSCREENINFO = 0x4602
)

// <linux/fb.h> struct fb_fix_screeninfo
//
// Fixed parameters of a screen
type FbFixScreenInfo struct {
	// Screen ID, e.g. "TT Builtin"
	ID [16]byte

	// Frame buffer mem (physical address)
	SMemStart uintptr
	SMemLen   uint32

	// Framebuffer type (see FB_TYPE_*)
	Type uint32

	// "Interleave for interleaved Planes"
	TypeAux uint32

	// ??? (see FB_VISUAL_)
	Visual uint32

	XPanStep  uint16
	YPanStep  uint16
	YWrapStep uint16

	// Length of a line in bytes
	LineLength uint32

	// Memory mapped I/O (physical address)
	MmioStart uintptr
	MmioLen   uint32

	// "Indicate to driver which specific chip/card we have"
	Accel uint32

	// Capabilities (see FB_CAP_*)
	Capabilities uint16

	// Reserved
	_ [2]uint16
}

// <linux/fb.h> struct fb_fix_screeninfo
//
// Variable parameters of a screen
type FbVarScreenInfo struct {
	// Visible resolution
	XRes, YRes uint32

	// Virtual resolution
	XResVirtual, YResVirtual uint32

	// Offset from virtual to visible resolution
	XOffset, YOffset uint32

	// Number of bits per pixel in the buffer
	BitsPerPixel uint32

	// 0 = color, 1 = grayscale, >1 = FOURCC
	Grayscale uint32

	// Pixel format
	Red, Green, Blue, Alpha FbBitField

	// A value other than zero indicates a non-standard pixel format
	NonStd uint32

	// ??? (see FB_ACTIVATE_*)
	Activate uint32

	// Dimensions of picture in mm
	Height, Width uint32

	// Deprecated, used to be accel_flags (see fb_info.flags)
	_ uint32

	// Pixel clock in pico seconds
	PixelClock uint32

	// Time from sync to picture
	LeftMargin uint32

	// Time from picture to sync
	RightMargin uint32

	UpperMargin uint32
	LowerMargin uint32

	// Length of horizontal sync
	HSyncLen uint32

	// Length of vertical sync
	VSyncLen uint32

	// ??? (see FB_SYNC_*)
	Sync uint32

	// ??? (see FB_VMODE_*)
	VMode uint32

	// Angle we rotate counter clockwise
	Rotate uint32

	// Color space for FOURCC-based modes
	ColorSpace uint32

	// Reserved
	_ [4]uint32
}

// <linux/fb.h> struct fb_bitfield
//
// Interpretation of offset for color fields: All offsets are from the right,
// inside a "pixel" value, which is exactly 'bits_per_pixel' wide (means: you
// can use the offset as right argument to <<). A pixel afterwards is a bit
// stream and is written to video memory as that unmodified.
//
// For pseudocolor: offset and length should be the same for all color
// components. Offset specifies the position of the least significant bit of the
// palette index in a pixel value. Length indicates the number of available
// palette entries (i.e. # of entries = 1 << length).
type FbBitField struct {
	Offset   uint32
	Length   uint32
	MsbRight uint32
}
