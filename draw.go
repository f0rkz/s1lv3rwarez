package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func drawHLine(dst *ebiten.Image, x0, x1, y int, c color.Color) {
	if y < 0 || y >= virtualHeight {
		return
	}
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 >= virtualWidth {
		x1 = virtualWidth - 1
	}
	if x0 > x1 {
		return
	}
	for x := x0; x <= x1; x++ {
		dst.Set(x, y, c)
	}
}

func drawVLine(dst *ebiten.Image, x, y0, y1 int, c color.Color) {
	if x < 0 || x >= virtualWidth {
		return
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	if y0 < 0 {
		y0 = 0
	}
	if y1 >= virtualHeight {
		y1 = virtualHeight - 1
	}
	if y0 > y1 {
		return
	}
	for y := y0; y <= y1; y++ {
		dst.Set(x, y, c)
	}
}

func drawLine(dst *ebiten.Image, x0, y0, x1, y1 int, c color.Color) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if x0 >= 0 && x0 < virtualWidth && y0 >= 0 && y0 < virtualHeight {
			dst.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
