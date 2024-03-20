package framebuffer

import (
	"image/color"
)

type colorModel struct {
	r, g, b, a colorChannel
}

type colorChannel struct {
	length, offset uint32
}

type colorValue struct {
	value uint32
	model colorModel
}

func (cm *colorModel) Convert(c color.Color) color.Color {
	return &colorValue{cm.convert(c), *cm}
}

func (c *colorValue) RGBA() (r, g, b, a uint32) {
	return c.model.r.unshift(c.value),
		c.model.g.unshift(c.value),
		c.model.b.unshift(c.value),
		c.model.a.unshift(c.value)
}

func (cm colorModel) convert(c color.Color) uint32 {
	r, g, b, a := c.RGBA()
	return cm.r.shift(r) |
		cm.g.shift(g) |
		cm.b.shift(b) |
		cm.a.shift(a)
}

func (ch colorChannel) shift(val uint32) uint32 {
	if ch.length == 0 {
		return 0
	}
	var mask uint32 = 1<<ch.length - 1
	val >>= 16 - ch.length
	return (val & mask) << uint32(ch.offset)
}

func (ch colorChannel) unshift(val uint32) uint32 {
	if ch.length == 0 {
		return 0xFFFF
	}
	var mask uint32 = 1<<ch.length - 1
	val = (val >> ch.offset) & mask
	return val << (16 - ch.length)
}
