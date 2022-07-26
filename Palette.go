package neuquant

import (
	"image"
	"image/color"
	"sync"
)

func Paletted(img image.Image) *image.Paletted {
	nq, palette := AnalyzePalette(img)
	img2 := image.NewPaletted(img.Bounds(), palette)
	ParallelWritePalette(nq, img, img2)
	return img2
}

const sample = 10 // default sample interval for quantizer

func AnalyzePalette(img image.Image) (*NeuQuant, color.Palette) {
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
	nq := NewNeuQuant(pixels, length, sample)
	// initialize quantizer
	colorTab := nq.ColorMap() // create reduced palette
	// convert map from BGR to RGB
	length = len(colorTab)
	palette := make(color.Palette, length/3)
	for i := 0; i < length; i += 3 {
		palette[i/3] = &color.RGBA{R: colorTab[i+2], G: colorTab[i+1], B: colorTab[i], A: 255}
	}
	return nq, palette
}

func ParallelWritePalette(nq *NeuQuant, src image.Image, dst *image.Paletted) {
	var wg sync.WaitGroup
	wg.Add(4 * 4)
	rect := src.Bounds()
	x, y := make([]int, 5), make([]int, 5)
	x[0], y[0], x[4], y[4] = rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y
	x[2], y[2] = (x[0]+x[4])/2, (y[0]+y[4])/2
	x[1], y[1], x[3], y[3] = (x[0]+x[2])/2, (y[0]+y[2])/2, (x[2]+x[4])/2, (y[2]+y[4])/2
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			go func(i, j int) {
				WritePalette(nq, src, dst, image.Rect(x[i], y[j], x[i+1], y[j+1]))
				wg.Done()
			}(i, j)
		}
	}
	wg.Wait()
}

func WritePalette(nq *NeuQuant, src image.Image, dst *image.Paletted, rect image.Rectangle) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			r, g, b, _ := src.At(x, y).RGBA()
			dst.SetColorIndex(x, y, uint8(nq.Map(int(b>>8), int(g>>8)&0xff, int(r>>8)&0xff)))
		}
	}
}
