package common

import "testing"

func TestHeaderCodec(t *testing.T) {
	for _, s := range []string{"plain", "Euro \u20AC   \U0001F600",
		"  A   spacey,  \"string\"   ", "% is cool", ""} {
		got, err := DecodeHeaderValue(EncodeHeaderValue(s))
		if err != nil || got != s {
			t.Errorf("roundtrip %q: got %q err %v", s, got, err)
		}
	}

	if e := EncodeHeaderValue("Euro \u20AC   \U0001F600"); e !=
		"Euro %E2%82%AC   %F0%9F%98%80" {
		t.Errorf("bad encoding: %s", e)
	}

	got, err := DecodeHeaderValue(`"  A   spacey,  %22string%22  "`)
	if err != nil || got != `  A   spacey,  "string"  ` {
		t.Errorf("bad decode: %q %v", got, err)
	}

	for _, bad := range []string{"%", "%G1", "abc%2", `"unterminated`} {
		if _, err := DecodeHeaderValue(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}

	if EncodeHeaderName("org:team") != "org%3Ateam" {
		t.Errorf("bad name encoding")
	}
}
