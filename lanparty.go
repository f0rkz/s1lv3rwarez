package main

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part III: "LAN Party". A spinning wireframe lambda, a server browser full
// of basement servers, a killfeed of crew grudges and a breakbeat tune.

const (
	lanBPM  = 132.0
	lanBars = 32
)

const lanBeatSeconds = 60 / lanBPM

const lanScrollText = "   ... S1LV3RWAREZ III: LAN PARTY ... TEN CRT MONITORS CARRIED DOWN THE BASEMENT STAIRS, ONE 10BASE-T HUB, ZERO SLEEP ... f0rkz HOSTED, sp00nz BROUGHT THE ZIP DISKS WITH THE CS BETA ON THEM ... HALF-LIFE, STARCRAFT, COUNTER-STRIKE, TEAM FORTRESS CLASSIC ... NO CAMPING IN THE VENTS ... WHOEVER KICKED THE POWER STRIP OWES EVERYONE PIZZA ... GG NO RE ...   "

var lanServers = [5]struct {
	name, mapName string
	ping          int
}{
	{"f0rkz bsmt", "de_dust", 12},
	{"sp00nz lan", "cs_office", 18},
	{"the hub", "2fort", 31},
	{"zipdisk tfc", "well", 44},
	{"bnet usw", "lost temple", 87},
}

var lanKills = []struct{ killer, weapon, victim string }{
	{"f0rkz", "AWP", "sp00nz"},
	{"sp00nz", "HEADSHOT", "f0rkz"},
	{"f0rkz", "CROWBAR", "sp00nz"},
	{"sp00nz", "ZERG RUSH", "f0rkz"},
	{"f0rkz", "SATCHEL", "sp00nz"},
	{"sp00nz", "CONC JUMP", "f0rkz"},
	{"f0rkz", "DEAGLE", "sp00nz"},
	{"sp00nz", "PHOTON CANNON", "f0rkz"},
}

// One event per four bars, lined up with the song's arrangement.
var lanEvents = [lanBars / 4]string{
	"f0rkz HAS JOINED THE GAME",
	"sp00nz HAS JOINED THE GAME",
	"FIRE IN THE HOLE!",
	"COUNTER-TERRORISTS WIN",
	"NUCLEAR LAUNCH DETECTED",
	"BLUE TEAM CAPTURED THE FLAG",
	"WHO UNPLUGGED THE HUB?",
	"TERRORISTS WIN",
}

type pt2 struct{ x, y float64 }

// lambdaSegs is a unit circle with a lambda inside; lambdaVerts get depth struts.
var lambdaVerts = []pt2{{-.38, -.62}, {-.12, -.62}, {.42, .58}, {.6, .58}, {.04, -.2}, {-.42, .58}}

var lambdaSegs = func() [][2]pt2 {
	var segs [][2]pt2
	for i := 0; i < 24; i++ {
		a0, a1 := float64(i)*math.Pi/12, float64(i+1)*math.Pi/12
		segs = append(segs, [2]pt2{{math.Cos(a0), math.Sin(a0)}, {math.Cos(a1), math.Sin(a1)}})
	}
	v := lambdaVerts
	return append(segs, [2]pt2{v[0], v[1]}, [2]pt2{v[1], v[2]}, [2]pt2{v[2], v[3]}, [2]pt2{v[4], v[5]})
}()

// lanKick is the break's kick pattern, shared by the synth and the visuals.
func lanKick(bar, step int) bool {
	return bar >= 4 && !(bar >= 24 && bar < 28) && (step == 0 || step == 6 || step == 10)
}

func (g *Game) lanBeats() float64 {
	return g.music3.Position().Seconds() * lanBPM / 60
}

func (g *Game) updateLAN() {
	beats := g.lanBeats()
	if p := frac(beats * 4); lanKick(int(beats/4), int(beats*4)%16) && p < .5 {
		g.flash = 1 - p/.5
	} else {
		g.flash *= .85
	}
}

func (g *Game) drawLAN(screen *ebiten.Image) {
	beats := g.lanBeats()
	screen.Fill(color.RGBA{4, 9, 6, 255})
	g.drawLANGrid(screen)
	g.drawCRT(screen)
	g.drawLANTitle(screen)
	g.drawLambda(screen)
	g.drawServerBrowser(screen, beats)
	drawKillfeed(screen, beats)
	drawLANEvent(screen, beats)
	g.drawScroller(screen, lanScrollText)
	drawHint(screen, "SPACE: PART IV")
}

// drawLANGrid is a faint scrolling network mesh, brighter on the kick.
func (g *Game) drawLANGrid(screen *ebiten.Image) {
	c := color.RGBA{10, uint8(34 + 30*g.flash), 18, 255}
	for x := 0; x < virtualWidth; x += 20 {
		drawVLine(screen, x, 0, virtualHeight-1, c)
	}
	for y := int(g.frame/2) % 20; y < virtualHeight; y += 20 {
		drawHLine(screen, 0, virtualWidth-1, y, c)
	}
}

func (g *Game) drawLANTitle(screen *ebiten.Image) {
	const sub, title = "S1LV3RWAREZ III", "LAN PARTY"
	drawText(screen, sub, (virtualWidth-textWidth(sub, 1))/2, 4, 1, color.RGBA{255, 176, 0, 255})
	x := (virtualWidth - textWidth(title, 3)) / 2
	lift := uint8(120 + 100*g.flash)
	for i := 0; i < len(title); i++ {
		y := 14 + int(2*math.Sin(float64(g.frame)*.09+float64(i)*.6))
		drawGlyph(screen, title[i], x+i*18+1, y+1, 3, color.Black)
		drawGlyph(screen, title[i], x+i*18, y, 3, color.RGBA{lift, 255, 140, 255})
	}
}

// drawLambda spins an extruded wireframe lambda badge around the Y axis.
func (g *Game) drawLambda(screen *ebiten.Image) {
	const cx, cy, r, depth = 70, 90, 40.0, .12
	a := float64(g.frame) * .03
	sinA, cosA := math.Sin(a), math.Cos(a)
	scale := r * (1 + .06*g.flash)
	proj := func(p pt2, z float64) (int, int, float64) {
		x := p.x*cosA - z*sinA
		zz := p.x*sinA + z*cosA
		f := 1 / (1 + zz*.15)
		return cx + int(x*scale*f), cy + int(p.y*scale*f), zz
	}
	shade := func(zz float64) color.RGBA {
		b := clamp(.6-zz*.45, .2, 1)
		return color.RGBA{uint8(255 * b), uint8(140 * b), uint8(20 * b), 255}
	}
	for _, z := range []float64{depth, -depth} { // back face first
		for _, s := range lambdaSegs {
			x0, y0, _ := proj(s[0], z)
			x1, y1, zz := proj(s[1], z)
			drawLine(screen, x0, y0, x1, y1, shade(zz))
		}
	}
	for _, v := range lambdaVerts {
		x0, y0, _ := proj(v, depth)
		x1, y1, zz := proj(v, -depth)
		drawLine(screen, x0, y0, x1, y1, shade(zz+.3))
	}
}

// drawServerBrowser is an olive VGUI-style server list with jittering pings.
func (g *Game) drawServerBrowser(screen *ebiten.Image, beats float64) {
	const x, y, w, h = 138, 42, 174, 60
	edge, text := color.RGBA{136, 145, 128, 255}, color.RGBA{216, 222, 211, 255}
	dim, amber := color.RGBA{160, 170, 150, 255}, color.RGBA{255, 176, 0, 255}
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{62, 70, 55, 255}, false)
	drawHLine(screen, x, x+w-1, y, edge)
	drawHLine(screen, x, x+w-1, y+h-1, edge)
	drawVLine(screen, x, y, y+h-1, edge)
	drawVLine(screen, x+w-1, y, y+h-1, edge)
	drawText(screen, "FIND SERVERS", x+4, y+3, 1, amber)
	drawText(screen, "PING", x+w-28, y+3, 1, amber)
	sel := int(beats) % len(lanServers)
	for i, s := range lanServers {
		ry := y + 13 + i*9
		if i == sel {
			vector.DrawFilledRect(screen, x+2, float32(ry-1), w-4, 9, color.RGBA{149, 136, 49, 255}, false)
		}
		ping := strconv.Itoa(s.ping + int(6*math.Sin(float64(g.frame)*.05+float64(i)*1.7)))
		drawText(screen, s.name, x+4, ry, 1, text)
		drawText(screen, s.mapName, x+74, ry, 1, dim)
		drawText(screen, ping, x+w-6-textWidth(ping, 1), ry, 1, text)
	}
}

// drawKillfeed shows the three latest frags, a new one every two beats.
func drawKillfeed(screen *ebiten.Image, beats float64) {
	const x, y = 138, 106
	vector.DrawFilledRect(screen, x, y-2, 174, 31, color.RGBA{0, 0, 0, 150}, false)
	newest := int(beats / 2)
	for row := 0; row < 3 && newest-row >= 0; row++ {
		k := lanKills[(newest-row)%len(lanKills)]
		fade := 1 - .3*float64(row)
		parts := [3]string{k.killer, "[" + k.weapon + "]", k.victim}
		cols := [3]color.RGBA{{120, 170, 255, 255}, {230, 230, 230, 255}, {255, 90, 80, 255}}
		px := x + 3
		for j, p := range parts {
			c := cols[j]
			c.R, c.G, c.B = uint8(float64(c.R)*fade), uint8(float64(c.G)*fade), uint8(float64(c.B)*fade)
			drawText(screen, p, px, y+row*10, 1, c)
			px += textWidth(p, 1) + 6
		}
	}
}

func drawLANEvent(screen *ebiten.Image, beats float64) {
	msg := lanEvents[int(beats/16)%len(lanEvents)]
	if since := math.Mod(beats, 16); since < 1 && int(since*8)%2 == 1 {
		return // blink on arrival
	}
	x := (virtualWidth - textWidth(msg, 1)) / 2
	drawText(screen, msg, x+1, 143, 1, color.Black)
	drawText(screen, msg, x, 142, 1, color.RGBA{255, 220, 120, 255})
}

// synthLAN renders the Part III tune: a 132 BPM big-beat breakbeat in D minor
// with a distorted reese bass, filtered chord stabs, a metallic FM hit and an
// echoed two-operator FM lead.
func synthLAN(sr int) []byte {
	frames := int(float64(lanBars*4) * lanBeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0x1a4b17)
	const twoPi = 2 * math.Pi
	sr64 := float64(sr)
	roots := [4]int{38, 34, 36, 33}                                            // D Bb C A
	stabs := [4][3]int{{62, 65, 69}, {58, 62, 65}, {60, 64, 67}, {57, 60, 64}} // Dm Bb C Am
	bassHits := [16]bool{0: true, 3: true, 6: true, 8: true, 10: true, 13: true}
	melody := [32]int{
		74, 0, 77, 74, 81, 0, 79, 77,
		77, 0, 74, 70, 74, 0, 0, 0,
		72, 0, 76, 79, 84, 0, 79, 76,
		76, 0, 72, 69, 72, 0, 76, 0,
	}
	echo := make([]float64, int(.75*lanBeatSeconds*sr64))
	kickT, snareT, bassT, stabT, hitT := 9.0, 9.0, 9.0, 9.0, 9.0
	var snareAmp, bassA, bassB, bassLP, stabLP, leadPh, modPh float64
	bassNote, lastStep := roots[0], -1
	for i := 0; i < frames; i++ {
		t := float64(i) / sr64
		beat := t / lanBeatSeconds
		bar := int(beat / 4)
		chord := bar % 4
		breakdown := bar >= 24 && bar < 28
		drums := bar >= 4 && !breakdown

		// Trigger this sixteenth's events once, then let envelopes run.
		if s16 := int(beat * 4); s16 != lastStep {
			lastStep = s16
			step := s16 % 16
			if lanKick(bar, step) {
				kickT = 0
			}
			if drums && (step == 4 || step == 12) {
				snareT, snareAmp = 0, 1
			} else if drums && (step == 7 || step == 15) {
				snareT, snareAmp = 0, .35 // ghost notes
			}
			if bassHits[step] {
				bassT, bassNote = 0, roots[chord]
				if step == 10 {
					bassNote += 12
				}
			}
			if step == 2 || step == 10 {
				stabT = 0
			}
			if step == 14 && bar%2 == 1 {
				hitT = 0
			}
		}
		mixL, mixR := 0.0, 0.0

		if kickT < .3 {
			k := math.Sin(twoPi*(48*kickT-(130.0/25)*(math.Exp(-kickT*25)-1))) * math.Exp(-kickT*9)
			k = math.Tanh(k*1.6) * .62
			mixL += k
			mixR += k
		}
		if snareT < .2 {
			s := (noiseSample(&noise)*math.Exp(-snareT*22)*.2 + math.Sin(twoPi*190*snareT)*math.Exp(-snareT*28)*.12) * snareAmp
			mixL += s
			mixR += s
		}
		if p16 := frac(beat * 4); bar >= 2 && p16 < .3 {
			level := .025
			if int(beat*4)%2 == 0 {
				level = .045
			}
			if breakdown {
				level *= .5
			}
			h := noiseSample(&noise) * math.Exp(-p16*40) * level
			mixL += h * .8
			mixR += h * 1.2
		}

		// Reese bass: two detuned saws, filtered, driven and ducked by the kick.
		if bar >= 8 && !breakdown {
			f := midiHz(bassNote)
			bassA = frac(bassA + f*1.006/sr64)
			bassB = frac(bassB + f*.994/sr64)
			bassLP += (.04 + .1*math.Exp(-bassT*12)) * ((2*bassA - 1) + (2*bassB - 1) - bassLP)
			b := math.Tanh(bassLP*2.5) * math.Exp(-bassT*3) * .2 * (1 - .5*math.Exp(-kickT*14))
			mixL += b
			mixR += b
		}

		// Offbeat chord stabs; the filter opens over the first eight bars.
		if stabT < .25 {
			s := 0.0
			for _, n := range stabs[chord] {
				s += 2*frac(t*midiHz(n)) - 1
			}
			stabLP += (.05 + .25*math.Min(1, beat/32)) * (s/3 - stabLP)
			v := stabLP * math.Exp(-stabT*14) * .16
			mixL += v * 1.1
			mixR += v * .8
		}

		// Metallic FM clang every other bar.
		if hitT < .6 {
			mod := math.Sin(twoPi*155*hitT) * 6 * math.Exp(-hitT*8)
			s := math.Sin(twoPi*110*hitT+mod) * math.Exp(-hitT*5) * .14
			mixL += s * .6
			mixR += s
		}

		// Two-operator FM lead into a dotted-eighth echo.
		lead := 0.0
		if bar >= 16 {
			p := frac(beat * 2)
			if n := melody[int(beat*2)%32]; n != 0 && p < .9 {
				f := midiHz(n)
				leadPh = frac(leadPh + f/sr64)
				modPh = frac(modPh + 2*f/sr64)
				env := math.Min(1, p*30) * math.Exp(-p*1.8)
				lead = math.Sin(twoPi*leadPh+1.8*env*math.Sin(twoPi*modPh)) * env * .12
			}
		}
		e := echo[i%len(echo)]
		echo[i%len(echo)] = lead + e*.4
		mixL += lead + e*.5
		mixR += lead + e*.25

		// Noise riser into the final section.
		if bar == 27 {
			r := frac(beat / 4)
			n := noiseSample(&noise) * r * r * .1
			mixL += n
			mixR += n
		}

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
		dt := 1 / sr64
		kickT, snareT, bassT, stabT, hitT = kickT+dt, snareT+dt, bassT+dt, stabT+dt, hitT+dt
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
