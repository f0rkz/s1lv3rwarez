package main

import (
	"encoding/binary"
	"math"
)

const (
	bpm       = 150.0
	musicBars = 48
)

const beatSeconds = 60 / bpm

// MusicState describes the musical position used by the visualizer.
type MusicState struct {
	BPM             float64
	Beat, Bar, Step int
	Kick, Snare     bool
	Section         string
	Energy          float64
}

func musicState(elapsed float64) MusicState {
	beat := int(math.Max(0, elapsed/beatSeconds))
	bar := beat / 4
	step := int(math.Max(0, elapsed/(beatSeconds/4))) % 16
	section, energy := "intro", 0.25
	switch {
	case bar >= 4 && bar < 8:
		section, energy = "groove", 0.48
	case bar >= 8 && bar < 16:
		section, energy = "drive", 0.7
	case bar >= 16 && bar < 24:
		section, energy = "main theme", 0.88
	case bar >= 24 && bar < 28:
		section, energy = "breakdown", 0.4
	case bar >= 28 && bar < 32:
		section, energy = "build", 0.72
	case bar >= 32:
		section, energy = "final drop", 1
	}
	phase := math.Mod(math.Max(0, elapsed), beatSeconds)
	return MusicState{BPM: bpm, Beat: beat, Bar: bar, Step: step, Kick: phase < .12 && (bar >= 4 || bar >= 32), Snare: phase >= beatSeconds/2 && phase < beatSeconds/2+.035 && bar >= 8, Section: section, Energy: energy}
}

func synthLoop(sr int) []byte {
	frames := int(float64(musicBars*4) * beatSeconds * float64(sr))
	pcm := make([]byte, frames*4)
	noise := uint32(0x1aceb00c)
	var bassLP, acidLP, padLP float64
	const twoPi = 2 * math.Pi
	progression := [4]int{53, 49, 56, 51} // Fm, Db, Ab, Eb
	// Two-bar original hook, repeated with a small variation in the final drop.
	melody := [16]int{0, 3, 7, 10, 7, 3, 5, 7, 0, 3, 7, 12, 10, 7, 5, 3}
	for i := 0; i < frames; i++ {
		t := float64(i) / float64(sr)
		beat := t / beatSeconds
		bar := int(beat / 4)
		beatInBar := math.Mod(beat, 4)
		step16 := int(math.Floor(math.Mod(beat*4, 16)))
		chord := progression[(bar/2)%4]
		state := musicState(t)
		mixL, mixR := 0.0, 0.0

		// Four-on-the-floor kick, tuned body sweep and a short click.
		kickPhase := math.Mod(beat, 1)
		if kickPhase < .34 && state.Kick {
			d := kickPhase * beatSeconds
			freq := 44 + 145*math.Exp(-d*28)
			body := math.Sin(twoPi*(freq*d-(145/28)*(math.Exp(-d*28)-1))) * math.Exp(-d*10)
			click := 0.0
			if d < .012 {
				click = noiseSample(&noise) * (1 - d/.012) * .18
			}
			k := math.Tanh((body*.8+click)*1.7) * .72
			mixL += k
			mixR += k
		}

		// Tight offbeat bass notes, tucked under each kick.
		bassGate := math.Mod(beat, 1) - .5
		if bassGate >= 0 && bassGate < .43 && bar >= 4 {
			note := chord - 12
			if step16%4 == 3 {
				note += 7
			}
			d := bassGate * beatSeconds
			f := midiHz(note)
			phase := frac(t * f)
			saw := 2*phase - 1
			pulse := math.Sin(twoPi*t*f*.5) * .25
			cut := 0.035 + 0.12*math.Exp(-d*20)
			bassLP += cut * (saw*.75 + pulse - bassLP)
			duck := 1 - .55*math.Exp(-math.Mod(beat, 1)*16)
			b := math.Tanh(bassLP*2.2) * math.Exp(-d*7) * duck * .33
			mixL += b
			mixR += b
		}

		// Hats and backbeat snare/clap from shaped, deterministic noise.
		if bar >= 5 {
			p8 := math.Mod(beat*2, 1)
			if p8 < .11 {
				h := noiseSample(&noise) * math.Exp(-p8*75) * .075
				mixL += h
				mixR += h
			}
			if p8 >= .5 && p8 < .58 && bar >= 8 {
				d := p8 - .5
				sn := noiseSample(&noise)*math.Exp(-d*45)*.13 + math.Sin(twoPi*185*d)*math.Exp(-d*35)*.08
				mixL += sn
				mixR += sn
			}
			// Open hat on alternate offbeats.
			if step16 == 6 || step16 == 14 {
				p := math.Mod(beat*4, 1)
				if p < .3 {
					h := noiseSample(&noise) * math.Exp(-p*12) * .04
					mixL += h
					mixR += h
				}
			}
		}

		// Resonant, moving acid line.
		if bar >= 8 && !(bar >= 24 && bar < 28) {
			acidNotes := [8]int{0, 12, 7, 3, 0, 7, 10, 7}
			p := math.Mod(beat*2, 1)
			dur := .78 * beatSeconds / 2
			if p*beatSeconds/2 < dur {
				n := chord + acidNotes[(step16/2)%8]
				f := midiHz(n)
				saw := 2*frac(t*f) - 1
				cut := .11 + .22*math.Exp(-p*7)
				acidLP += cut * (saw - acidLP)
				a := math.Exp(-p*5) * .13
				mixL += acidLP * a
				mixR += acidLP * a
			}
		}

		// Intro pad and breakdown sustain the harmonic progression.
		if bar < 8 || (bar >= 24 && bar < 32) {
			root := midiHz(chord - 12)
			pad := (math.Sin(twoPi*t*root) + math.Sin(twoPi*t*midiHz(chord+3))*.7 + math.Sin(twoPi*t*midiHz(chord+7))*.5) / 3
			padLP += .015 * (pad - padLP)
			level := .13
			if bar >= 24 && bar < 28 {
				level = .24
			}
			mixL += padLP * level
			mixR += padLP * level
		}

		// Supersaw-style five-voice hook, opened into the drops.
		if bar >= 16 && !(bar >= 24 && bar < 28) {
			active := bar >= 16 && bar < 24 || bar >= 32
			if active {
				p := math.Mod(beat*2, 1)
				dur := .88 * beatSeconds / 2
				if p*beatSeconds/2 < dur {
					note := chord + 12 + melody[(int(beat*2)%16)]
					if bar >= 32 && (bar%8 >= 4) {
						note += 12
					}
					f := midiHz(note)
					env := math.Min(1, p*18) * math.Exp(-p*1.65)
					voices := [5]float64{-0.012, -0.005, 0, 0.006, 0.013}
					lead := 0.0
					for _, detune := range voices {
						lead += polySaw(t * f * (1 + detune))
					}
					lead /= float64(len(voices))
					v := lead * env * .28
					mixL += v * .76
					mixR += v * 1.08
				}
			}
		}

		// Build riser is a filtered deterministic noise swell.
		if bar >= 28 && bar < 32 {
			p := (float64(bar-28) + beatInBar/4) / 4
			r := noiseSample(&noise) * p * p * .08
			mixL += r
			mixR += r
		}
		// Fixed soft limiter and conservative master gain.
		l := math.Tanh(mixL*1.45) * .82
		r := math.Tanh(mixR*1.45) * .82
		putSample(pcm[i*4:i*4+2], l)
		putSample(pcm[i*4+2:i*4+4], r)
	}
	// Crossfade a short boundary window to remove a discontinuity at loop wrap.
	cross := sr / 20
	for j := 0; j < cross; j++ {
		f := float64(j) / float64(cross)
		start, end := int16(binary.LittleEndian.Uint16(pcm[j*4:j*4+2])), int16(binary.LittleEndian.Uint16(pcm[(frames-cross+j)*4:(frames-cross+j)*4+2]))
		v := int16(float64(end)*(1-f) + float64(start)*f)
		binary.LittleEndian.PutUint16(pcm[(frames-cross+j)*4:(frames-cross+j)*4+2], uint16(v))
		startR, endR := int16(binary.LittleEndian.Uint16(pcm[j*4+2:j*4+4])), int16(binary.LittleEndian.Uint16(pcm[(frames-cross+j)*4+2:(frames-cross+j)*4+4]))
		v = int16(float64(endR)*(1-f) + float64(startR)*f)
		binary.LittleEndian.PutUint16(pcm[(frames-cross+j)*4+2:(frames-cross+j)*4+4], uint16(v))
	}
	return pcm
}

func putSample(dst []byte, s float64) {
	binary.LittleEndian.PutUint16(dst, uint16(int16(clamp(s*32767, -32768, 32767))))
}
func noiseSample(state *uint32) float64 {
	*state ^= *state << 13
	*state ^= *state >> 17
	*state ^= *state << 5
	return float64(int32(*state)) / float64(math.MaxInt32)
}
func midiHz(note int) float64 { return 440 * math.Pow(2, float64(note-69)/12) }
func frac(v float64) float64  { return v - math.Floor(v) }
func polySaw(phase float64) float64 {
	p := frac(phase)
	dt := 440 * (1 + 0.013) / 44100
	blep := 0.0
	if p < dt {
		x := p / dt
		blep = x + x - x*x - 1
	} else if p > 1-dt {
		x := (p - 1) / dt
		blep = x*x + x + x + 1
	}
	return 2*p - 1 - blep
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
