package framebuffer

import "os"

// TTY is a wrapper around [os.File] that provides convenience methods for
// interacting with a TTY.
type TTY os.File

// OpenMyTTY opens the current process's TTY. Requires procfs mounted at /proc.
func OpenMyTTY(flags int) (*TTY, error) {
	name, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		return nil, err
	}
	return OpenTTY(name, flags)
}

// OpenTTY opens a TTY.
func OpenTTY(name string, flags int) (*TTY, error) {
	f, err := os.OpenFile(name, flags, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	return (*TTY)(f), nil
}

// File converts the TTY back to a file.
func (d *TTY) File() *os.File {
	return (*os.File)(d)
}

// TextMode puts the TTY in text mode.
func (d *TTY) TextMode() error {
	return d.SetMode(kKD_TEXT)
}

// GraphicsMode puts the TTY in graphics mode.
func (d *TTY) GraphicsMode() error {
	return d.SetMode(kKD_GRAPHICS)
}

// SetMode sets the TTY's mode.
func (d *TTY) SetMode(mode TTYMode) error {
	return ioctl(d.File(), kKDSETMODE, uintptr(mode))
}

// GetMode returns the TTY's current mode.
func (d *TTY) GetMode() (TTYMode, error) {
	return ioctlGet[TTYMode](d.File(), kKDGETMODE)
}

type TTYMode int

const (
	TTYTextMode     TTYMode = kKD_TEXT
	TTYGraphicsMode TTYMode = kKD_GRAPHICS
)

// <linux/kd.h> ioctls
//
// 0x4B is 'K', to avoid collision with termios and vt
const (
	kKDSETMODE = 0x4B3A
	kKDGETMODE = 0x4B3B

	kKD_TEXT     = 0x00
	kKD_GRAPHICS = 0x01
)
