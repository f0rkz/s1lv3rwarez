package main

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part V: "QuakeCon". The annual August trip to Dallas. Why Dallas? The
// surface of the sun was already booked. A spinning wireframe Q emblem
// swaps 1 -> II -> III -> 4 each time it turns edge-on, over a shimmering
// skyline, a console full of obituaries and a dark industrial electro tune.

const (
	qBPM  = 110.0
	qBars = 32
)

const qBeatSeconds = 60 / qBPM

const qScrollText = "   ... S1LV3RWAREZ V: QUAKECON ... EVERY AUGUST, THE ANNUAL PILGRIMAGE TO DALLAS, TEXAS ... WHY DALLAS? BECAUSE THE SURFACE OF THE SUN WAS ALREADY BOOKED ... BRING YOUR OWN COMPUTER, BRING YOUR OWN SUNSCREEN ... ROCKET JUMPS, RAILGUN DUELS, NO SLEEP ... QUAKE, QUAKE II, QUAKE III ARENA, QUAKE 4 ... f0rkz + sp00nz ... SEE YOU NEXT YEAR ...   "

// One line per four bars, lined up with the song.
var qEvents = [qBars / 4]string{
	"DESTINATION: DALLAS, TEXAS",
	"WHY DALLAS?",
	"THE SURFACE OF THE SUN WAS ALREADY BOOKED",
	"BYOC: BRING YOUR OWN COMPUTER",
	"FIGHT!",
	"OUTSIDE 108F, INSIDE 300 CRTS",
	"sp00nz TAKES THE LEAD",
	"SEE YOU NEXT YEAR, DALLAS",
}

var qObits = []string{
	"sp00nz rides f0rkz's rocket",
	"f0rkz was railed by sp00nz",
	"sp00nz was nailed by f0rkz",
	"f0rkz turned into hot slag",
	"sp00nz was gibbed by f0rkz",
	"f0rkz was telefragged",
	"sp00nz tried to leave",
	"f0rkz was squished",
}

// Each version of the emblem: numeral and colour.
var qVersions = [4]struct {
	numeral string
	color   color.RGBA
}{
	{"1", color.RGBA{190, 120, 60, 255}},
	{"II", color.RGBA{120, 170, 90, 255}},
	{"III", color.RGBA{255, 100, 30, 255}},
	{"4", color.RGBA{160, 170, 190, 255}},
}

// qEmblem is the ring broken at the top with a nail through it; qNail
// vertices also get depth struts.
var qNail = []pt2{{-.16, -1.02}, {.16, -1.02}, {.06, -.8}, {.04, .75}, {0, 1}, {-.04, .75}, {-.06, -.8}}

var qEmblem = func() [][2]pt2 {
	var segs [][2]pt2
	arc := func(r float64) []pt2 {
		var pts []pt2
		for i := 0; i <= 20; i++ { // -60deg round the bottom to 240deg
			a := (-60 + float64(i)*15) * math.Pi / 180
			pts = append(pts, pt2{r * math.Cos(a), r * math.Sin(a)})
		}
		return pts
	}
	outer, inner := arc(1), arc(.78)
	for i := 0; i+1 < len(outer); i++ {
		segs = append(segs, [2]pt2{outer[i], outer[i+1]}, [2]pt2{inner[i], inner[i+1]})
	}
	segs = append(segs, [2]pt2{outer[0], inner[0]}, [2]pt2{outer[len(outer)-1], inner[len(inner)-1]})
	for i := range qNail {
		segs = append(segs, [2]pt2{qNail[i], qNail[(i+1)%len(qNail)]})
	}
	return segs
}()

// glyphOutline turns bitmap-font text into line art: the boundary edges of
// its lit cells, cell units, top-left at the origin.
func glyphOutline(s string, cell float64) (segs [][2]pt2, width float64) {
	lit := func(rows [7]string, r, c int) bool {
		return r >= 0 && r < 7 && c >= 0 && c < 5 && rows[r][c] == '1'
	}
	for i := 0; i < len(s); i++ {
		rows := glyphs[s[i]]
		ox := float64(i * 6)
		for r := 0; r < 7; r++ {
			for c := 0; c < 5; c++ {
				if !lit(rows, r, c) {
					continue
				}
				x0, y0 := (ox+float64(c))*cell, float64(r)*cell
				x1, y1 := x0+cell, y0+cell
				if !lit(rows, r-1, c) {
					segs = append(segs, [2]pt2{{x0, y0}, {x1, y0}})
				}
				if !lit(rows, r+1, c) {
					segs = append(segs, [2]pt2{{x0, y1}, {x1, y1}})
				}
				if !lit(rows, r, c-1) {
					segs = append(segs, [2]pt2{{x0, y0}, {x0, y1}})
				}
				if !lit(rows, r, c+1) {
					segs = append(segs, [2]pt2{{x1, y0}, {x1, y1}})
				}
			}
		}
	}
	return segs, float64(len(s)*6-1) * cell
}

// qBadge is the emblem plus numeral for one version, centred on the spin axis.
func qBadge(v int) (segs [][2]pt2, struts []pt2) {
	num, w := glyphOutline(qVersions[v].numeral, .14)
	const gap, numTop = 1.25, -7 * .14 / 2
	shift := (-1 + gap + w) / 2
	move := func(p pt2, dx, dy float64) pt2 { return pt2{p.x + dx - shift, p.y + dy} }
	for _, s := range qEmblem {
		segs = append(segs, [2]pt2{move(s[0], 0, 0), move(s[1], 0, 0)})
	}
	for _, s := range num {
		segs = append(segs, [2]pt2{move(s[0], gap, numTop), move(s[1], gap, numTop)})
	}
	for _, p := range qNail {
		struts = append(struts, move(p, 0, 0))
	}
	return segs, struts
}

var qBadges = func() (b [4]struct {
	segs   [][2]pt2
	struts []pt2
}) {
	for v := range b {
		b[v].segs, b[v].struts = qBadge(v)
	}
	return b
}()

// qSpin returns the badge angle and version: half a turn every two bars, and
// the version advances every four bars exactly when the badge is edge-on.
func qSpin(beats float64) (angle float64, version int) {
	return beats * math.Pi / 8, int((beats+12)/16) % len(qVersions)
}

func qKick(bar, step int) bool {
	return bar >= 4 && !(bar >= 24 && bar < 28) && step%4 == 0
}

// makeSkyline paints Dallas: towers with lit windows and Reunion Tower's ball.
func makeSkyline() *ebiten.Image {
	const w, h = virtualWidth, 46
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	dark := color.RGBA{34, 10, 8, 255}
	seed := uint32(0xda11a5)
	for x := 0; x < w; {
		bw := 8 + int(noiseSample(&seed)*6+6)
		bh := 12 + int(noiseSample(&seed)*14+14)
		for y := h - bh; y < h; y++ {
			for xx := x; xx < x+bw && xx < w; xx++ {
				img.SetRGBA(xx, y, dark)
				if xx%3 == 1 && y%4 == 2 && noiseSample(&seed) > .55 {
					img.SetRGBA(xx, y, color.RGBA{255, 200, 90, 255})
				}
			}
		}
		x += bw + 1
	}
	// Reunion Tower: a stalk and a ball of lights.
	const tx, ballY, ballR = 64, 6, 5
	for y := ballY; y < h; y++ {
		img.SetRGBA(tx, y, dark)
		img.SetRGBA(tx+1, y, dark)
	}
	for y := -ballR; y <= ballR; y++ {
		for x := -ballR; x <= ballR; x++ {
			if x*x+y*y <= ballR*ballR {
				c := dark
				if (x+y)%2 == 0 && (x*x+y*y)%3 == 0 {
					c = color.RGBA{255, 230, 160, 255}
				}
				img.SetRGBA(tx+x, ballY+y, c)
			}
		}
	}
	return ebiten.NewImageFromImage(img)
}

func (g *Game) qBeats() float64 {
	return g.music5.Position().Seconds() * qBPM / 60
}

func (g *Game) updateQuake() {
	beats := g.qBeats()
	if p := frac(beats * 4); qKick(int(beats/4), int(beats*4)%16) && p < .5 {
		g.flash = 1 - p/.5
	} else {
		g.flash *= .85
	}
}

func (g *Game) drawQuake(screen *ebiten.Image) {
	beats := g.qBeats()
	g.drawDallas(screen)
	g.drawCRT(screen)
	g.drawQuakeTitle(screen)
	angle, v := qSpin(beats)
	drawWire(screen, qBadges[v].segs, qBadges[v].struts, 80, 86, 30*(1+.05*g.flash), angle, qVersions[v].color)
	drawQuakeConsole(screen, beats)
	msg := qEvents[int(beats/16)%len(qEvents)]
	if since := math.Mod(beats, 16); !(since < 1 && int(since*8)%2 == 1) {
		x := (virtualWidth - textWidth(msg, 1)) / 2
		drawText(screen, msg, x+1, 152, 1, color.Black)
		drawText(screen, msg, x, 151, 1, color.RGBA{255, 200, 60, 255})
	}
	g.drawScroller(screen, qScrollText)
	drawHint(screen, "SPACE: PART VI")
}

// drawDallas is a scorched sky, a pulsing sun and a heat-shimmered skyline.
func (g *Game) drawDallas(screen *ebiten.Image) {
	const horizon = 148
	for y := 0; y < virtualHeight; y++ {
		c := color.RGBA{16, 6, 4, 255}
		if y < horizon {
			t := float64(y) / horizon
			c = color.RGBA{uint8(20 + 190*t*t), uint8(4 + 70*t*t*t), uint8(6 + 14*t), 255}
		}
		drawHLine(screen, 0, virtualWidth-1, y, c)
	}
	r := float32(16 + 1.5*math.Sin(float64(g.frame)*.05) + 2*g.flash)
	vector.DrawFilledCircle(screen, 282, 30, r+5, color.RGBA{60, 22, 6, 60}, false)
	vector.DrawFilledCircle(screen, 282, 30, r, color.RGBA{255, 200, 80, 255}, false)
	h := g.skyline.Bounds().Dy()
	for y := 0; y < h; y++ {
		row := g.skyline.SubImage(image.Rect(0, y, virtualWidth, y+1)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(.8*math.Sin(float64(g.frame)*.06+float64(y)*.2)), float64(horizon-h+y))
		screen.DrawImage(row, op)
	}
}

func (g *Game) drawQuakeTitle(screen *ebiten.Image) {
	const sub, title = "S1LV3RWAREZ V", "QUAKECON"
	drawText(screen, sub, (virtualWidth-textWidth(sub, 1))/2, 4, 1, color.RGBA{255, 220, 150, 255})
	x := (virtualWidth - textWidth(title, 3)) / 2
	for i := 0; i < len(title); i++ {
		y := 14 + int(2*math.Sin(float64(g.frame)*.09+float64(i)*.6))
		heat := uint8(80 + 120*g.flash)
		drawGlyph(screen, title[i], x+i*18+1, y+1, 3, color.Black)
		drawGlyph(screen, title[i], x+i*18, y, 3, color.RGBA{255, heat, 40, 255})
	}
}

// drawQuakeConsole is the drop-down console: year, temperature and obituaries.
func drawQuakeConsole(screen *ebiten.Image, beats float64) {
	const x, y, w, h = 146, 50, 170, 56
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{30, 20, 10, 200}, false)
	drawHLine(screen, x, x+w-1, y+h-1, color.RGBA{150, 100, 50, 255})
	seg := int(beats/16) % len(qEvents)
	temp := 104 + seg + int(1.5*math.Sin(beats*.3))
	head := "QUAKECON " + strconv.Itoa(1996+seg) + "  " + strconv.Itoa(temp) + "F"
	drawText(screen, head, x+3, y+3, 1, color.RGBA{255, 140, 40, 255})
	newest := int(beats / 2)
	for row := 0; row < 5 && newest-row >= 0; row++ {
		line := qObits[(newest-row)%len(qObits)]
		c := uint8(210 - row*25)
		drawText(screen, line, x+3, y+h-10-row*9, 1, color.RGBA{c, uint8(float64(c) * .8), uint8(float64(c) * .5), 255})
	}
}

// synthQuake renders the Part V tune: a 110 BPM dark industrial electro in
// C minor with an ambient drone and hot wind, metallic clanks, a pounding
// kick, a sixteenth-note EBM bass and a zapping detuned lead.
func synthQuake(sr int) []byte {
	frames := int(float64(qBars*4) * qBeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0x9a4ec0)
	const twoPi = 2 * math.Pi
	sr64 := float64(sr)
	roots := [4]int{36, 32, 34, 31} // C Ab Bb G
	melody := [32]int{
		72, 0, 75, 0, 79, 77, 75, 72,
		75, 0, 72, 0, 68, 0, 70, 72,
		70, 0, 74, 0, 77, 75, 74, 70,
		71, 0, 74, 0, 79, 0, 0, 0,
	}
	echo := make([]float64, int(.75*qBeatSeconds*sr64))
	kickT, snareT, bassT, clankT := 9.0, 9.0, 9.0, 9.0
	var bassPh, bassLP, windLP, windLP2, leadA, leadB, leadLP float64
	bassNote, clankHz, lastStep := roots[0], 220.0, -1
	for i := 0; i < frames; i++ {
		t := float64(i) / sr64
		beat := t / qBeatSeconds
		bar := int(beat / 4)
		chord := bar % 4
		breakdown := bar >= 24 && bar < 28

		if s16 := int(beat * 4); s16 != lastStep {
			lastStep = s16
			step := s16 % 16
			if qKick(bar, step) {
				kickT = 0
			}
			if bar >= 8 && !breakdown && (step == 4 || step == 12) {
				snareT = 0
			}
			bassT, bassNote = 0, roots[chord]
			if step%4 == 2 {
				bassNote += 12
			}
			if step == 6 {
				clankT, clankHz = 0, 220
			} else if step == 14 {
				clankT, clankHz = 0, 330
			}
		}
		mixL, mixR := 0.0, 0.0

		// Ambient drone and hot Texas wind, loudest when the band drops out.
		amb := .6
		if bar < 4 || breakdown {
			amb = 1
		}
		drone := (math.Sin(twoPi*t*midiHz(24)) + .6*math.Sin(twoPi*t*midiHz(31)) + .3*math.Sin(twoPi*t*midiHz(36)*(1+.002*math.Sin(t)))) * .07 * amb
		windLP += (.01 + .008*math.Sin(t*.4)) * (noiseSample(&noise) - windLP)
		windLP2 += .5 * (windLP - windLP2)
		mixL += drone + windLP*.25*amb
		mixR += drone + windLP2*.25*amb

		if kickT < .35 {
			k := math.Sin(twoPi*(45*kickT-(150.0/22)*(math.Exp(-kickT*22)-1))) * math.Exp(-kickT*8)
			k = math.Tanh(k*2.2) * .6
			mixL += k
			mixR += k
		}
		if snareT < .5 {
			s := noiseSample(&noise)*math.Exp(-snareT*12)*.16 + math.Sin(twoPi*170*snareT)*math.Exp(-snareT*30)*.1
			mixL += s
			mixR += s
		}
		if clankT < .4 {
			mod := math.Sin(twoPi*clankHz*1.41*clankT) * 5 * math.Exp(-clankT*10)
			c := math.Sin(twoPi*clankHz*clankT+mod) * math.Exp(-clankT*9) * .09
			mixL += c * 1.2
			mixR += c * .6
		}

		// EBM bass: sixteenth saws with an octave bounce and a snappy filter.
		if bar >= 8 || (breakdown && bar >= 26) {
			bassPh = frac(bassPh + midiHz(bassNote)/sr64)
			bassLP += (.03 + .15*math.Exp(-bassT*25)) * (2*bassPh - 1 - bassLP)
			b := math.Tanh(bassLP*2) * math.Exp(-bassT*9) * .24 * (1 - .5*math.Exp(-kickT*14))
			if breakdown {
				b *= .5
			}
			mixL += b
			mixR += b
		}

		// Detuned saw lead with a pitch zap on each note, into the echo.
		lead := 0.0
		if bar >= 16 && !breakdown {
			p := frac(beat * 2)
			if n := melody[int(beat*2)%32]; n != 0 && p < .85 {
				d := p * qBeatSeconds / 2
				f := midiHz(n) * (1 + .12*math.Exp(-d*90))
				leadA = frac(leadA + f*1.004/sr64)
				leadB = frac(leadB + f*.996/sr64)
				env := math.Min(1, p*40) * math.Exp(-p*1.5)
				leadLP += (.1 + .2*env) * ((2*leadA - 1) + (2*leadB - 1) - leadLP)
				lead = leadLP * env * .07
			}
		}
		e := echo[i%len(echo)]
		echo[i%len(echo)] = lead + e*.45
		mixL += lead + e*.3
		mixR += lead + e*.45

		if bar == 27 {
			r := frac(beat / 4)
			n := noiseSample(&noise) * r * r * .1
			mixL += n
			mixR += n
		}

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
		dt := 1 / sr64
		kickT, snareT, bassT, clankT = kickT+dt, snareT+dt, bassT+dt, clankT+dt
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
