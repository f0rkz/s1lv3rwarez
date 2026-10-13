package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// README hero GIFs. Opens a window and runs the real game loop, so it only
// runs on request:
//
//	S1LV3RWAREZ_GIFS=1 go test -run TestReadmeGIFs .
//
// Each shot seeks that part's song to a showcase moment, warms up for a
// second, then records 3 seconds at 30 fps into docs/part<N>.gif.
var gifShots = []struct {
	part    int
	seconds float64
}{
	{0, 16 * 4 * beatSeconds},             // Part I main theme, logo revealed
	{1, 30 * sequelBeatSeconds},           // Part II: banner typing, installer
	{2, 16 * 4 * lanBeatSeconds},          // Part III: lead enters
	{3, (impactBeat - 6) * raBeatSeconds}, // Part IV: missile in, nuke lands
}

const gifWarmup, gifFrames = 60, 90

type gifCapture struct {
	*Game
	shot, n int
	frames  [][]*image.RGBA
}

func (c *gifCapture) Update() error {
	if c.shot == len(gifShots) {
		return ebiten.Termination
	}
	if c.n == 0 {
		s := gifShots[c.shot]
		for c.part != s.part {
			if err := c.switchPart(); err != nil {
				return err
			}
		}
		if err := c.player().SetPosition(time.Duration(s.seconds * float64(time.Second))); err != nil {
			return err
		}
	}
	return c.Game.Update()
}

func (c *gifCapture) Draw(screen *ebiten.Image) {
	c.Game.Draw(screen)
	if c.shot == len(gifShots) {
		return
	}
	c.n++
	if c.n > gifWarmup && c.n%2 == 0 {
		img := image.NewRGBA(screen.Bounds())
		screen.ReadPixels(img.Pix)
		c.frames[c.shot] = append(c.frames[c.shot], img)
		if len(c.frames[c.shot]) == gifFrames {
			c.shot, c.n = c.shot+1, 0
		}
	}
}

func TestReadmeGIFs(t *testing.T) {
	if os.Getenv("S1LV3RWAREZ_GIFS") == "" {
		t.Skip("set S1LV3RWAREZ_GIFS=1 to regenerate docs/*.gif")
	}
	g, err := newGame()
	if err != nil {
		t.Fatal(err)
	}
	c := &gifCapture{Game: g, frames: make([][]*image.RGBA, len(gifShots))}
	ebiten.SetWindowSize(virtualWidth*2, virtualHeight*2)
	_ = ebiten.RunGame(c) // GLFW can report a teardown error after Termination on Windows.
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	for i, frames := range c.frames {
		if len(frames) != gifFrames {
			t.Fatalf("shot %d captured %d frames", i, len(frames))
		}
		if err := writeGIF(fmt.Sprintf("docs/part%d.gif", gifShots[i].part+1), frames); err != nil {
			t.Fatal(err)
		}
	}
}

// writeGIF builds a popularity palette (the clip's 256 most common colors)
// and maps every pixel to its nearest entry. Screen pixels are premultiplied
// over black, so alpha is ignored.
func writeGIF(path string, frames []*image.RGBA) error {
	counts := map[[3]uint8]int{}
	for _, f := range frames {
		for i := 0; i < len(f.Pix); i += 4 {
			counts[[3]uint8{f.Pix[i], f.Pix[i+1], f.Pix[i+2]}]++
		}
	}
	keys := make([][3]uint8, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return string(keys[i][:]) < string(keys[j][:])
	})
	pal := color.Palette{}
	for _, k := range keys[:min(256, len(keys))] {
		pal = append(pal, color.RGBA{k[0], k[1], k[2], 255})
	}

	out := &gif.GIF{}
	cache := map[[3]uint8]uint8{}
	for _, f := range frames {
		p := image.NewPaletted(f.Bounds(), pal)
		for i := 0; i < len(f.Pix); i += 4 {
			key := [3]uint8{f.Pix[i], f.Pix[i+1], f.Pix[i+2]}
			idx, ok := cache[key]
			if !ok {
				idx = uint8(pal.Index(color.RGBA{key[0], key[1], key[2], 255}))
				cache[key] = idx
			}
			p.Pix[i/4] = idx
		}
		out.Image = append(out.Image, p)
		out.Delay = append(out.Delay, 3)
	}
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(w, out); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
