# About
The framebuffer library was created to have an easy way to access the pixels on the screen from the Raspberry Pi.
It memory-maps the framebuffer device and provides it as a draw.Image (which is itself an image.Image).
This makes it easy to use with Go's image, color and draw packages.

Right now the library only implements the RGB 565 color model, which is the default under Raspbian. Also the OS is assumed to be
little endian, also the default for Raspbian.

# Usage
To access the framebuffer you have to call Open and pass the device file to it. When you are done, call Close on the returned device.
Note that you usually need root access for this so make sure to run your program as a super user.

Once you have a device open you can use it like a Go draw.Image (image.Image).

Here is a simple example that clears the whole screen to a dark magenta:

```Go
package main

import (
	"github.com/gonutz/framebuffer"
	"image"
	"image/color"
	"image/draw"
)

func main() {
	fb, err := framebuffer.Open("/dev/fb0")
	if err != nil {
		panic(err)
	}
	defer fb.Close()

	magenta := image.NewUniform(color.RGBA{255, 0, 128, 255})
	draw.Draw(fb, fb.Bounds(), magenta, image.ZP, draw.Src)
}
```
