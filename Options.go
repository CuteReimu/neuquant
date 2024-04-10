package neuquant

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
)

type quantizer struct {
	nq *NeuQuant
}

func (q *quantizer) Quantize(_ color.Palette, img image.Image) color.Palette {
	var pixels []byte
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels = append(pixels, byte(b))
			pixels = append(pixels, byte(g))
			pixels = append(pixels, byte(r))
		}
	}
	length := len(pixels)
	q.nq = NewNeuQuant(pixels, length, sample)
	// initialize quantizer
	colorTab := q.nq.ColorMap() // create reduced palette
	// convert map from BGR to RGB
	length = len(colorTab)
	palette := make(color.Palette, length/3)
	for i := 0; i < length; i += 3 {
		palette[i/3] = &color.RGBA{R: colorTab[i+2], G: colorTab[i+1], B: colorTab[i], A: 255}
	}
	return palette
}

type drawer struct {
	q *quantizer
}

func (d drawer) Draw(dstOri draw.Image, rect image.Rectangle, src image.Image, sp image.Point) {
	dst, ok := dstOri.(*image.Paletted)
	if !ok {
		panic("can only draw on paletted image")
	}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			r, g, b, _ := src.At(x-rect.Min.X+sp.X, y-rect.Min.Y+sp.Y).RGBA()
			dst.SetColorIndex(x, y, uint8(d.q.nq.Map(int(b>>8), int(g>>8)&0xff, int(r>>8)&0xff)))
		}
	}
}

func Opt() *gif.Options {
	q := &quantizer{}
	return &gif.Options{
		NumColors: 256,
		Quantizer: q,
		Drawer:    drawer{q: q},
	}
}
