package main

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part IV: "Web Club". Sixth grade, a lab of donated IBM PCs, homepages,
// then Red Alert, then a teacher's nuke in front of an executive's campus
// tour. A real-time-strategy battlefield, sidebar and EVA-style story feed
// over an original industrial march.

const (
	raBPM  = 120.0
	raBars = 32
)

const raBeatSeconds = 60 / raBPM

const raScrollText = "   ... S1LV3RWAREZ IV: WEB CLUB ... 6TH GRADE, A COMPUTER LAB FULL OF BRAND NEW IBM PCS FROM A BIG DONATION ... WE WERE SUPPOSED TO BUILD HOMEPAGES (UNDER CONSTRUCTION GIFS INCLUDED) ... THEN WE FOUND RED ALERT, AND SO DID THE TEACHERS ... HOMEPAGES DONE, NETWORK UP, TANK RUSH ... JUST AS AN IBM EXEC WALKS THROUGH ON A CAMPUS TOUR, OUR TEACHER JUMPS UP, YELLS THAT HE NUKED US ALL, AND CHEERS ... NO MORE RED ALERT AFTER THAT ... WORTH IT ... f0rkz + sp00nz ...   "

// One line per four bars: the story, lined up with the song.
var raEvents = [raBars / 4]string{
	"MISSION: WEB CLUB, GRADE 6",
	"NEW IBM PCS ACQUIRED",
	"CONSTRUCTION COMPLETE: HOMEPAGE.HTML",
	"TEACHERS HAVE DISCOVERED RED ALERT",
	"WARNING: EXECUTIVE TOUR APPROACHING",
	"ATOM BOMB LAUNCHED",
	"TEACHER: I NUKED YOU ALL! WOOO!",
	"MISSION FAILED: NO MORE RED ALERT",
}

const (
	fieldX, fieldY, fieldW, fieldH = 4, 38, 232, 108
	nukeX, nukeY                   = 55.0, 52.0 // the kids' base, in field coords
	missileBeat, impactBeat        = 88.0, 96.0
)

// raKick is the march's kick pattern, shared by the synth and the visuals.
func raKick(bar, step int) bool {
	return bar >= 4 && !(bar >= 24 && bar < 28) && (step == 0 || step == 3 || step == 8 || step == 10)
}

// nukeFlash is 1 at impact and fades over a bar.
func nukeFlash(beats float64) float64 {
	if beats < impactBeat || beats >= impactBeat+4 {
		return 0
	}
	return 1 - (beats-impactBeat)/4
}

// makeRAField paints the static terrain: grass, sand, water and rock tiles.
func makeRAField() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, fieldW, fieldH))
	for ty := 0; ty < fieldH/4; ty++ {
		for tx := 0; tx < fieldW/4; tx++ {
			n := math.Sin(float64(tx)*.31) + math.Sin(float64(ty)*.47+float64(tx)*.13) + .8*math.Sin(float64(tx+ty)*.21)
			c := color.RGBA{40, uint8(70 + (tx*7+ty*13)%3*10), 30, 255}
			switch {
			case n < -1.3:
				c = color.RGBA{24, 48, 104, 255}
			case n < -1.0:
				c = color.RGBA{150, 130, 80, 255}
			case n > 1.6:
				c = color.RGBA{90, 86, 80, 255}
			}
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					img.SetRGBA(tx*4+x, ty*4+y, c)
				}
			}
			img.SetRGBA(tx*4+1, ty*4+2, color.RGBA{c.R * 3 / 4, c.G * 3 / 4, c.B * 3 / 4, 255})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// raUnit places unit i of a side (0 = the kids, 1 = the teachers) in field coords.
func raUnit(i, side int, frame uint64) (float64, float64) {
	f := float64(frame)
	col, row := float64(i%3), float64(i/3)
	x := 20 + col*22 + float64(side)*130 + 10*math.Sin(f*.01*float64(1+i%4)+float64(i))
	y := 14 + row*24 + 6*math.Sin(f*.013+float64(i)*1.3)
	return x, y
}

func (g *Game) raBeats() float64 {
	return g.music4.Position().Seconds() * raBPM / 60
}

func (g *Game) updateRA() {
	beats := g.raBeats()
	if p := frac(beats * 4); raKick(int(beats/4), int(beats*4)%16) && p < .5 {
		g.flash = 1 - p/.5
	} else {
		g.flash *= .85
	}
}

func (g *Game) drawRA(screen *ebiten.Image) {
	beats := g.raBeats()
	screen.Fill(color.RGBA{8, 6, 6, 255})
	g.drawCRT(screen)
	g.drawRATitle(screen)
	g.drawBattlefield(screen, beats)
	g.drawSidebar(screen, beats)
	drawEVA(screen, beats)
	g.drawScroller(screen, raScrollText)
	drawHint(screen, "SPACE: PART I")
	if a := nukeFlash(beats); a > 0 {
		v := uint8(255 * a * a)
		vector.DrawFilledRect(screen, 0, 0, virtualWidth, virtualHeight, color.RGBA{v, v, v, v}, false)
	}
}

func (g *Game) drawRATitle(screen *ebiten.Image) {
	const sub, title = "S1LV3RWAREZ IV", "WEB CLUB"
	drawText(screen, sub, (virtualWidth-textWidth(sub, 1))/2, 4, 1, color.RGBA{255, 176, 0, 255})
	x := (virtualWidth - textWidth(title, 3)) / 2
	lift := uint8(30 + 120*g.flash)
	for i := 0; i < len(title); i++ {
		y := 14 + int(2*math.Sin(float64(g.frame)*.09+float64(i)*.6))
		drawGlyph(screen, title[i], x+i*18+1, y+1, 3, color.Black)
		drawGlyph(screen, title[i], x+i*18, y, 3, color.RGBA{230, lift, lift, 255})
	}
}

func (g *Game) drawBattlefield(screen *ebiten.Image, beats float64) {
	shake := nukeFlash(beats)
	ox := fieldX + 3*shake*math.Sin(float64(g.frame)*2.3)
	oy := fieldY + 3*shake*math.Cos(float64(g.frame)*1.9)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(g.raField, op)

	nuked := beats >= impactBeat
	sides := [2]color.RGBA{{60, 110, 230, 255}, {210, 40, 40, 255}}
	for side, c := range sides {
		for i := 0; i < 10; i++ {
			x, y := raUnit(i, side, g.frame)
			if nuked && math.Hypot(x-nukeX, y-nukeY) < 34 {
				continue
			}
			sx, sy := float32(ox+x), float32(oy+y)
			vector.DrawFilledRect(screen, sx-2, sy-2, 5, 5, color.Black, false)
			vector.DrawFilledRect(screen, sx-1, sy-1, 3, 3, c, false)
		}
	}
	// Teacher tank fire on the beat.
	if int(beats/4) >= 4 && frac(beats) < .15 {
		k := int(beats)
		ax, ay := raUnit((k*7)%10, 1, g.frame)
		bx, by := raUnit(k%10, 0, g.frame)
		drawLine(screen, int(ox+ax), int(oy+ay), int(ox+bx), int(oy+by), color.RGBA{255, 230, 90, 255})
	}

	// Silo, missile arc, then the blast.
	sx, sy := 200.0, 95.0
	vector.DrawFilledRect(screen, float32(ox+sx-4), float32(oy+sy-4), 9, 9, color.RGBA{90, 90, 90, 255}, false)
	vector.DrawFilledRect(screen, float32(ox+sx-1), float32(oy+sy-1), 3, 3, color.RGBA{220, 30, 30, 255}, false)
	if beats >= missileBeat && beats < impactBeat {
		p := (beats - missileBeat) / (impactBeat - missileBeat)
		for k := 8; k >= 0; k-- {
			q := math.Max(0, p-float64(k)*.02)
			mx := sx + (nukeX-sx)*q
			my := sy + (nukeY-sy)*q - 70*math.Sin(math.Pi*q)
			c := color.RGBA{140, 140, 140, 255}
			if k == 0 {
				c = color.RGBA{255, 255, 255, 255}
			}
			vector.DrawFilledRect(screen, float32(ox+mx-1), float32(oy+my-1), 2, 2, c, false)
		}
	}
	if nuked {
		cx, cy := float32(ox+nukeX), float32(oy+nukeY)
		if r := (beats - impactBeat) * 15; r < 60 {
			a := 1 - r/60
			vector.StrokeCircle(screen, cx, cy, float32(r), 2, color.RGBA{uint8(255 * a), uint8(160 * a), uint8(40 * a), uint8(255 * a)}, false)
		}
		vector.DrawFilledCircle(screen, cx, cy, 16, color.RGBA{30, 20, 15, 255}, false)
		for k := 0; k < 6; k++ {
			a := float64(k) * math.Pi / 3
			fr := float32(2 + 1.5*math.Sin(float64(g.frame)*.3+float64(k)))
			vector.DrawFilledCircle(screen, cx+float32(9*math.Cos(a)), cy+float32(9*math.Sin(a)), fr, color.RGBA{255, 120, 20, 255}, false)
		}
	}
}

// drawSidebar is the RTS command bar: credits, radar sweep and build queue.
func (g *Game) drawSidebar(screen *ebiten.Image, beats float64) {
	const x, y, w, h = 240, fieldY, 76, fieldH
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{44, 40, 40, 255}, false)
	drawHLine(screen, x, x+w-1, y, color.RGBA{110, 100, 100, 255})
	drawVLine(screen, x, y, y+h-1, color.RGBA{110, 100, 100, 255})
	drawText(screen, "$"+strconv.Itoa(1200+int(beats*23)), x+4, y+3, 1, color.RGBA{255, 210, 60, 255})

	const rx, ry, rw, rh = x + 3, y + 12, w - 6, 44
	vector.DrawFilledRect(screen, rx, ry, rw, rh, color.RGBA{0, 16, 0, 255}, false)
	for side, c := range [2]color.RGBA{{90, 140, 255, 255}, {255, 70, 70, 255}} {
		for i := 0; i < 10; i++ {
			ux, uy := raUnit(i, side, g.frame)
			if side == 0 && beats >= impactBeat && math.Hypot(ux-nukeX, uy-nukeY) < 34 {
				continue
			}
			screen.Set(rx+int(ux*rw/fieldW), ry+int(uy*rh/fieldH), c)
		}
	}
	a := float64(g.frame) * .05
	drawLine(screen, rx+rw/2, ry+rh/2, rx+rw/2+int(21*math.Cos(a)), ry+rh/2+int(21*math.Sin(a)), color.RGBA{60, 200, 80, 255})

	labels := [4]string{"HTML", "GIF", "MIDI", "NUKE"}
	building, progress := int(beats/4)%4, frac(beats/4)
	bar := int(beats / 4)
	for j, label := range labels {
		ix, iy := x+3+(j%2)*36, y+60+(j/2)*23
		vector.DrawFilledRect(screen, float32(ix), float32(iy), 33, 20, color.RGBA{86, 74, 62, 255}, false)
		if j == 3 && bar >= 20 && bar < 24 {
			label = ""
			if g.frame/10%2 == 0 {
				label = "READY"
			}
		}
		drawText(screen, label, ix+(33-textWidth(label, 1))/2, iy+7, 1, color.RGBA{230, 220, 200, 255})
		if j == building {
			vector.DrawFilledRect(screen, float32(ix), float32(iy), 33, float32(20*(1-progress)), color.RGBA{0, 0, 0, 170}, false)
		}
	}
}

func drawEVA(screen *ebiten.Image, beats float64) {
	msg := raEvents[int(beats/16)%len(raEvents)]
	if since := math.Mod(beats, 16); since < 1 && int(since*8)%2 == 1 {
		return // blink on arrival
	}
	x := (virtualWidth - textWidth(msg, 1)) / 2
	drawText(screen, msg, x+1, 152, 1, color.Black)
	drawText(screen, msg, x, 151, 1, color.RGBA{255, 80, 60, 255})
}

// synthRA renders the Part IV tune: a 120 BPM industrial march in E minor
// with distorted power-chord chugs, marching snare, an air-raid siren, a
// swelling brass line and one very large explosion.
func synthRA(sr int) []byte {
	frames := int(float64(raBars*4) * raBeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0x6b0b0b)
	const twoPi = 2 * math.Pi
	sr64 := float64(sr)
	riff := [2][16]int{
		{0, 0, -1, 0, 0, -1, 3, -1, 0, 0, -1, 0, 5, -1, 3, -1},
		{0, 0, -1, 0, 0, -1, 3, -1, 0, 0, -1, 0, 7, 6, 5, 3},
	}
	brass := [16]int{64, 64, 67, 66, 64, 0, 62, 59, 64, 64, 67, 71, 69, 67, 66, 0}
	march := [16]bool{0: true, 2: true, 3: true, 4: true, 6: true, 8: true, 10: true, 11: true, 12: true, 14: true}
	kickT, snareT, gtrT, boomT := 9.0, 9.0, 9.0, 9.0
	var snareAmp, gtrA, gtrB, gtrLP, sirenPh, brassPh, brassLP, boomLP float64
	gtrNote, gtrLong, lastStep := 40, false, -1
	for i := 0; i < frames; i++ {
		t := float64(i) / sr64
		beat := t / raBeatSeconds
		bar := int(beat / 4)
		breakdown := bar >= 24 && bar < 28
		heavy := bar >= 4 && !breakdown
		alarm := bar < 4 || (bar >= 20 && bar < 24)

		if s16 := int(beat * 4); s16 != lastStep {
			lastStep = s16
			step := s16 % 16
			if raKick(bar, step) {
				kickT = 0
			}
			if heavy && bar >= 8 && (step == 4 || step == 12) {
				snareT, snareAmp = 0, 1
			} else if alarm && march[step] {
				snareT, snareAmp = 0, .45 // marching roll
			}
			if heavy {
				off := riff[bar%2][step]
				if bar < 8 || alarm { // root chugs while it builds: quarters, then eighths under the siren
					off = -1
					if (bar < 8 && step%4 == 0) || (alarm && step%2 == 0) {
						off = 0
					}
				}
				if off >= 0 {
					gtrT, gtrNote, gtrLong = 0, 40+off, off != 0
				}
			}
			if s16 == 24*16 {
				boomT = 0
			}
		}
		mixL, mixR := 0.0, 0.0

		if kickT < .3 {
			k := math.Sin(twoPi*(52*kickT-(140.0/26)*(math.Exp(-kickT*26)-1))) * math.Exp(-kickT*10)
			k = math.Tanh(k*1.8) * .6
			mixL += k
			mixR += k
		}
		if snareT < .2 {
			s := (noiseSample(&noise)*math.Exp(-snareT*24)*.2 + math.Sin(twoPi*180*snareT)*math.Exp(-snareT*30)*.12) * snareAmp
			mixL += s
			mixR += s
		}
		if p8 := frac(beat * 2); heavy && bar >= 8 && p8 < .1 {
			h := noiseSample(&noise) * math.Exp(-p8*70) * .035
			mixL += h
			mixR += h
		}

		// Palm-muted power chords: root and fifth saws through hard clipping.
		if gtrT < 1.2 {
			f := midiHz(gtrNote)
			gtrA = frac(gtrA + f*1.003/sr64)
			gtrB = frac(gtrB + f*1.5*.997/sr64)
			gtrLP += .18 * (math.Tanh(((2*gtrA-1)+.8*(2*gtrB-1))*5) - gtrLP)
			decay := 18.0
			if gtrLong {
				decay = 5
			}
			v := gtrLP * math.Exp(-gtrT*decay) * .2
			mixL += v * 1.15
			mixR += v * .85
		}

		// Air-raid siren rising and falling over two bars.
		if alarm {
			sirenPh = frac(sirenPh + (380+320*(.5-.5*math.Cos(twoPi*beat/8)))/sr64)
			v := (math.Sin(twoPi*sirenPh) + .3*math.Sin(2*twoPi*sirenPh)) * .05
			mixL += v * .7
			mixR += v
		}

		// Low drone for the intro and the aftermath.
		if bar < 8 || breakdown {
			d := (math.Sin(twoPi*t*midiHz(28)) + .5*math.Sin(twoPi*t*midiHz(35))) * .08
			mixL += d
			mixR += d
		}

		// Swelling brass line; quieter and lonelier after the blast.
		if (bar >= 12 && bar < 20) || bar >= 24 {
			q := frac(beat)
			if n := brass[int(beat)%16]; n != 0 && q < .92 {
				brassPh = frac(brassPh + midiHz(n)*(1+.004*math.Sin(twoPi*5*t))/sr64)
				env := math.Min(1, q*8) * (1 - .3*q)
				brassLP += (.05 + .12*env) * (2*brassPh - 1 - brassLP)
				level := .14
				if breakdown {
					level = .08
				}
				v := brassLP * env * level
				mixL += v
				mixR += v
			}
		}

		// Noise riser into the impact, then the bomb itself.
		if bar == 23 {
			r := frac(beat / 4)
			n := noiseSample(&noise) * r * r * .1
			mixL += n
			mixR += n
		}
		if boomT < 3 {
			boomLP += .02 * (noiseSample(&noise) - boomLP)
			b := (boomLP*3 + math.Sin(twoPi*35*boomT)) * math.Exp(-boomT*1.3) * .45
			mixL += b
			mixR += b
		}

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
		dt := 1 / sr64
		kickT, snareT, gtrT, boomT = kickT+dt, snareT+dt, gtrT+dt, boomT+dt
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
