package arw

import (
	"image"
	"image/color"
)

// NewRGBA returns a new RGBA image with the given bounds.
func NewRGB14(r image.Rectangle) *RGB14 {
	w, h := r.Dx(), r.Dy()
	buf := make([]pixel16, w*h)
	return &RGB14{buf, w, r}
}

// RGBA64 is an in-memory image whose At method returns pixel16 values.
type RGB14 struct {
	Pix []pixel16
	// Stride is the Pix stride between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func (r *RGB14) at(x, y int) pixel16 {
	return r.Pix[(y*r.Stride)+x]
}

func (r *RGB14) At(x, y int) color.Color {
	return r.at(x, y)
}

func (r *RGB14) Bounds() image.Rectangle {
	return r.Rect.Bounds()
}

func (r *RGB14) ColorModel() color.Model {
	return color.RGBA64Model
}

func (c pixel16) RGBA() (uint32, uint32, uint32, uint32) {
	r := uint32(c.R) << 2
	g := uint32(c.G) << 2
	b := uint32(c.B) << 2
	if r > 0xffff {
		r = 0xffff
	}
	if g > 0xffff {
		g = 0xffff
	}
	if b > 0xffff {
		b = 0xffff
	}
	return r, g, b, 0xffff
	//return uint32(c.R), uint32(c.G), uint32(c.B), 0xffff
}

func (r *RGB14) set(x, y int, pixel pixel16) {
	r.Pix[y*r.Stride+x] = pixel
}

type pixel16 struct {
	R uint16
	G uint16
	B uint16
	_ uint16
}
