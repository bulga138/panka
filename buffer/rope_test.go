package buffer

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRope(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"single char", "a", "a"},
		{"single line", "hello world", "hello world"},
		{"multiple lines", "line1\nline2\nline3", "line1\nline2\nline3"},
		{"unicode", "こんにちは", "こんにちは"},
		{"mixed unicode", "hello 世界\nworld", "hello 世界\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.input)
			if r == nil {
				t.Fatal("NewRope returned nil")
			}
			var buf bytes.Buffer
			_, err := r.WriteTo(&buf)
			if err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestRope_Insert(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		line     int
		col      int
		r        rune
		expected string
	}{
		{"insert at start", "hello", 0, 0, 'X', "Xhello"},
		{"insert at end", "hello", 0, 5, 'X', "helloX"},
		{"insert middle", "hello", 0, 2, 'X', "heXllo"},
		{"insert newline", "hello", 0, 2, '\n', "he\nllo"},
		{"insert at line start", "line1\nline2", 1, 0, 'X', "line1\nXline2"},
		{"insert at line end", "line1\nline2", 1, 5, 'X', "line1\nline2X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			r.Insert(tt.line, tt.col, tt.r)
			var buf bytes.Buffer
			r.WriteTo(&buf)
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestRope_Delete(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		line     int
		col      int
		expected string
	}{
		{"delete at start (no-op)", "hello", 0, 0, "hello"},
		{"delete first char", "hello", 0, 1, "ello"},
		{"delete middle char", "hello", 0, 3, "helo"},
		{"delete last char", "hello", 0, 5, "hell"},
		{"delete newline", "line1\nline2", 1, 0, "line1line2"},
		{"delete across lines", "a\nb", 1, 0, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			r.Delete(tt.line, tt.col)
			var buf bytes.Buffer
			r.WriteTo(&buf)
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestRope_GetLine(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		line     int
		expected string
	}{
		{"single line", "hello", 0, "hello"},
		{"first line", "line1\nline2\nline3", 0, "line1"},
		{"middle line", "line1\nline2\nline3", 1, "line2"},
		{"last line", "line1\nline2\nline3", 2, "line3"},
		{"empty line", "line1\n\nline3", 1, ""},
		{"out of bounds", "hello", 5, ""},
		{"negative line", "hello", -1, ""},
		{"trailing newline", "line1\nline2\n", 1, "line2"},
		{"windows line endings", "line1\r\nline2", 0, "line1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			result := r.GetLine(tt.line)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRope_LineCount(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		expected int
	}{
		{"empty", "", 1},
		{"single line", "hello", 1},
		{"two lines", "line1\nline2", 2},
		{"three lines", "line1\nline2\nline3", 3},
		{"trailing newline", "line1\nline2\n", 3},
		{"empty lines", "\n\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			result := r.LineCount()
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestRope_WriteTo(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		expected string
	}{
		{"empty", "", ""},
		{"single line", "hello", "hello"},
		{"multiple lines", "line1\nline2\nline3", "line1\nline2\nline3"},
		{"unicode", "こんにちは\n世界", "こんにちは\n世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			var buf bytes.Buffer
			n, err := r.WriteTo(&buf)
			if err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
			expectedLen := int64(len(tt.expected))
			if n != expectedLen {
				t.Errorf("expected %d bytes written, got %d", expectedLen, n)
			}
		})
	}
}

func TestRope_RuneAt(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		index    int
		expected rune
		hasError bool
	}{
		{"first char", "hello", 0, 'h', false},
		{"middle char", "hello", 2, 'l', false},
		{"last char", "hello", 4, 'o', false},
		{"newline", "a\nb", 1, '\n', false},
		{"unicode", "こんにちは", 0, 'こ', false},
		{"out of bounds", "hello", 10, 0, true},
		{"negative index", "hello", -1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRope(tt.initial)
			result, err := r.RuneAt(tt.index)
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %c, got %c", tt.expected, result)
				}
			}
		})
	}
}

func TestRope_InsertDeleteSequence(t *testing.T) {
	r := NewRope("")
	
	// Insert "hello"
	for i, c := range "hello" {
		r.Insert(0, i, c)
	}
	
	// Insert " world" at end
	for i, c := range " world" {
		r.Insert(0, 5+i, c)
	}
	
	var buf bytes.Buffer
	r.WriteTo(&buf)
	if buf.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}
	
	// Delete " world"
	for i := 0; i < 6; i++ {
		r.Delete(0, 11-i)
	}
	
	buf.Reset()
	r.WriteTo(&buf)
	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestRope_LargeInsert(t *testing.T) {
	r := NewRope("")
	
	// Insert a large string to trigger node splitting
	largeText := strings.Repeat("a", maxLeafSize*3)
	for i, c := range largeText {
		r.Insert(0, i, c)
	}
	
	var buf bytes.Buffer
	r.WriteTo(&buf)
	if buf.String() != largeText {
		t.Errorf("large insert failed: length mismatch")
	}
	
	// Verify we can still get lines correctly
	if r.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", r.LineCount())
	}
}

func TestRope_MultipleLines(t *testing.T) {
	r := NewRope("")
	
	// Build "line1\nline2\nline3"
	text := "line1\nline2\nline3"
	for i, c := range text {
		line := strings.Count(text[:i], "\n")
		col := i - strings.LastIndex(text[:i], "\n") - 1
		if col < 0 {
			col = 0
		}
		r.Insert(line, col, c)
	}
	
	if r.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", r.LineCount())
	}
	
	if r.GetLine(0) != "line1" {
		t.Errorf("line 0: expected 'line1', got %q", r.GetLine(0))
	}
	if r.GetLine(1) != "line2" {
		t.Errorf("line 1: expected 'line2', got %q", r.GetLine(1))
	}
	if r.GetLine(2) != "line3" {
		t.Errorf("line 2: expected 'line3', got %q", r.GetLine(2))
	}
}

func BenchmarkRope_Insert(b *testing.B) {
	r := NewRope("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Insert(0, i%100, 'a')
	}
}

func BenchmarkRope_GetLine(b *testing.B) {
	text := strings.Repeat("line with some text\n", 100)
	r := NewRope(text)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.GetLine(i % r.LineCount())
	}
}

func BenchmarkRope_WriteTo(b *testing.B) {
	text := strings.Repeat("line with some text\n", 1000)
	r := NewRope(text)
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		r.WriteTo(&buf)
	}
}

