package mimeparse

import (
	"reflect"
	"runtime"
	"testing"
)

func parsedEqual(test *testing.T, mime string, t string, st string, params map[string]string) {
	r, err := ParseMediaRange(mime)
	_, file, line, _ := runtime.Caller(1)
	if err != nil {
		test.Errorf("%s:%d Failed to parse", file, line, err)
	}
	if t != r.mtype {
		test.Errorf("%s:%d Failed to parse major type %s from %s, got %s\n", file, line, t, mime, r.mtype)
	}
	if st != r.subtype {
		test.Errorf("%s:%d Failed to parse minor type %s from %s, got %s\n", file, line, st, mime, r.subtype)
	}
	if !reflect.DeepEqual(params, r.params) {
		test.Errorf("%s:%d Failed to parse parameters, expected %v, got %v\n", file, line, params, r.params)
	}
}

func TestParseMimeType(t *testing.T) {
	parsedEqual(t, "Application/xhtml;q=0.5;vEr=1.2", "application", "xhtml", map[string]string{"q": "0.5", "ver": "1.2"})
}

func TestParseMediaRange(t *testing.T) {
	parsedEqual(t, "application/xml;q=1", "application", "xml", map[string]string{"q": "1"})
	parsedEqual(t, "application/xml;q=", "application", "xml", map[string]string{"q": "1"})
	parsedEqual(t, "application/xml;q", "application", "xml", map[string]string{"q": "1"})
	parsedEqual(t, "application/xml ; q=", "application", "xml", map[string]string{"q": "1"})
	parsedEqual(t, "application/xml ; q=1;b=other", "application", "xml", map[string]string{"q": "1", "b": "other"})
	parsedEqual(t, "application/xml ; q=2;b=other", "application", "xml", map[string]string{"q": "1", "b": "other"})
	// Java URLConnection class sends an Accept header that includes a single *
	parsedEqual(t, " *;q=.2", "*", "*", map[string]string{"q": ".2"})
}

func TestRFC2616Example(t *testing.T) {
	accept := "text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, * /*;q=0.5"
	cond := map[string]float{
		"text/html;level=1": 1.0,
		"text/html":         0.7,
		"text/plain":        0.3,
		"image/jpeg":        0.5,
		"text/html;level=2": 0.4,
		"text/html;level=3": 0.7,
	}
	for mime, q := range cond {
		if q != Quality(mime, accept) {
			t.Errorf("Failed to match %v at %f, got %f instead", mime, q, Quality(mime, accept))
		}
	}
}

func bestMatch(t *testing.T, supported []string, headers map[string]string) {
	for header, result := range headers {
		match := BestMatch(supported, header)
		if match != result {
			t.Errorf("BestMatch(%v, %v) == %s, not %s\n", supported, header, match, result)
		}
	}
}

func TestBestMatch(t *testing.T) {
	supported := []string{"application/xml", "application/xbel+xml"}
	headers := map[string]string{
		"application/xbel+xml":      "application/xbel+xml",
		"application/xbel+xml; q=1": "application/xbel+xml",
		"application/xml; q=1":      "application/xml",
		"application/*; q=1":        "application/xml",
		"*/*":                       "application/xml",
	}
	bestMatch(t, supported, headers)
}

func TestBestMatchDirect(t *testing.T) {
	supported := []string{"application/xbel+xml", "text/xml"}
	headers := map[string]string{
		"text/*;q=0.5,*/*; q=0.1":               "text/xml",
		"text/html,application/atom+xml; q=0.9": "",
	}
	bestMatch(t, supported, headers)
}

func TestBestMatchAjax(t *testing.T) {
	// Common AJAX scenario
	supported := []string{"application/json", "text/html"}
	headers := map[string]string{
		"application/json, text/javascript, */*": "application/json",
		"application/json, text/html;q=0.9":      "application/json",
	}
	bestMatch(t, supported, headers)
}

func TestSupportWildcards(t *testing.T) {
	supported := []string{"image/*", "application/xml"}
	headers := map[string]string{
		"image/png": "image/*",
		"image/*":   "image/*",
	}
	bestMatch(t, supported, headers)
}
