package main

import "testing"

func TestMatcher(t *testing.T) {
	patterns := []string{
		"example.com",
		"*.example.com",
		"*.example.*",
		"*.com",
		"**.example.com",
		"**.com",
	}
	cases := map[string]string{
		"example.com":         "example.com",
		"www.example.com":     "*.example.com",
		"www.example.net":     "*.example.*",
		"www.www.example.com": "**.example.com",
		"www.com":             "*.com",
		"www.www.com":         "**.com",
	}
	matcher := NewMatcher[string]()
	for _, pattern := range patterns {
		_ = matcher.Set(pattern, pattern)
	}
	for domain, expected := range cases {
		value, ok := matcher.Match(domain)
		if !ok {
			t.Errorf("Expected %s to match %s [FAILED]", domain, expected)
		} else if value != expected {
			t.Errorf("Expected %s to match %s, got %s [FAILED]", domain, expected, value)
		} else {
			t.Logf("Matched %s to %s", domain, expected)
		}
	}
}
