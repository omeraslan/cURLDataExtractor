package main

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

// TestReprBytes tests the reprBytes function.
func TestReprBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty byte slice", []byte{}, "b''"},
		{"printable ASCII", []byte("hello"), "b'hello'"},
		{"single quote", []byte("it's"), "b'it\\'s'"},
		{"backslash", []byte("a\\b"), "b'a\\\\b'"},
		{"newline", []byte("line\nbreak"), "b'line\\nbreak'"},
		{"carriage return", []byte("line\rbreak"), "b'line\\rbreak'"},
		{"tab", []byte("line\tbreak"), "b'line\\tbreak'"},
		{"non-printable ASCII", []byte{0x00, 0x1f}, "b'\\x00\\x1f'"},
		{"mixed", []byte("a\nb'\\c\x01"), "b'a\\nb\\'\\\\c\\x01'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reprBytes(tt.input)
			if got != tt.expected {
				t.Errorf("reprBytes(%v) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestExtractDataRaw tests the extractDataRaw function.
func TestExtractDataRaw(t *testing.T) {
	tests := []struct {
		name        string
		curlCommand string
		expected    string
		expectError bool
	}{
		{
			name:        "valid command",
			curlCommand: "curl 'url' --data-raw $'content'",
			expected:    "content",
			expectError: false,
		},
		{
			name:        "multiline content",
			curlCommand: "curl 'url' --data-raw $'line1\nline2'",
			expected:    "line1\nline2",
			expectError: false,
		},
		{
			name:        "empty content",
			curlCommand: "curl 'url' --data-raw $''",
			expected:    "",
			expectError: false,
		},
		{
			name:        "no --data-raw",
			curlCommand: "curl 'url'",
			expected:    "",
			expectError: true,
		},
		{
			name:        "malformed - missing $",
			curlCommand: "curl 'url' --data-raw 'content'",
			expected:    "",
			expectError: true,
		},
		{
			name:        "malformed - missing quotes",
			curlCommand: "curl 'url' --data-raw $content",
			expected:    "",
			expectError: true,
		},
		{
			name:        "other args in command",
			curlCommand: "curl -X POST 'url' -H 'Content-Type: application/json' --data-raw $'{\"key\":\"value\"}' --compressed",
			expected:    "{\"key\":\"value\"}",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractDataRaw(tt.curlCommand)
			if tt.expectError {
				if err == nil {
					t.Errorf("extractDataRaw(%q) should have returned an error, but got nil", tt.curlCommand)
				}
			} else {
				if err != nil {
					t.Errorf("extractDataRaw(%q) returned an unexpected error: %v", tt.curlCommand, err)
				}
				if got != tt.expected {
					t.Errorf("extractDataRaw(%q) = %q; want %q", tt.curlCommand, got, tt.expected)
				}
			}
		})
	}
}

// TestDecodeRawData tests the decodeRawData function.
func TestDecodeRawData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []byte
		expectError bool
		errorMsg    string // Expected substring in error message
	}{
		{"empty string", "", []byte{}, false, ""},
		{"simple string", "hello", []byte("hello"), false, ""},
		{"newline", "\\n", []byte("\n"), false, ""},
		{"carriage return", "\\r", []byte("\r"), false, ""},
		{"tab", "\\t", []byte("\t"), false, ""},
		{"backslash", "\\\\", []byte("\\"), false, ""},
		{"single quote", "\\'", []byte("'"), false, ""},
		{"double quote", "\\\"", []byte("\""), false, ""},
		{"form feed", "\\f", []byte("\f"), false, ""},
		{"backspace", "\\b", []byte("\b"), false, ""},
		{"vertical tab", "\\v", []byte("\v"), false, ""},
		{"alert", "\\a", []byte("\a"), false, ""},
		{"valid hex", "\\x48\\x65", []byte("He"), false, ""},
		{"incomplete hex", "\\x4", nil, true, "incomplete hex escape"},
		{"invalid hex char", "\\x4G", nil, true, "invalid hex escape"},
		{"valid unicode (latin1)", "\\u0041", []byte("A"), false, ""},
		{"valid unicode (latin1 non-ascii)", "\\u00E4", []byte{0xe4}, false, ""}, // ä
		{"unicode outside latin1", "\\u0100", nil, true, "outside Latin-1 range"},
		{"incomplete unicode", "\\u004", nil, true, "incomplete unicode escape"},
		{"invalid unicode char", "\\u004G", nil, true, "invalid unicode escape"},
		{"valid U unicode (latin1)", "\\U00000061", []byte("a"), false, ""},
		{"U unicode outside latin1", "\\U00000100", nil, true, "outside Latin-1 range"},
		{"incomplete U unicode", "\\U0000006", nil, true, "incomplete unicode escape"},
		{"invalid U unicode char", "\\U0000006G", nil, true, "invalid unicode escape"},
		{"octal 1 digit", "\\0", []byte{0}, false, ""},
		{"octal 2 digits", "\\77", []byte{0x3f}, false, ""}, // '?'
		{"octal 3 digits", "\\101", []byte{'A'}, false, ""},
		{"octal 3 digits max val", "\\377", []byte{0xff}, false, ""},
		{"octal too large", "\\400", nil, true, "too large for a byte"},                    // This will be caught by val > 0xFF after parsing "400" as octal
		{"octal invalid char (like \\08)", "\\08", nil, true, "invalid octal escape \\08"}, // Corrected
		{"octal invalid char (like \\79)", "\\79", nil, true, "invalid octal escape \\79"}, // Added
		{"octal followed by non-digit", "\\0a", []byte{0x00, 'a'}, false, ""},              // \0 followed by literal 'a'
		{"unrecognized escape", "\\z", []byte("\\z"), false, ""},
		{"trailing backslash", "abc\\", nil, true, "trailing backslash"},
		{"literal char outside latin1", "H€llo", nil, true, "literal character U+20AC ('€') is outside Latin-1 range"}, // € is U+20AC
		{"valid literal char within latin1", "Hällo", []byte{0x48, 0xE4, 0x6C, 0x6C, 0x6F}, false, ""},                 // Corrected: ä is 0xE4 in Latin-1
		{"invalid utf8 sequence for literal", string([]byte{0x41, 0xff, 0x42}), nil, true, "invalid UTF-8 sequence"},   // 0xff is invalid standalone
		{"mixed escapes", "A\\nB\\x43D\\105F", []byte("A\nBCDEF"), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeRawData(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("decodeRawData(%q) should have returned an error, but got nil", tt.input)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("decodeRawData(%q) error = %v, want error containing %q", tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("decodeRawData(%q) returned an unexpected error: %v", tt.input, err)
				}
				if !bytes.Equal(got, tt.expected) {
					// For byte slices, it's often helpful to see hex output on mismatch
					t.Errorf("decodeRawData(%q) = %x (%s); want %x (%s)", tt.input, got, string(got), tt.expected, string(tt.expected))
				}
			}
		})
	}
}

// TestDecompressGzipData tests the decompressGzipData function.
func TestDecompressGzipData(t *testing.T) {
	// Helper function to create gzipped data
	gzipData := func(data string) []byte {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		if _, err := gz.Write([]byte(data)); err != nil {
			t.Fatalf("Failed to gzip data: %v", err)
		}
		if err := gz.Close(); err != nil {
			t.Fatalf("Failed to close gzip writer: %v", err)
		}
		return b.Bytes()
	}

	tests := []struct {
		name        string
		input       []byte
		expected    []byte
		expectError bool
		errorMsg    string
	}{
		{"valid gzipped data", gzipData("hello world"), []byte("hello world"), false, ""},
		{"empty gzipped data", gzipData(""), []byte(""), false, ""},
		{"corrupted gzip data", []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x01, 0x02}, nil, true, "failed to decompress data"}, // Corrupted
		{"non-gzipped data", []byte("just plain text"), nil, true, "failed to create gzip reader"},
		{"empty input slice", []byte{}, nil, true, "failed to create gzip reader"}, // gzip.NewReader expects a header
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decompressGzipData(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("decompressGzipData() for %s should have returned an error, but got nil", tt.name)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("decompressGzipData() for %s error = %q, want error containing %q", tt.name, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("decompressGzipData() for %s returned an unexpected error: %v", tt.name, err)
				}
				if !bytes.Equal(got, tt.expected) {
					t.Errorf("decompressGzipData() for %s = %s; want %s", tt.name, string(got), string(tt.expected))
				}
			}
		})
	}
}

// TestMin tests the min function.
func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a less than b", 1, 5, 1},
		{"b less than a", 10, 2, 2},
		{"a equals b", 7, 7, 7},
		{"negative numbers a less", -5, -1, -5},
		{"negative numbers b less", -2, -8, -8},
		{"zero and positive", 0, 100, 0},
		{"zero and negative", 0, -100, -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := min(tt.a, tt.b); got != tt.expected {
				t.Errorf("min(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
