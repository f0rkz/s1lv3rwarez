package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part II: "The Zip Disk Strikes Back". A plasma, a typed-in ASCII crew
// banner, a stalled InstallWiz copy dialog and a tracker-style chip tune.

const (
	sequelBPM  = 140.0
	sequelBars = 32
)

const sequelBeatSeconds = 60 / sequelBPM

const sequelScrollText = "   ... S1LV3RWAREZ II ... STILL INSTALLING ... f0rkz CODED IT, sp00nz REPACKED IT WITH INSTALLWIZ AND BURNED IT ONTO 100MB OF PURE ZIP DISK ... HANDED OUT BETWEEN CLASSES, NO REFUNDS ... IF THE CLICK OF DEATH GETS YOU, YOU NEVER SAW US ... GREETZ TO THE HALLWAY COURIERS AND EVERY KID WHO SAID JUST ONE MORE DISK ...   "

var crew = [2]struct{ name, role string }{
	{"f0rkz", "CODE + CHAOS"},
	{"sp00nz", "PIXELS + PROBLEMS"},
}

var installFiles = []string{"QUAKE2.PAK", "STARCRAFT.MPQ", "WINAMP.EXE", "UNREAL.UTX", "FALLOUT2.DAT", "README.NFO"}

// installStatus maps the song position to the fake installer: two bars per
// file, then two bars of "disk full" before the next zip disk.
func installStatus(beats float64) (file string, progress float64, diskFull bool) {
	step := int(beats/8) % (len(installFiles) + 1)
	if step == len(installFiles) {
		return "", 1, true
	}
	return installFiles[step], math.Mod(beats, 8) / 8, false
}

func (g *Game) sequelBeats() float64 {
	return g.music2.Position().Seconds() * sequelBPM / 60
}

func (g *Game) updateSequel() {
	beats := g.sequelBeats()
	bar, p := int(beats/4), frac(beats)
	if bar >= 8 && !(bar >= 24 && bar < 28) && int(beats)%2 == 0 && p < .1 {
		g.flash = 1 - p/.1
	} else {
		g.flash *= .85
	}
}

func (g *Game) drawSequel(screen *ebiten.Image) {
	beats := g.sequelBeats()
	g.drawPlasma(screen)
	g.drawCRT(screen) // Set() overwrites, so scanlines go under the foreground like Part I
	g.drawSequelTitle(screen)
	g.drawASCIIName(screen, beats)
	drawInstallWiz(screen, beats, g.frame)
	g.drawScroller(screen, sequelScrollText)
	drawHint(screen, "SPACE: PART I")
}

func drawHint(screen *ebiten.Image, s string) {
	drawText(screen, s, virtualWidth-textWidth(s, 1)-8, 189, 1, color.RGBA{90, 100, 125, 255})
}

func (g *Game) drawPlasma(screen *ebiten.Image) {
	const w, h = virtualWidth / 2, virtualHeight / 2
	t := float64(g.frame) * .03
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x), float64(y)
			v := math.Sin(fx*.07+t) + math.Sin(fy*.09-t*1.3) + math.Sin((fx+fy)*.05+t*.7) +
				math.Sin(math.Hypot(fx-w/2, fy-h/2)*.11-t*1.7)
			o := (y*w + x) * 4
			g.plasmaPixels[o] = uint8(28 + 26*math.Sin(v*1.6))
			g.plasmaPixels[o+1] = uint8(14 + 12*math.Sin(v*1.6+2.1))
			g.plasmaPixels[o+2] = uint8(52 + 40*math.Sin(v*1.6+4.2))
			g.plasmaPixels[o+3] = 255
		}
	}
	g.plasma.WritePixels(g.plasmaPixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	v := float32(1 + .5*g.flash)
	op.ColorScale.Scale(v, v, v, 1)
	screen.DrawImage(g.plasma, op)
}

func (g *Game) drawSequelTitle(screen *ebiten.Image) {
	const title, sub = "S1LV3RWAREZ II", "THE ZIP DISK STRIKES BACK"
	x := (virtualWidth - textWidth(title, 2)) / 2
	for i := 0; i < len(title); i++ {
		y := 6 + int(2*math.Sin(float64(g.frame)*.1+float64(i)*.5))
		s := uint8(190 + 65*math.Sin(float64(g.frame)*.05-float64(i)*.4))
		drawGlyph(screen, title[i], x+i*12+1, y+1, 2, color.Black)
		drawGlyph(screen, title[i], x+i*12, y, 2, color.RGBA{s, s, 255, 255})
	}
	drawText(screen, sub, (virtualWidth-textWidth(sub, 1))/2, 27, 1, color.RGBA{255, 205, 110, 255})
}

// drawASCIIName blows a 5x7 glyph up into ASCII art: every lit pixel becomes
// a character cell holding the letter itself, typed in like a slow BBS.
func (g *Game) drawASCIIName(screen *ebiten.Image, beats float64) {
	const cw, chh, top = 6, 8, 38
	m := crew[int(beats/8)%len(crew)]
	name := strings.ToUpper(m.name)
	reveal := int(math.Mod(beats, 8) * 90)
	x0 := (virtualWidth - (len(name)*6-1)*cw) / 2
	n := 0
	for row := 0; row < 7; row++ {
		wobble := int(3 * math.Sin(float64(g.frame)*.07+float64(row)*.7))
		f := .6 * g.flash
		lift := func(v int) uint8 { return uint8(float64(v) + (255-float64(v))*f) }
		c := color.RGBA{lift(110 + row*20), lift(230 - row*22), 255, 255}
		for i := 0; i < len(name); i++ {
			bits := glyphs[name[i]][row]
			for col := 0; col < 5; col++ {
				if bits[col] != '1' {
					continue
				}
				if n++; n > reveal {
					continue
				}
				x, y := x0+wobble+(i*6+col)*cw, top+row*chh
				drawGlyph(screen, name[i], x+1, y+1, 1, color.Black)
				drawGlyph(screen, name[i], x, y, 1, c)
			}
		}
	}
	drawText(screen, m.role, (virtualWidth-textWidth(m.role, 1))/2, 97, 1, color.RGBA{185, 196, 198, 255})
}

func drawInstallWiz(screen *ebiten.Image, beats float64, frame uint64) {
	const x, y, w, h = 48, 108, 224, 39
	grey, dark := color.RGBA{192, 192, 192, 255}, color.RGBA{128, 128, 128, 255}
	navy, white := color.RGBA{0, 0, 128, 255}, color.RGBA{255, 255, 255, 255}
	vector.DrawFilledRect(screen, x, y, w, h, grey, false)
	drawHLine(screen, x, x+w-1, y, white)
	drawVLine(screen, x, y, y+h-1, white)
	drawHLine(screen, x, x+w-1, y+h-1, color.Black)
	drawVLine(screen, x+w-1, y, y+h-1, color.Black)
	vector.DrawFilledRect(screen, x+2, y+2, w-4, 10, navy, false)
	drawText(screen, "INSTALLWIZ 2.0 - SETUP", x+5, y+4, 1, white)

	file, progress, full := installStatus(beats)
	if !full {
		drawText(screen, "COPYING: "+file, x+6, y+16, 1, color.Black)
	} else if (frame/15)%2 == 0 {
		drawText(screen, "DISK FULL! INSERT ZIP DISK 2", x+6, y+16, 1, color.RGBA{200, 0, 0, 255})
	}

	// Sunken Win9x progress well with chunky blocks.
	const bx, by, bw, bh = x + 6, y + 26, w - 12, 9
	drawHLine(screen, bx, bx+bw-1, by, dark)
	drawVLine(screen, bx, by, by+bh-1, dark)
	drawHLine(screen, bx, bx+bw-1, by+bh-1, white)
	drawVLine(screen, bx+bw-1, by, by+bh-1, white)
	for i := 0; i < int(progress*26); i++ {
		vector.DrawFilledRect(screen, float32(bx+2+i*8), by+2, 6, bh-4, navy, false)
	}
}

func square(phase, duty float64) float64 {
	if phase < duty {
		return 1
	}
	return -1
}

// synthSequel renders the Part II tune: a four-channel tracker-style chip
// song in A minor with 50 Hz arpeggios, a pulse bass, noise drums and an
// echoed square lead.
func synthSequel(sr int) []byte {
	frames := int(float64(sequelBars*4) * sequelBeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0x5b00f5)
	const twoPi = 2 * math.Pi
	roots := [4]int{45, 41, 48, 43}                                             // A F C G
	chords := [4][3]int{{69, 72, 76}, {65, 69, 72}, {67, 72, 76}, {67, 71, 74}} // Am F C G
	melody := [32]int{
		76, 0, 72, 76, 81, 0, 79, 76,
		77, 0, 76, 72, 69, 0, 72, 0,
		72, 0, 76, 79, 84, 0, 83, 79,
		79, 0, 74, 71, 74, 0, 79, 0,
	}
	echo := make([]float64, int(.75*sequelBeatSeconds*float64(sr)))
	var arpPh, bassPh, leadPh, arpLP float64
	introLen := 16 * sequelBeatSeconds
	for i := 0; i < frames; i++ {
		t := float64(i) / float64(sr)
		beat := t / sequelBeatSeconds
		bar := int(beat / 4)
		chord := bar % 4
		breakdown := bar >= 24 && bar < 28
		mixL, mixR := 0.0, 0.0

		// Ch1: 50 Hz tracker arpeggio with a pulse-width sweep; the filter
		// opens across the four-bar intro.
		arpPh = frac(arpPh + midiHz(chords[chord][int(t*50)%3])/float64(sr))
		arpLP += (.08 + .25*math.Min(1, t/introLen)) * (square(arpPh, .3+.18*math.Sin(t*.7)) - arpLP)
		mixL += arpLP * .1
		mixR += arpLP * .13

		// Ch2: octave-jumping pulse bass on eighths.
		if bar >= 4 {
			note := roots[chord]
			if int(beat*2)%2 == 1 {
				note += 12
			}
			bassPh = frac(bassPh + midiHz(note)/float64(sr))
			b := square(bassPh, .5) * math.Exp(-frac(beat*2)*3.5) * .17
			if breakdown {
				b *= .4
			}
			mixL += b
			mixR += b
		}

		// Ch3: kick on 1 and 3, snare on 2 and 4, offbeat sixteenth hats.
		d := frac(beat) * sequelBeatSeconds
		if bar >= 8 && !breakdown {
			if int(beat)%2 == 0 && d < .2 {
				k := math.Sin(twoPi*(50*d-(120.0/30)*(math.Exp(-d*30)-1))) * math.Exp(-d*12) * .6
				mixL += k
				mixR += k
			} else if int(beat)%2 == 1 && d < .15 {
				s := noiseSample(&noise)*math.Exp(-d*25)*.18 + math.Sin(twoPi*200*d)*math.Exp(-d*30)*.1
				mixL += s
				mixR += s
			}
		}
		if p16 := frac(beat * 4); bar >= 4 && int(beat*4)%2 == 1 && p16 < .15 {
			h := noiseSample(&noise) * math.Exp(-p16*60) * .04
			mixL += h * 1.2
			mixR += h * .8
		}

		// Ch4: square lead with delayed vibrato, fed into a dotted-eighth echo.
		lead := 0.0
		if bar >= 8 && !breakdown {
			p := frac(beat * 2)
			if n := melody[int(beat*2)%32]; n != 0 && p < .85 {
				vib := 1 + .007*math.Sin(twoPi*6*t)*math.Min(1, p*sequelBeatSeconds*3)
				leadPh = frac(leadPh + midiHz(n)*vib/float64(sr))
				lead = square(leadPh, .25) * math.Min(1, p*40) * math.Exp(-p*1.2) * .11
			}
		}
		e := echo[i%len(echo)]
		echo[i%len(echo)] = lead + e*.4
		mixL += lead + e*.25
		mixR += lead*.8 + e*.5

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
