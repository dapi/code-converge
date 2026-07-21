package event

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestEmit(t *testing.T) {
	var out bytes.Buffer
	logger := Logger{Out: &out, Now: func() time.Time { return time.Date(2026, 7, 21, 10, 4, 5, 0, time.FixedZone("x", 3600)) }}
	if err := logger.Emit("stage_started", F("stage", "review"), F("cycle", "1")); err != nil {
		t.Fatal(err)
	}
	want := "ts=2026-07-21T09:04:05Z event=stage_started stage=review cycle=1\n"
	if out.String() != want {
		t.Fatalf("record = %q, want %q", out.String(), want)
	}
	for _, field := range []Field{F("Bad", "x"), F("ok", "has space"), F("ok", "a=b"), F("ok", "line\nbreak")} {
		if err := logger.Emit("test", field); err == nil {
			t.Errorf("invalid field accepted: %#v", field)
		}
	}
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("invalid records were written: %q", out.String())
	}
}
