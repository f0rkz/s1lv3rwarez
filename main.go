package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	virtualWidth  = 320
	virtualHeight = 200
	sampleRate    = 44100
)

//go:embed assets/s1lv3rwarez-banner.png
var assets embed.FS

type Star struct {
	x, y, z float64
	pz      float64
}

type Game struct {
	frame        uint64
	stars        []Star
	logo         *ebiten.Image
	logoMask     []bool
	glint        *ebiten.Image
	glintPixels  []byte
	audioCtx     *audio.Context
	music        *audio.Player
	musicStart   bool
	rng          *rand.Rand
	scrollX      float64
	marqueeTime  float64
	lastUpdate   time.Time
	clockStarted bool
	logoReveal   int
	flash        float64
	musicState   MusicState
	initialized  bool
}

func main() {
	g, err := newGame()
	if err != nil {
		panic(err)
	}

	ebiten.SetWindowTitle("S1LV3RWAREZ — The Lost Cracktro")
	ebiten.SetWindowSize(virtualWidth*3, virtualHeight*3)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}

func newGame() (*Game, error) {
	logo, logoMask, err := loadLogo()
	if err != nil {
		return nil, err
	}

	rng := rand.New(rand.NewSource(0x51_4c_56_33))
	g := &Game{
		logo:        logo,
		logoMask:    logoMask,
		glint:       ebiten.NewImage(logo.Bounds().Dx(), logo.Bounds().Dy()),
		glintPixels: make([]byte, logo.Bounds().Dx()*logo.Bounds().Dy()*4),
		rng:         rng,
		scrollX:     virtualWidth,
	}
	g.stars = make([]Star, 180)
	for i := range g.stars {
		g.resetStar(i, true)
	}

	g.audioCtx = audio.NewContext(sampleRate)
	pcm := synthLoop(sampleRate)
	loop := audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm)))
	g.music, err = g.audioCtx.NewPlayer(loop)
	if err != nil {
		return nil, fmt.Errorf("create music player: %w", err)
	}
	return g, nil
}

func loadLogo() (*ebiten.Image, []bool, error) {
	f, err := assets.Open("assets/s1lv3rwarez-banner.png")
	if err != nil {
		return nil, nil, fmt.Errorf("open embedded banner: %w", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, nil, fmt.Errorf("decode embedded banner: %w", err)
	}

	b := img.Bounds()
	cropH := int(float64(b.Dy()) * 0.78)
	cropped := img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+cropH))

	targetW := 304
	targetH := int(float64(cropH) * float64(targetW) / float64(b.Dx()))
	mask := makeLogoMask(cropped, targetW, targetH)
	out := ebiten.NewImage(targetW, targetH)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(targetW)/float64(b.Dx()), float64(targetH)/float64(cropH))
	op.Filter = ebiten.FilterNearest
	out.DrawImage(ebiten.NewImageFromImage(cropped), op)
	return out, mask, nil
}

// makeLogoMask derives membership from the resized source artwork. The banner
// has an opaque black background, so alpha alone cannot distinguish lettering.
func makeLogoMask(src image.Image, width, height int) []bool {
	b := src.Bounds()
	mask := make([]bool, width*height)
	for y := 0; y < height; y++ {
		sy := b.Min.Y + y*b.Dy()/height
		for x := 0; x < width; x++ {
			sx := b.Min.X + x*b.Dx()/width
			mask[y*width+x] = isLogoPixel(src.At(sx, sy))
		}
	}
	return mask
}

func isLogoPixel(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0x1800 || g > 0x1800 || b > 0x1800
}

func glintAlpha(distance float64) uint8 {
	if math.Abs(distance) > 8 {
		return 0
	}
	a := 112 * math.Exp(-(distance*distance)/(2*2.4*2.4))
	return uint8(a + .5)
}

func (g *Game) drawLogoGlint(screen *ebiten.Image, revealH int, pulse float64) {
	w := g.logo.Bounds().Dx()
	h := g.logo.Bounds().Dy()
	if revealH > h {
		revealH = h
	}
	if revealH <= 0 || len(g.glintPixels) != w*h*4 {
		return
	}
	// Cross the word in about 3.7 seconds, retaining the old sweep cadence.
	sweepX := math.Mod(float64(g.frame)*1.7, 380) - 38
	centerY := float64(h) / 2
	for y := 0; y < revealH; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			o := i * 4
			g.glintPixels[o], g.glintPixels[o+1], g.glintPixels[o+2], g.glintPixels[o+3] = 0, 0, 0, 0
			if !g.logoMask[i] {
				continue
			}
			d := float64(x) - (sweepX + .22*(float64(y)-centerY))
			a := glintAlpha(d)
			if a == 0 {
				continue
			}
			// White core with restrained icy-blue shoulders.
			shoulder := math.Min(1, math.Abs(d)/5)
			g.glintPixels[o] = uint8(255 - 50*shoulder)
			g.glintPixels[o+1] = uint8(255 - 22*shoulder)
			g.glintPixels[o+2] = 255
			g.glintPixels[o+3] = a
		}
	}
	g.glint.WritePixels(g.glintPixels)
	part := g.glint.SubImage(image.Rect(0, 0, w, revealH)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, 0)
	op.GeoM.Scale(pulse, 1)
	op.GeoM.Translate(float64(virtualWidth)/2, 12)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(part, op)
}

func (g *Game) resetStar(i int, randomDepth bool) {
	z := 1.0
	if randomDepth {
		z = 0.15 + g.rng.Float64()*0.85
	}
	g.stars[i] = Star{
		x:  (g.rng.Float64()*2 - 1) * 1.8,
		y:  (g.rng.Float64()*2 - 1) * 1.2,
		z:  z,
		pz: z,
	}
}

func (g *Game) Update() error {
	now := time.Now()
	if g.clockStarted {
		g.marqueeTime += now.Sub(g.lastUpdate).Seconds()
	} else {
		g.clockStarted = true
	}
	g.lastUpdate = now

	if !g.musicStart {
		g.music.Play()
		g.musicStart = true
	}

	g.frame++
	g.scrollX -= 0.85
	if g.scrollX < -float64(textWidth(scrollerText, 1)) {
		g.scrollX = virtualWidth
	}

	if g.logoReveal < g.logo.Bounds().Dy() {
		g.logoReveal += 2
	}

	for i := range g.stars {
		s := &g.stars[i]
		s.pz = s.z
		s.z -= 0.012
		if s.z <= 0.04 {
			g.resetStar(i, false)
		}
	}

	g.musicState = musicState(g.music.Position().Seconds())
	beat := g.beatPhase()
	if g.musicState.Kick && beat < 0.12 {
		g.flash = (1.0 - beat/0.12) * (0.5 + g.musicState.Energy*0.5)
	} else {
		g.flash *= 0.82
	}
	return nil
}

func (g *Game) beatPhase() float64 {
	seconds := g.music.Position().Seconds()
	beats := seconds * bpm / 60.0
	return beats - math.Floor(beats)
}

func (g *Game) beatIndex() int {
	seconds := g.music.Position().Seconds()
	return int(seconds * bpm / 60.0)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{2, 3, 6, 255})
	g.drawRasterBars(screen)
	g.drawStars(screen)
	g.drawCRT(screen)
	g.drawLogo(screen)
	g.drawCredits(screen)
	g.drawScroller(screen)
}

func (g *Game) drawStars(screen *ebiten.Image) {
	cx, cy := float64(virtualWidth)/2, float64(virtualHeight)/2
	for i := range g.stars {
		s := &g.stars[i]
		x := cx + (s.x/s.z)*86
		y := cy + (s.y/s.z)*68
		px := cx + (s.x/s.pz)*86
		py := cy + (s.y/s.pz)*68
		if x < 0 || x >= virtualWidth || y < 0 || y >= virtualHeight {
			g.resetStar(i, false)
			continue
		}
		brightness := uint8(clamp(60+(1-s.z)*195, 0, 255))
		c := color.RGBA{brightness, brightness, brightness, 255}
		drawLine(screen, int(px), int(py), int(x), int(y), c)
	}
}

func (g *Game) drawLogo(screen *ebiten.Image) {
	if g.logoReveal <= 0 {
		return
	}
	h := g.logoReveal
	if h > g.logo.Bounds().Dy() {
		h = g.logo.Bounds().Dy()
	}
	part := g.logo.SubImage(image.Rect(0, 0, g.logo.Bounds().Dx(), h)).(*ebiten.Image)
	pulse := 1.0 + 0.008*math.Sin(float64(g.frame)*0.08) + 0.006*g.flash
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(part.Bounds().Dx())/2, 0)
	op.GeoM.Scale(pulse, 1)
	op.GeoM.Translate(float64(virtualWidth)/2, 12)
	op.Filter = ebiten.FilterNearest
	v := 0.88 + 0.12*g.flash
	op.ColorScale.Scale(float32(v), float32(v), float32(v), 1)
	screen.DrawImage(part, op)

	g.drawLogoGlint(screen, h, pulse)
}

func (g *Game) drawRasterBars(screen *ebiten.Image) {
	baseY := 142
	phase := float64(g.frame) * 0.04
	palette := []color.RGBA{
		{22, 17, 70, 255}, {44, 25, 115, 255}, {85, 35, 140, 255},
		{22, 90, 120, 255}, {30, 150, 155, 255}, {85, 205, 180, 255},
	}
	for i := 0; i < 18; i++ {
		y := baseY + i
		idx := (i/3 + int(phase)) % len(palette)
		c := palette[idx]
		c.A = uint8(105 + int(40*math.Sin(phase+float64(i)*0.45)))
		drawHLine(screen, 0, virtualWidth-1, y, c)
	}
}

func (g *Game) drawCredits(screen *ebiten.Image) {
	beat := g.beatIndex()
	const marqueeX0, marqueeWidth, marqueeGap = 7, 306, 42
	messageWidth := textWidth(statusMarqueeText, 1)
	marqueePos := marqueeX(g.marqueeTime, messageWidth, marqueeX0, marqueeWidth, marqueeGap, 48)
	drawTextClipped(screen, statusMarqueeText, int(math.Floor(marqueePos)), 106, 1,
		color.RGBA{160, 177, 184, 255}, image.Rect(marqueeX0, 106, marqueeX0+marqueeWidth, 113))

	greets := []string{
		"GREETZ: RAZOR 1911 / FAIRLIGHT / THE SCENE",
		"CODE+CHAOS: F0RKZ   //   PIXELS+PROBLEMS: SP00NZ",
		"320X200 FOREVER   //   INSERT DISK 2 TO CONTINUE",
	}
	line := greets[(beat/4)%len(greets)]
	x := (virtualWidth - textWidth(line, 1)) / 2
	drawText(screen, line, x, 126, 1, color.RGBA{185, 196, 198, 255})
}

const statusMarqueeText = "THE LOST CRACKTRO // RECOVERED FROM ZIP DISK // 2026     F0RKZ + SP00NZ PRESENT // NO MONEY NO MASTERS JUST BITS     RECOVERED BY F0RKZ + SP00NZ"

func marqueeX(elapsed float64, messageWidth, regionX, regionWidth, gap int, speed float64) float64 {
	cycle := float64(regionWidth + messageWidth + gap)
	if cycle <= 0 || speed <= 0 {
		return float64(regionX + regionWidth)
	}
	distance := math.Mod(math.Max(0, elapsed)*speed, cycle)
	return float64(regionX+regionWidth) - distance
}

const scrollerText = "   ... S1LV3RWAREZ LIVES AGAIN ... BIG GREETZ TO EVERYONE WHO EVER TRADED A ZIP DISK IN A SCHOOL HALLWAY ... F0RKZ + SP00NZ ... SOFTWARE :: GAMES :: UTILITIES :: NO LIMITS ...   "

func (g *Game) drawScroller(screen *ebiten.Image) {
	x := g.scrollX
	for i := 0; i < len(scrollerText); i++ {
		ch := scrollerText[i]
		y := 167 + math.Sin((x+float64(i*6))*0.065+float64(g.frame)*0.06)*5
		shade := uint8(170 + 70*math.Sin(float64(i)*0.4+float64(g.frame)*0.03))
		drawGlyph(screen, ch, int(x)+i*6, int(y), 1, color.RGBA{shade, shade, 255, 255})
	}
}

func (g *Game) drawCRT(screen *ebiten.Image) {
	for y := 1; y < virtualHeight; y += 2 {
		drawHLine(screen, 0, virtualWidth-1, y, color.RGBA{0, 0, 0, 24})
	}
	// Coarse vignette bands. Cheap, deterministic, and suitably retro.
	for i := 0; i < 8; i++ {
		a := uint8(8 + i*2)
		drawHLine(screen, i, virtualWidth-1-i, i, color.RGBA{0, 0, 0, a})
		drawHLine(screen, i, virtualWidth-1-i, virtualHeight-1-i, color.RGBA{0, 0, 0, a})
	}
}

func (g *Game) Layout(_, _ int) (int, int) {
	return virtualWidth, virtualHeight
}
