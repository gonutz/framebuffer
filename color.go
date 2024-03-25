package framebuffer

import (
	"image/color"
)

// colorModel is an RGBA color model.
type colorModel struct {
	r, g, b, a colorChannel
}

// colorChannel is one color of a [colorModel].
type colorChannel struct {
	length, offset uint32
}

// colorValue is [colorModel]'s implementation of [color.Color].
type colorValue struct {
	value uint32
	model colorModel
}

// Convert converts a color to this model.
func (cm *colorModel) Convert(c color.Color) color.Color {
	return &colorValue{cm.convert(c), *cm}
}

// RGBA returns each color channel as a separate value.
func (c *colorValue) RGBA() (r, g, b, a uint32) {
	return c.model.r.unshift(c.value),
		c.model.g.unshift(c.value),
		c.model.b.unshift(c.value),
		c.model.a.unshift(c.value)
}

// convert converts a color this model.
func (cm colorModel) convert(c color.Color) uint32 {
	r, g, b, a := c.RGBA()
	return cm.r.shift(r) |
		cm.g.shift(g) |
		cm.b.shift(b) |
		cm.a.shift(a)
}

// shift shifts and masks the value to the offset and length of this channel.
func (ch colorChannel) shift(val uint32) uint32 {
	// Skip the math if this channel is not used
	if ch.length == 0 {
		return 0
	}

	// Shift and mask
	var mask uint32 = 1<<ch.length - 1
	val >>= 16 - ch.length
	return (val & mask) << uint32(ch.offset)
}

// unshift extracts the value of this channel.
func (ch colorChannel) unshift(val uint32) uint32 {
	// The alpha channel is the only one that should have zero length. If the
	// alpha channel doesn't exist, pretend like it's value is the max (full
	// opacity).
	if ch.length == 0 {
		return 0xFFFF
	}

	// Mask and shift
	var mask uint32 = 1<<ch.length - 1
	val = (val >> ch.offset) & mask
	return val << (16 - ch.length)
}
