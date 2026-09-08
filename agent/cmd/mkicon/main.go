// mkicon génère les icônes .ico de la barre des tâches (disque de couleur). Usage : go run ./cmd/mkicon <dir>
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func discPNG(fill color.RGBA) []byte {
	ico := disc(fill)
	return ico[22:] // le PNG encapsulé suit l'en-tête ICO de 22 octets
}

func disc(fill color.RGBA) []byte {
	const n = 32
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	cx, cy, r := 15.5, 15.5, 14.0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			d := dx*dx + dy*dy
			switch {
			case d <= (r-2)*(r-2):
				img.Set(x, y, fill)
			case d <= r*r:
				img.Set(x, y, color.RGBA{20, 22, 28, 255}) // liseré sombre
			}
		}
	}
	// barre blanche horizontale et verticale : un "F" stylisé
	white := color.RGBA{255, 255, 255, 235}
	for x := 11; x < 22; x++ {
		img.Set(x, 9, white); img.Set(x, 10, white); img.Set(x, 11, white)
	}
	for x := 11; x < 19; x++ {
		img.Set(x, 15, white); img.Set(x, 16, white)
	}
	for y := 9; y < 24; y++ {
		img.Set(11, y, white); img.Set(12, y, white); img.Set(13, y, white)
	}
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	var ico bytes.Buffer
	binary.Write(&ico, binary.LittleEndian, struct{ Reserved, Type, Count uint16 }{0, 1, 1})
	binary.Write(&ico, binary.LittleEndian, struct {
		W, H, Colors, Reserved uint8
		Planes, Bits           uint16
		Size, Offset           uint32
	}{n, n, 0, 0, 1, 32, uint32(pngBuf.Len()), 22})
	ico.Write(pngBuf.Bytes())
	return ico.Bytes()
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	os.WriteFile(filepath.Join(dir, "online.ico"), disc(color.RGBA{61, 220, 132, 255}), 0o644)
	os.WriteFile(filepath.Join(dir, "warn.ico"), disc(color.RGBA{255, 176, 32, 255}), 0o644)
	os.WriteFile(filepath.Join(dir, "offline.ico"), disc(color.RGBA{255, 92, 92, 255}), 0o644)
	os.WriteFile(filepath.Join(dir, "app.png"), discPNG(color.RGBA{91, 156, 255, 255}), 0o644) // icône de l'application (bleu)
}
