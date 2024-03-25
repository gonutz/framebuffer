package framebuffer

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// ioctl makes an ioctl system call.
func ioctl(dev *os.File, cmd, data uintptr) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		dev.Fd(),
		cmd,
		uintptr(data))
	if errno != 0 {
		return os.NewSyscallError(fmt.Sprintf("fb ioctl (cmd=0x%x)", cmd), errno)
	}

	return nil
}

// ioctlGet calls an ioctl, passing it a pointer to a value (of type V) and
// returning the value.
func ioctlGet[V any](dev *os.File, cmd uintptr) (V, error) {
	var v V
	err := ioctl(dev, cmd, uintptr(unsafe.Pointer(&v)))
	return v, err
}
