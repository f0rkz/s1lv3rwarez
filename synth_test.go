package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"testing"
)

func TestSynthLoopProducesStereoPCM(t *testing.T) {
	pcm := synthLoop(44100)
	if len(pcm) == 0 {
		t.Fatal("synthLoop returned no audio")
	}
	if len(pcm)%4 != 0 {
		t.Fatalf("PCM length %d is not aligned to stereo 16-bit frames", len(pcm))
	}
}

func TestSynthLoopDurationAndDeterminism(t *testing.T) {
	a, b := synthLoop(8000), synthLoop(8000)
	wantFrames := int(48 * 4 * beatSeconds * 8000)
	if len(a) != wantFrames*4 {
		t.Fatalf("duration frames = %d, want %d", len(a)/4, wantFrames)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("PCM differs at byte %d", i)
		}
	}
}

func TestMusicStateTiming(t *testing.T) {
	if bpm != 150 {
		t.Fatalf("BPM = %v, want 150", bpm)
	}
	s := musicState(8 * 4 * beatSeconds)
	if s.Bar != 8 || s.Beat != 32 || s.Section != "drive" || !s.Kick {
		t.Fatalf("state at bar 9 = %+v", s)
	}
	if got := musicState(24*4*beatSeconds + .001).Section; got != "breakdown" {
		t.Fatalf("section = %q", got)
	}
	if got := musicState(32*4*beatSeconds + .001).Section; got != "final drop" {
		t.Fatalf("section = %q", got)
	}
}

func TestSynthLoopSamplesAreFiniteAndBounded(t *testing.T) {
	pcm := synthLoop(8000)
	peak := 0.0
	for i := 0; i < len(pcm); i += 2 {
		v := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		f := float64(v) / 32768
		if math.IsNaN(f) || math.IsInf(f, 0) {
			t.Fatalf("invalid sample at %d", i/2)
		}
		peak = math.Max(peak, math.Abs(f))
	}
	if peak > .83 || peak < .1 {
		t.Fatalf("peak = %v, expected audible headroom", peak)
	}
	if len(pcm)%4 != 0 {
		t.Fatalf("unaligned stereo PCM length %d", len(pcm))
	}
}

func TestLoopBoundaryCrossfade(t *testing.T) {
	pcm := synthLoop(8000)
	firstL := int16(binary.LittleEndian.Uint16(pcm[:2]))
	lastL := int16(binary.LittleEndian.Uint16(pcm[len(pcm)-4 : len(pcm)-2]))
	firstR := int16(binary.LittleEndian.Uint16(pcm[2:4]))
	lastR := int16(binary.LittleEndian.Uint16(pcm[len(pcm)-2:]))
	if math.Abs(float64(firstL-lastL)) > 2500 || math.Abs(float64(firstR-lastR)) > 2500 {
		t.Fatalf("loop boundary jump too large: L %d, R %d", firstL-lastL, firstR-lastR)
	}
}

func TestMidiHzA4(t *testing.T) {
	if got := midiHz(69); got != 440 {
		t.Fatalf("midiHz(69) = %v, want 440", got)
	}
}

func TestLogoMaskUsesVisibleArtworkPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 1))
	img.Set(1, 0, color.RGBA{R: 180, G: 190, B: 200, A: 255})
	mask := makeLogoMask(img, 3, 1)
	if len(mask) != 3 || mask[0] || !mask[1] || mask[2] {
		t.Fatalf("mask = %v, want only the non-black center pixel", mask)
	}
}

func TestGlintFalloffIsSoftAndBounded(t *testing.T) {
	center := glintAlpha(0)
	shoulder := glintAlpha(4)
	outside := glintAlpha(9)
	if center == 0 || center > 112 || shoulder == 0 || shoulder >= center || outside != 0 {
		t.Fatalf("glint alpha center/shoulder/outside = %d/%d/%d", center, shoulder, outside)
	}
}

func TestStatusMarqueeCoversFullViewportCycle(t *testing.T) {
	const regionX, regionWidth, gap, speed = 7, 306, 42, 48.0
	messageWidth := textWidth(statusMarqueeText, 1)
	if messageWidth <= regionWidth {
		t.Fatalf("message width %d should require a marquee", messageWidth)
	}
	startX := marqueeX(0, messageWidth, regionX, regionWidth, gap, speed)
	if startX != regionX+regionWidth {
		t.Fatalf("marquee starts at %v, want right edge %d", startX, regionX+regionWidth)
	}
	leftEdgeTime := float64(regionWidth) / speed
	if got := marqueeX(leftEdgeTime, messageWidth, regionX, regionWidth, gap, speed); got != regionX {
		t.Fatalf("message reaches left edge at %v, want %d", got, regionX)
	}
	exitTime := float64(regionWidth+messageWidth) / speed
	if got := marqueeX(exitTime, messageWidth, regionX, regionWidth, gap, speed); got != float64(regionX-messageWidth) {
		t.Fatalf("message exits at %v, want %d", got, regionX-messageWidth)
	}
	cycleTime := float64(regionWidth+messageWidth+gap) / speed
	if got := marqueeX(cycleTime, messageWidth, regionX, regionWidth, gap, speed); math.Abs(got-float64(regionX+regionWidth)) > 1e-9 {
		t.Fatalf("marquee loop returns to %v, want %d", got, regionX+regionWidth)
	}
}

func TestSequelSongsAreDeterministicAndBounded(t *testing.T) {
	for _, song := range []struct {
		name        string
		synth       func(int) []byte
		bars        int
		beatSeconds float64
	}{
		{"part II", synthSequel, sequelBars, sequelBeatSeconds},
		{"part III", synthLAN, lanBars, lanBeatSeconds},
		{"part IV", synthRA, raBars, raBeatSeconds},
	} {
		a, b := song.synth(8000), song.synth(8000)
		if want := int(float64(song.bars*4)*song.beatSeconds*8000) * 4; len(a) != want {
			t.Fatalf("%s: len = %d, want %d", song.name, len(a), want)
		}
		peak := 0.0
		for i := 0; i < len(a); i += 2 {
			if a[i] != b[i] || a[i+1] != b[i+1] {
				t.Fatalf("%s: PCM differs at byte %d", song.name, i)
			}
			peak = math.Max(peak, math.Abs(float64(int16(binary.LittleEndian.Uint16(a[i:i+2])))/32768))
		}
		if peak > .83 || peak < .1 {
			t.Fatalf("%s: peak = %v, expected audible headroom", song.name, peak)
		}
	}
}

func TestInstallStatusStallsOnDiskFull(t *testing.T) {
	if f, p, full := installStatus(4); f != installFiles[0] || p != .5 || full {
		t.Fatalf("beat 4 = %q %v %v", f, p, full)
	}
	if _, p, full := installStatus(float64(len(installFiles)*8) + 1); !full || p != 1 {
		t.Fatalf("expected disk full stall, got %v %v", p, full)
	}
	if f, _, _ := installStatus(float64((len(installFiles) + 1) * 8)); f != installFiles[0] {
		t.Fatalf("installer should restart on disk 2, got %q", f)
	}
}
