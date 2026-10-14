package main

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part VI: "F7LANS". Older, and running the LAN parties ourselves: 200
// seats, servers with ~30,000 players a week, TF2 tourneys, rigs shipped by
// Intel, Tripwire beta keys (so everyone played Killing Floor) and about
// $10k raised for March of Dimes. A spinning wireframe F7 badge over a laser
// rig, a seat map filling up, a donation meter and a 128 BPM electro house tune.

const (
	f7BPM  = 128.0
	f7Bars = 32
)

const f7BeatSeconds = 60 / f7BPM

const f7ScrollText = "   ... S1LV3RWAREZ VI: F7LANS ... WE GREW UP AND STARTED RUNNING THE LAN PARTIES OURSELVES ... 200 SEATS AND ROUGHLY 30,000 PLAYERS A WEEK ON THE F7 SERVERS ... INTEL SHIPPED IN RIGS TO PLAY ON, TRIPWIRE SENT BETA KEYS, SO OF COURSE EVERYONE PLAYED KILLING FLOOR ... TF2 TOURNEYS UNTIL SUNRISE ... AND ABOUT $10,000 RAISED FOR MARCH OF DIMES ... f0rkz + sp00nz ... THANKS TO EVERYONE WHO CARRIED A TOWER THROUGH THE DOOR ...   "

// One line per four bars, lined up with the song.
var f7Events = [f7Bars / 4]string{
	"F7LANS: DOORS OPEN",
	"200 SEATS, 200 RIGS",
	"INTEL SHIPPED THE HARDWARE",
	"TRIPWIRE SENT BETA KEYS",
	"KILLING FLOOR ON EVERY SCREEN",
	"TF2 TOURNEY: GRAND FINALS",
	"$10,000 FOR MARCH OF DIMES",
	"30,000 PLAYERS A WEEK. GG",
}

// f7Badge is the lambda badge's ring with "F7" inside; struts sit on the ring.
var f7Badge = func() [][2]pt2 {
	segs := append([][2]pt2(nil), lambdaSegs[:24]...)
	const cell = .13
	text, w := glyphOutline("F7", cell)
	for _, s := range text {
		segs = append(segs, [2]pt2{{s[0].x - w/2, s[0].y - 3.5*cell}, {s[1].x - w/2, s[1].y - 3.5*cell}})
	}
	return segs
}()

var f7Struts = []pt2{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// f7Progress is how far the night has got: seats checked in, dollars raised
// for March of Dimes and players on the servers this week.
func f7Progress(beats float64) (seats, raised, players int) {
	return min(200, int(beats*200/48)), min(10000, int(beats*200/96)*50), min(30000, int(beats*30000/112))
}

func commas(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func (g *Game) f7Beats() float64 {
	return g.music6.Position().Seconds() * f7BPM / 60
}

// pulseOnKick flashes on the four-on-the-floor kick shared with Part V.
func (g *Game) pulseOnKick(beats float64) {
	if p := frac(beats * 4); qKick(int(beats/4), int(beats*4)%16) && p < .5 {
		g.flash = 1 - p/.5
	} else {
		g.flash *= .85
	}
}

func (g *Game) drawF7(screen *ebiten.Image) {
	beats := g.f7Beats()
	g.drawLasers(screen)
	g.drawCRT(screen)
	g.drawPartTitle(screen, "S1LV3RWAREZ VI", "F7LANS", color.RGBA{120, 220, 255, 255}, color.RGBA{uint8(150 + 100*g.flash), 90, 255, 255})
	drawWire(screen, f7Badge, f7Struts, 74, 94, 40*(1+.05*g.flash), float64(g.frame)*.03, color.RGBA{60, 210, 255, 255})
	seats, raised, players := f7Progress(beats)
	drawSeatMap(screen, seats)
	drawF7Meters(screen, raised, players)
	drawAnnouncement(screen, f7Events[int(beats/16)%len(f7Events)], beats, color.RGBA{120, 230, 255, 255})
	g.drawScroller(screen, f7ScrollText)
	drawHint(screen, "SPACE: PART I")
}

// drawLasers is a dark venue with two banks of sweeping lasers.
func (g *Game) drawLasers(screen *ebiten.Image) {
	screen.Fill(color.RGBA{8, 4, 18, 255})
	t := float64(g.frame) * .02
	for i := 0; i < 6; i++ {
		a := math.Pi/2 + .9*math.Sin(t+float64(i)*1.1)
		ox, c := 0, color.RGBA{uint8(90 + 100*g.flash), 30, 160, 255}
		if i%2 == 1 {
			ox, c = virtualWidth-1, color.RGBA{20, uint8(100 + 100*g.flash), 170, 255}
		}
		drawLine(screen, ox, 0, ox+int(400*math.Cos(a)), int(400*math.Sin(a)), c)
	}
}

func (g *Game) drawPartTitle(screen *ebiten.Image, sub, title string, subC, titleC color.RGBA) {
	drawText(screen, sub, (virtualWidth-textWidth(sub, 1))/2, 4, 1, subC)
	x := (virtualWidth - textWidth(title, 3)) / 2
	for i := 0; i < len(title); i++ {
		y := 14 + int(2*math.Sin(float64(g.frame)*.09+float64(i)*.6))
		drawGlyph(screen, title[i], x+i*18+1, y+1, 3, color.Black)
		drawGlyph(screen, title[i], x+i*18, y, 3, titleC)
	}
}

// drawAnnouncement centres msg above the scroller, blinking for its first beat.
func drawAnnouncement(screen *ebiten.Image, msg string, beats float64, c color.RGBA) {
	if since := math.Mod(beats, 16); since < 1 && int(since*8)%2 == 1 {
		return
	}
	x := (virtualWidth - textWidth(msg, 1)) / 2
	drawText(screen, msg, x+1, 152, 1, color.Black)
	drawText(screen, msg, x, 151, 1, c)
}

// drawSeatMap is the check-in grid: 20x10 seats, RED side and BLU side,
// lighting up in a scattered order as people arrive.
func drawSeatMap(screen *ebiten.Image, seats int) {
	const x, y, w, h = 150, 42, 162, 66
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{0, 0, 0, 170}, false)
	head := strconv.Itoa(seats) + "/200"
	drawText(screen, "CHECK-IN", x+4, y+3, 1, color.RGBA{120, 220, 255, 255})
	drawText(screen, head, x+w-4-textWidth(head, 1), y+3, 1, color.RGBA{230, 230, 240, 255})
	for s := 0; s < 200; s++ {
		col, row := s%20, s/20
		sx := x + 9 + col*7
		c := color.RGBA{34, 30, 48, 255}
		if col >= 10 {
			sx += 4 // the aisle
		}
		if (s*37)%200 < seats { // 37 is coprime to 200, so this visits every seat
			c = color.RGBA{210, 64, 52, 255}
			if col >= 10 {
				c = color.RGBA{80, 130, 230, 255}
			}
		}
		vector.DrawFilledRect(screen, float32(sx), float32(y+13+row*5), 6, 4, c, false)
	}
}

func drawF7Meters(screen *ebiten.Image, raised, players int) {
	const x, y, w = 150, 112, 162
	gold, white := color.RGBA{255, 200, 60, 255}, color.RGBA{230, 230, 240, 255}
	vector.DrawFilledRect(screen, x, y, w, 32, color.RGBA{0, 0, 0, 170}, false)
	amt := "$" + commas(raised)
	drawText(screen, "MARCH OF DIMES", x+4, y+3, 1, gold)
	drawText(screen, amt, x+w-4-textWidth(amt, 1), y+3, 1, white)
	vector.DrawFilledRect(screen, x+4, y+12, w-8, 4, color.RGBA{60, 50, 20, 255}, false)
	vector.DrawFilledRect(screen, x+4, y+12, float32(w-8)*float32(raised)/10000, 4, gold, false)
	n := commas(players)
	drawText(screen, "PLAYERS / WEEK", x+4, y+21, 1, color.RGBA{120, 220, 255, 255})
	drawText(screen, n, x+w-4-textWidth(n, 1), y+21, 1, white)
}

// synthF7 renders the Part VI tune: a 128 BPM electro house track in G minor
// with a pumping kick, flutter claps, offbeat open hats, a sidechained
// detuned bass wobbling on eighths, offbeat chord stabs and an echoed
// pulse-width lead.
func synthF7(sr int) []byte {
	frames := int(float64(f7Bars*4) * f7BeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0xf71a45)
	const twoPi = 2 * math.Pi
	sr64 := float64(sr)
	roots := [4]int{43, 39, 46, 41} // G Eb Bb F
	chords := [4][3]int{{67, 70, 74}, {63, 67, 70}, {70, 74, 77}, {65, 69, 72}}
	melody := [32]int{
		74, 0, 74, 77, 0, 74, 72, 70,
		75, 0, 75, 74, 0, 72, 70, 67,
		70, 0, 70, 74, 0, 77, 74, 72,
		72, 0, 74, 0, 77, 0, 74, 0,
	}
	echo := make([]float64, int(.75*f7BeatSeconds*sr64))
	kickT, clapT, hatT, stabT := 9.0, 9.0, 9.0, 9.0
	var bassA, bassB, bassLP, hatLP, leadPh, leadLP float64
	var stabPh [3]float64
	lastStep := -1
	for i := 0; i < frames; i++ {
		t := float64(i) / sr64
		beat := t / f7BeatSeconds
		bar := int(beat / 4)
		chord := bar % 4
		breakdown := bar >= 24 && bar < 28
		band := bar >= 8 && !breakdown

		if s16 := int(beat * 4); s16 != lastStep {
			lastStep = s16
			step := s16 % 16
			if qKick(bar, step) {
				kickT = 0
			}
			if band && (step == 4 || step == 12) {
				clapT = 0
			}
			if bar >= 2 && step%4 == 2 {
				hatT = 0
			}
			if band && (step == 3 || step == 6 || step == 11 || step == 14) {
				stabT = 0
			}
		}
		mixL, mixR := 0.0, 0.0
		pump := 1 - .85*math.Exp(-kickT*9)

		if kickT < .4 {
			k := math.Sin(twoPi*(50*kickT-(160.0/25)*(math.Exp(-kickT*25)-1))) * math.Exp(-kickT*7)
			k = math.Tanh(k*1.8) * .55
			mixL += k
			mixR += k
		}
		n := noiseSample(&noise)
		if clapT < .3 {
			env := math.Exp(-clapT * 18)
			if clapT < .03 {
				env = math.Exp(-math.Mod(clapT, .01) * 300) // three quick flutters
			}
			mixL += n * env * .13
			mixR += n * env * .13
		}
		hp := n - hatLP
		hatLP += .3 * (n - hatLP)
		if hatT < .2 {
			h := hp * math.Exp(-hatT*22) * .07
			mixL += h * .7
			mixR += h
		}

		// Soft pad for the intro and breakdown.
		if !band {
			for _, note := range chords[chord] {
				p := math.Sin(twoPi*t*midiHz(note-12)) + .5*math.Sin(twoPi*t*midiHz(note)*1.003)
				mixL += p * .025
				mixR += p * .025
			}
		}

		// Detuned bass with the filter wobbling on eighths, ducked by the kick.
		if band {
			f := midiHz(roots[chord])
			bassA = frac(bassA + f*1.007/sr64)
			bassB = frac(bassB + f*.993/sr64)
			cut := .02 + .12*(.5+.5*math.Sin(twoPi*beat*2))
			bassLP += cut * ((2*bassA - 1) + (2*bassB - 1) - bassLP)
			b := math.Tanh(bassLP*1.5) * .22 * pump
			mixL += b
			mixR += b
		}

		if band && stabT < .25 {
			env := math.Exp(-stabT * 14)
			for k, note := range chords[chord] {
				stabPh[k] = frac(stabPh[k] + midiHz(note)/sr64)
				s := (2*stabPh[k] - 1) * env * .045 * pump
				mixL += s * (1.2 - .2*float64(k))
				mixR += s * (.8 + .2*float64(k))
			}
		}

		lead := 0.0
		if bar >= 16 && !breakdown {
			p := frac(beat * 2)
			if note := melody[int(beat*2)%32]; note != 0 && p < .8 {
				leadPh = frac(leadPh + midiHz(note)/sr64)
				env := math.Min(1, p*40) * math.Exp(-p*1.2)
				leadLP += .25 * (square(leadPh, .25+.15*math.Sin(t*3)) - leadLP)
				lead = leadLP * env * .09
			}
		}
		e := echo[i%len(echo)]
		echo[i%len(echo)] = lead + e*.4
		mixL += lead + e*.4
		mixR += lead + e*.25

		if bar == 27 {
			r := frac(beat / 4)
			mixL += n * r * r * .1
			mixR += n * r * r * .1
		}

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
		dt := 1 / sr64
		kickT, clapT, hatT, stabT = kickT+dt, clapT+dt, hatT+dt, stabT+dt
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
