package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Part VII: "DATACENTER". f0rkz and sp00nz on the datacenter floor in
// Atlanta, across the street from the Telx building: racking and stacking
// CDN nodes all day and fighting terrible KVM consoles to install random
// packages. A rack filling up one server per bar, a spinning wireframe globe
// of CDN points of presence, a laggy KVM session and a 124 BPM minimal
// techno tune built on server hum, disk clicks and POST beeps.

const (
	dcBPM  = 124.0
	dcBars = 32
)

const dcBeatSeconds = 60 / dcBPM

const dcScrollText = "   ... S1LV3RWAREZ VII: DATACENTER ... f0rkz AND sp00nz, ALL GROWN UP AND WORKING THE DATACENTER FLOOR IN ATLANTA ... RIGHT ACROSS THE STREET FROM THE TELX BUILDING, ONE OF THE OG ROOT DNS SPOTS BACK IN THE DAY (IIRC) ... ALL DAY GRINDING FOR CDNS: RACK, STACK, CABLE, REPEAT ... FIGHTING THE WORST KVM INTERFACES EVER MADE JUST TO INSTALL RANDOM PACKAGES ... JAVA UPDATE REQUIRED ... FUN GIG ... f0rkz + sp00nz ...   "

// One line per four bars, lined up with the song.
var dcEvents = [dcBars / 4]string{
	"ATLANTA, GA: NIGHT SHIFT",
	"ACROSS THE STREET FROM TELX",
	"OG ROOT DNS COUNTRY (IIRC)",
	"RACK. STACK. CABLE. REPEAT.",
	"CDN BUILD: 24 NODES BY FRIDAY",
	"KVM: CONNECTION LOST",
	"JAVA UPDATE REQUIRED",
	"FUN GIG. LIGHTS OUT.",
}

const kvmScript = "login: root\npassword:\n# yum install -y bind\n# apt-get install snmpd\nbash: apt-get: not found\n# yum install -y nginx\n# rpm -ivh cdn-agent.rpm\n# mkfs.xfs /dev/sdb1\n# vi /etc/resolv.conf\n# reboot"

// CDN points of presence on the globe: latitude, longitude, label.
var dcPOPs = []struct {
	lat, lon float64
	name     string
}{
	{33.7, -84.4, "ATL"}, {40.7, -74, "NYC"}, {34, -118.2, "LAX"}, {41.9, -87.6, "ORD"},
	{51.5, 0, "LON"}, {50.1, 8.7, "FRA"}, {35.7, 139.7, "TYO"}, {-33.9, 151.2, "SYD"},
	{-23.5, -46.6, "GRU"}, {1.3, 103.8, "SIN"},
}

// kvmClock is the KVM session's typing clock in beats. Every four bars the
// console freezes for three beats while it reconnects.
func kvmClock(beats float64) (clock float64, lagging bool) {
	cycle := math.Floor(beats / 16)
	m := beats - cycle*16
	return cycle*13 + math.Min(m, 10) + math.Max(0, m-13), m >= 10 && m < 13
}

func (g *Game) dcBeats() float64 {
	return g.music7.Position().Seconds() * dcBPM / 60
}

func (g *Game) drawDatacenter(screen *ebiten.Image) {
	beats := g.dcBeats()
	g.drawServerRoom(screen)
	g.drawCRT(screen)
	g.drawPartTitle(screen, "S1LV3RWAREZ VII", "DATACENTER", color.RGBA{120, 255, 160, 255}, color.RGBA{uint8(60 + 120*g.flash), 230, 140, 255})
	g.drawRack(screen, beats)
	g.drawGlobe(screen, 104, 92, 30, float64(g.frame)*.02)
	g.drawKVM(screen, beats)
	drawAnnouncement(screen, dcEvents[int(beats/16)%len(dcEvents)], beats, color.RGBA{140, 255, 170, 255})
	g.drawScroller(screen, dcScrollText)
	drawHint(screen, "SPACE: PART I")
}

// drawServerRoom is a cold aisle in the dark: rows of blinking status LEDs.
func (g *Game) drawServerRoom(screen *ebiten.Image) {
	screen.Fill(color.RGBA{4, 8, 14, 255})
	tick := int(g.frame / 8)
	for y := 2; y < virtualHeight; y += 5 {
		for x := 3; x < virtualWidth; x += 6 {
			h := uint32(x*73856093 ^ y*19349663 ^ tick*83492791)
			if h%19 == 0 {
				screen.Set(x, y, color.RGBA{30, 140, 60, 255})
			} else if h%61 == 0 {
				screen.Set(x, y, color.RGBA{170, 110, 20, 255})
			}
		}
	}
}

// drawRack fills a 24U rack from the bottom, one server per bar, each one
// sliding in from the aisle.
func (g *Game) drawRack(screen *ebiten.Image, beats float64) {
	const x, y, w, slots = 10, 44, 52, 24
	vector.DrawFilledRect(screen, x, y-2, w, slots*4+3, color.RGBA{16, 18, 24, 255}, false)
	drawVLine(screen, x+2, y-2, y+slots*4, color.RGBA{90, 96, 110, 255})
	drawVLine(screen, x+w-3, y-2, y+slots*4, color.RGBA{90, 96, 110, 255})
	racked := min(slots, int(beats/4))
	for s := 0; s <= racked && s < slots; s++ {
		off := 0
		if s == racked {
			off = int(60 * (1 - math.Min(1, frac(beats/4)*2)))
		}
		sy := float32(y + (slots-1-s)*4)
		sx := float32(x + 4 + off)
		vector.DrawFilledRect(screen, sx, sy, w-8, 3, color.RGBA{70, 76, 88, 255}, false)
		led := color.RGBA{20, 70, 30, 255}
		if (s*7+int(g.frame/5)*13)%5 != 0 {
			led = color.RGBA{60, 255, 100, 255}
		}
		vector.DrawFilledRect(screen, sx+2, sy+1, 2, 1, led, false)
		vector.DrawFilledRect(screen, sx+w-12, sy+1, 2, 1, color.RGBA{255, uint8(120 + 100*g.flash), 30, 255}, false)
	}
}

// drawGlobe spins a tilted wireframe globe with the CDN's POPs pinging on
// the kick; Atlanta gets a label.
func (g *Game) drawGlobe(screen *ebiten.Image, cx, cy int, r, angle float64) {
	const tilt = .35
	proj := func(lat, lon float64) (int, int, float64) {
		x := math.Cos(lat) * math.Sin(lon+angle)
		y := math.Sin(lat)
		z := math.Cos(lat) * math.Cos(lon+angle)
		y, z = y*math.Cos(tilt)-z*math.Sin(tilt), y*math.Sin(tilt)+z*math.Cos(tilt)
		return cx + int(x*r), cy - int(y*r), z
	}
	seg := func(lat0, lon0, lat1, lon1 float64) {
		x0, y0, z0 := proj(lat0, lon0)
		x1, y1, z1 := proj(lat1, lon1)
		b := clamp(.5+.5*(z0+z1)/2, .15, 1)
		drawLine(screen, x0, y0, x1, y1, color.RGBA{uint8(40 * b), uint8(210 * b), uint8(190 * b), 255})
	}
	for k := 0; k < 8; k++ { // meridians
		lon := float64(k) * math.Pi / 4
		for i := 0; i < 12; i++ {
			seg(-math.Pi/2+float64(i)*math.Pi/12, lon, -math.Pi/2+float64(i+1)*math.Pi/12, lon)
		}
	}
	for _, lat := range []float64{-math.Pi / 3, -math.Pi / 6, 0, math.Pi / 6, math.Pi / 3} {
		for i := 0; i < 24; i++ {
			seg(lat, float64(i)*math.Pi/12, lat, float64(i+1)*math.Pi/12)
		}
	}
	for _, p := range dcPOPs {
		x, y, z := proj(p.lat*math.Pi/180, p.lon*math.Pi/180)
		if z <= 0 {
			continue
		}
		v := uint8(150 + 105*g.flash)
		vector.DrawFilledRect(screen, float32(x-1), float32(y-1), 3, 3, color.RGBA{v, v, 255, 255}, false)
		if p.name == "ATL" {
			drawText(screen, p.name, x+3, y-3, 1, color.RGBA{255, 220, 120, 255})
		}
	}
}

// drawKVM is a laggy Java KVM console typing a session. While it reconnects
// the picture tears and the keystrokes stop.
func (g *Game) drawKVM(screen *ebiten.Image, beats float64) {
	const x, y, w, h = 144, 42, 170, 100
	green := color.RGBA{110, 255, 140, 255}
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{0, 0, 0, 235}, false)
	vector.DrawFilledRect(screen, x, y, w, 9, color.RGBA{60, 70, 110, 255}, false)
	drawText(screen, "KVM 10.0.7.42 [JAVA]", x+3, y+1, 1, color.RGBA{220, 225, 240, 255})
	clock, lagging := kvmClock(beats)
	typed := int(clock*3) % (len(kvmScript) + 40) // pause on the reboot, then a new box
	lines := strings.Split(kvmScript[:min(typed, len(kvmScript))], "\n")
	if len(lines) > 9 {
		lines = lines[len(lines)-9:]
	}
	for r, l := range lines {
		off := 0
		if lagging && (r+int(g.frame/3))%3 == 0 {
			off = 4
		}
		if r == len(lines)-1 && g.frame/15%2 == 0 {
			l += "_"
		}
		drawText(screen, l, x+4+off, y+13+r*9, 1, green)
	}
	if lagging {
		const msg = "RECONNECTING..."
		mw := textWidth(msg, 1)
		vector.DrawFilledRect(screen, float32(x+(w-mw)/2-4), y+44, float32(mw+8), 13, color.RGBA{60, 70, 110, 255}, false)
		drawText(screen, msg, x+(w-mw)/2, y+47, 1, color.RGBA{255, 220, 120, 255})
	}
}

// synthDatacenter renders the Part VII tune: a 124 BPM minimal techno track
// in F# minor over fan noise and 60 Hz hum, with hard-drive seek clicks,
// PC-speaker POST beeps (a 1-long-2-short beep code in the breakdown), a
// rolling sidechained sub bass and an echoed square arpeggio.
func synthDatacenter(sr int) []byte {
	frames := int(float64(dcBars*4) * dcBeatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0xdc0a71)
	const twoPi = 2 * math.Pi
	sr64 := float64(sr)
	roots := [4]int{42, 38, 45, 40} // F#m D A E
	chords := [4][3]int{{66, 69, 73}, {62, 66, 69}, {69, 73, 76}, {64, 68, 71}}
	echo := make([]float64, int(.75*dcBeatSeconds*sr64))
	kickT, hatT, clickT, beepT, bassT, arpT := 9.0, 9.0, 9.0, 9.0, 9.0, 9.0
	var fanLP, hatLP, bassPh, arpPh, clickPan float64
	beepLen, arpNote, lastStep := 0.0, 0, -1
	for i := 0; i < frames; i++ {
		t := float64(i) / sr64
		beat := t / dcBeatSeconds
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
			if bar >= 4 && !breakdown {
				hatT = 0
			}
			if noiseSample(&noise) > .2 {
				clickT, clickPan = 0, noiseSample(&noise)
			}
			if band && (step == 3 || step == 11) {
				beepT, beepLen = 0, .05
			}
			if breakdown && step == 0 {
				beepT, beepLen = 0, .4
			} else if breakdown && (step == 8 || step == 10) {
				beepT, beepLen = 0, .1
			}
			if bar >= 4 && !breakdown {
				bassT = 0
			}
			if bar >= 16 && !breakdown {
				arpT, arpNote = 0, chords[chord][step%3]+12*(step/3%2)
			}
		}
		mixL, mixR := 0.0, 0.0
		pump := 1 - .8*math.Exp(-kickT*10)

		// Fan noise and 60 Hz mains hum, always on.
		n := noiseSample(&noise)
		fanLP += .03 * (n - fanLP)
		hum := .03*math.Sin(twoPi*60*t) + .02*math.Sin(twoPi*120*t) + .012*math.Sin(twoPi*180*t)
		mixL += fanLP*.8 + hum
		mixR += fanLP*.8 + hum

		if kickT < .4 {
			k := math.Sin(twoPi*(42*kickT-(140.0/28)*(math.Exp(-kickT*28)-1))) * math.Exp(-kickT*8)
			k = math.Tanh(k*2) * .58
			mixL += k
			mixR += k
		}
		hp := n - hatLP
		hatLP += .4 * (n - hatLP)
		if hatT < .06 {
			acc := .03
			if lastStep%4 == 2 {
				acc = .07
			}
			hh := hp * math.Exp(-hatT*60) * acc
			mixL += hh * .8
			mixR += hh
		}
		if clickT < .01 {
			c := (n*.6 + math.Sin(twoPi*3000*clickT)) * math.Exp(-clickT*700) * .09
			mixL += c * (1 - clickPan)
			mixR += c * (1 + clickPan)
		}
		if beepT < beepLen {
			b := square(frac(beepT*1000), .5) * .045
			mixL += b
			mixR += b
		}

		// Rolling sub: a 16th-note pluck on the root, octave on the offbeat.
		if bar >= 4 && !breakdown {
			note := roots[chord]
			if lastStep%4 == 2 {
				note += 12
			}
			bassPh = frac(bassPh + midiHz(note)/sr64)
			b := math.Tanh(math.Sin(twoPi*bassPh)*1.8) * math.Exp(-bassT*12) * .25 * pump
			mixL += b
			mixR += b
		}

		// Pad in the intro and breakdown.
		if bar < 8 || breakdown {
			for _, note := range chords[chord] {
				p := math.Sin(twoPi*t*midiHz(note-12)) + .4*math.Sin(twoPi*t*midiHz(note)*1.004)
				mixL += p * .022
				mixR += p * .022
			}
		}

		arp := 0.0
		if arpT < .12 {
			arpPh = frac(arpPh + midiHz(arpNote)/sr64)
			arp = square(arpPh, .5) * math.Exp(-arpT*30) * .05
		}
		e := echo[i%len(echo)]
		echo[i%len(echo)] = arp + e*.45
		mixL += arp + e*.25
		mixR += arp + e*.45

		if bar == 27 {
			r := frac(beat / 4)
			mixL += n * r * r * .1
			mixR += n * r * r * .1
		}

		putSample(pcm[i*4:i*4+2], math.Tanh(mixL*1.45)*.82)
		putSample(pcm[i*4+2:i*4+4], math.Tanh(mixR*1.45)*.82)
		dt := 1 / sr64
		kickT, hatT, clickT, beepT, bassT, arpT = kickT+dt, hatT+dt, clickT+dt, beepT+dt, bassT+dt, arpT+dt
	}
	crossfadeLoop(pcm, sr)
	return pcm
}
