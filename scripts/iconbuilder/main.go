package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

var iconSizes = []int{16, 20, 24, 32, 40, 48, 64, 128, 256}

type iconImage struct {
	size int
	data []byte
}

func main() {
	inputPath := flag.String("input", "", "source PNG path")
	iconPath := flag.String("ico-output", "", "destination ICO path")
	pngPath := flag.String("png-output", "", "optional destination for a 256px PNG")
	flag.Parse()

	if *inputPath == "" || *iconPath == "" {
		fatal("-input and -ico-output are required")
	}
	source, err := decodePNG(*inputPath)
	if err != nil {
		fatal(err.Error())
	}
	images, err := renderIconImages(source)
	if err != nil {
		fatal(err.Error())
	}
	if err := writeICO(*iconPath, images); err != nil {
		fatal(err.Error())
	}
	if *pngPath != "" {
		if err := os.WriteFile(*pngPath, images[len(images)-1].data, 0o644); err != nil {
			fatal(fmt.Sprintf("write PNG: %v", err))
		}
	}
}

func decodePNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source PNG: %w", err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode source PNG: %w", err)
	}
	return decoded, nil
}

func renderIconImages(source image.Image) ([]iconImage, error) {
	images := make([]iconImage, 0, len(iconSizes))
	for _, size := range iconSizes {
		resized := resizeBilinear(source, size, size)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, resized); err != nil {
			return nil, fmt.Errorf("encode %dpx icon: %w", size, err)
		}
		images = append(images, iconImage{size: size, data: encoded.Bytes()})
	}
	return images, nil
}

func resizeBilinear(source image.Image, width, height int) *image.NRGBA {
	bounds := source.Bounds()
	destination := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		sourceY := (float64(y)+0.5)*float64(bounds.Dy())/float64(height) - 0.5
		for x := range width {
			sourceX := (float64(x)+0.5)*float64(bounds.Dx())/float64(width) - 0.5
			destination.SetNRGBA(x, y, sampleBilinear(source, sourceX, sourceY))
		}
	}
	return destination
}

func sampleBilinear(source image.Image, x, y float64) color.NRGBA {
	x0, xWeight := floorAndWeight(x)
	y0, yWeight := floorAndWeight(y)
	bounds := source.Bounds()
	x0 = clamp(x0+bounds.Min.X, bounds.Min.X, bounds.Max.X-1)
	y0 = clamp(y0+bounds.Min.Y, bounds.Min.Y, bounds.Max.Y-1)
	x1 := clamp(x0+1, bounds.Min.X, bounds.Max.X-1)
	y1 := clamp(y0+1, bounds.Min.Y, bounds.Max.Y-1)

	top := blend(color.NRGBAModel.Convert(source.At(x0, y0)).(color.NRGBA), color.NRGBAModel.Convert(source.At(x1, y0)).(color.NRGBA), xWeight)
	bottom := blend(color.NRGBAModel.Convert(source.At(x0, y1)).(color.NRGBA), color.NRGBAModel.Convert(source.At(x1, y1)).(color.NRGBA), xWeight)
	return blend(top, bottom, yWeight)
}

func floorAndWeight(value float64) (int, float64) {
	integer := int(value)
	if value < float64(integer) {
		integer--
	}
	return integer, value - float64(integer)
}

func blend(left, right color.NRGBA, weight float64) color.NRGBA {
	interpolate := func(a, b uint8) uint8 {
		return uint8(float64(a)*(1-weight) + float64(b)*weight + 0.5)
	}
	return color.NRGBA{
		R: interpolate(left.R, right.R),
		G: interpolate(left.G, right.G),
		B: interpolate(left.B, right.B),
		A: interpolate(left.A, right.A),
	}
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func writeICO(path string, images []iconImage) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create ICO: %w", err)
	}
	defer file.Close()

	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(len(images))); err != nil {
		return err
	}

	offset := uint32(6 + 16*len(images))
	for _, icon := range images {
		dimension := byte(icon.size)
		if icon.size == 256 {
			dimension = 0
		}
		entry := []byte{dimension, dimension, 0, 0}
		if _, err := file.Write(entry); err != nil {
			return err
		}
		for _, value := range []any{uint16(1), uint16(32), uint32(len(icon.data)), offset} {
			if err := binary.Write(file, binary.LittleEndian, value); err != nil {
				return err
			}
		}
		offset += uint32(len(icon.data))
	}
	for _, icon := range images {
		if _, err := file.Write(icon.data); err != nil {
			return err
		}
	}
	return nil
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
