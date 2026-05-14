package main

import "testing"

func TestAdvertiseHTTPAddress(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:8089":          "http://127.0.0.1:8089",
		":8089":                   "http://127.0.0.1:8089",
		"0.0.0.0:8089":            "http://127.0.0.1:8089",
		"[::]:8089":               "http://127.0.0.1:8089",
		"http://example:8089/":    "http://example:8089",
		" https://example:8443/ ": "https://example:8443",
	}
	for in, want := range cases {
		if got := advertiseHTTPAddress(in); got != want {
			t.Fatalf("advertiseHTTPAddress(%q)=%q want %q", in, got, want)
		}
	}
}

func TestLocalCapabilitiesWithAddress(t *testing.T) {
	caps := localCapabilities(":8089")
	if len(caps) != 1 {
		t.Fatalf("caps len=%d", len(caps))
	}
	meta := caps[0].Metadata
	if meta["address"] != "http://127.0.0.1:8089" || meta["transport"] != "http" {
		t.Fatalf("metadata=%v", meta)
	}
}
