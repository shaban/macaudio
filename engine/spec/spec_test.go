package spec

import (
	"testing"

	sess "github.com/shaban/macaudio/session"
)

func TestResolve_Defaults(t *testing.T) {
	got := Resolve(sess.AudioSpec{})
	if got.SampleRate != 48000 {
		t.Fatalf("rate: want 48000 got %v", got.SampleRate)
	}
	if got.BufferSize != 512 {
		t.Fatalf("buf: want 512 got %v", got.BufferSize)
	}
	if got.ChannelCount != 2 {
		t.Fatalf("ch: want 2 got %v", got.ChannelCount)
	}
	if got.BitDepth != 32 {
		t.Fatalf("bd: want 32 got %v", got.BitDepth)
	}
}

func TestResolve_Overrides(t *testing.T) {
	s := sess.AudioSpec{PreferredSampleRate: 96000, LatencyHint: sess.LatencyLow, BufferSize: 0, ChannelCount: 1, BitDepth: 24}
	got := Resolve(s)
	if got.SampleRate != 96000 {
		t.Fatalf("rate: want 96000 got %v", got.SampleRate)
	}
	if got.BufferSize != 256 {
		t.Fatalf("buf: want 256 got %v", got.BufferSize)
	}
	if got.ChannelCount != 1 {
		t.Fatalf("ch: want 1 got %v", got.ChannelCount)
	}
	if got.BitDepth != 24 {
		t.Fatalf("bd: want 24 got %v", got.BitDepth)
	}
}

func TestResolve_BufferHintBeatsLatency(t *testing.T) {
	s := sess.AudioSpec{LatencyHint: sess.LatencyHigh, BufferSize: 384}
	got := Resolve(s)
	if got.BufferSize != 384 {
		t.Fatalf("buf: want 384 got %v", got.BufferSize)
	}
}
