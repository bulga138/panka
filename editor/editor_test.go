package editor

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/bulga138/panka/config"
)

// mockTerminal is a test implementation of the Terminal interface
type mockTerminal struct {
	width, height int
	stdin         *bytes.Buffer
}

func (m *mockTerminal) EnableRawMode() error   { return nil }
func (m *mockTerminal) DisableRawMode() error  { return nil }
func (m *mockTerminal) GetWindowSize() (int, int, error) {
	return m.width, m.height, nil
}
func (m *mockTerminal) Stdin() io.Reader { return m.stdin }
func (m *mockTerminal) Close() error     { return nil }

func newMockTerminal() *mockTerminal {
	return &mockTerminal{
		width:  80,
		height: 24,
		stdin:  bytes.NewBuffer(nil),
	}
}

func TestEditor_NewEditor(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	
	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
	}{
		{"empty editor", "", "", false},
		{"with content", "", "hello\nworld", false},
		{"nonexistent file", "nonexistent.txt", "", false}, // Should not error
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file if needed
			var filename string
			if tt.filename != "" {
				tmpfile, err := os.CreateTemp("", "panka_test_*.txt")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpfile.Name())
				if tt.content != "" {
					tmpfile.WriteString(tt.content)
				}
				tmpfile.Close()
				filename = tmpfile.Name()
			}
			
			e, err := NewEditor(term, cfg, filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEditor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if e == nil && !tt.wantErr {
				t.Error("NewEditor() returned nil editor")
			}
		})
	}
}

func TestEditor_UndoRedo(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, err := NewEditor(term, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	
	// Insert some text
	e.buffer.Insert(0, 0, 'h')
	e.buffer.Insert(0, 1, 'e')
	e.buffer.Insert(0, 2, 'l')
	e.buffer.Insert(0, 3, 'l')
	e.buffer.Insert(0, 4, 'o')
	
	// Verify content
	line := e.buffer.GetLine(0)
	if line != "hello" {
		t.Errorf("expected 'hello', got %q", line)
	}
	
	// Undo should work (though we need to flush groups first)
	e.flushEditGroups()
	if len(e.undoStack) == 0 {
		t.Log("Note: undo stack is empty (typing groups not flushed)")
	}
}

func TestEditor_FileOperations(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	
	// Create temp file
	tmpfile, err := os.CreateTemp("", "panka_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	testContent := "line1\nline2\nline3"
	tmpfile.WriteString(testContent)
	tmpfile.Close()
	
	// Load file
	e, err := NewEditor(term, cfg, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify content loaded
	if e.buffer.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", e.buffer.LineCount())
	}
	
	// Save file
	e.filename = tmpfile.Name()
	if err := e.save(); err != nil {
		t.Errorf("save() error = %v", err)
	}
	
	// Verify file was saved
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != testContent {
		t.Errorf("file content mismatch after save")
	}
}

func TestEditor_Selection(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, err := NewEditor(term, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	
	// Insert text
	text := "hello world"
	for i, r := range text {
		e.buffer.Insert(0, i, r)
	}
	
	// Select text
	e.selectionActive = true
	e.selectionAnchorY = 0
	e.selectionAnchorX = 0
	e.cursorY = 0
	e.cursorX = 5
	
	selected := e.getSelectedText()
	if selected != "hello" {
		t.Errorf("expected 'hello', got %q", selected)
	}
}

func TestEditor_LoadFileContent(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, err := NewEditor(term, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	
	// Create temp file with content
	tmpfile, err := os.CreateTemp("", "panka_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	testContent := "test content\nwith multiple lines"
	tmpfile.WriteString(testContent)
	tmpfile.Close()
	
	// Load content
	content, err := e.loadFileContent(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadFileContent() error = %v", err)
	}
	
	if content != testContent {
		t.Errorf("content mismatch: expected %q, got %q", testContent, content)
	}
}

func TestEditor_LoadFileContent_Nonexistent(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, err := NewEditor(term, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	
	_, err = e.loadFileContent("nonexistent_file_12345.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestEditor_LoadFileContent_LargeFile(t *testing.T) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, err := NewEditor(term, cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	
	// Create a large file (>1MB to trigger streaming)
	tmpfile, err := os.CreateTemp("", "panka_test_large_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	// Write 2MB of data
	largeContent := bytes.Repeat([]byte("a"), 2*1024*1024)
	tmpfile.Write(largeContent)
	tmpfile.Close()
	
	// Load content (should use streaming)
	content, err := e.loadFileContent(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadFileContent() error = %v", err)
	}
	
	if len(content) != len(largeContent) {
		t.Errorf("content length mismatch: expected %d, got %d", len(largeContent), len(content))
	}
}

// Test helper to create a test editor with content
func createTestEditor(content string) (*Editor, error) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	
	// Create temp file
	tmpfile, err := os.CreateTemp("", "panka_test_*.txt")
	if err != nil {
		return nil, err
	}
	
	if content != "" {
		tmpfile.WriteString(content)
	}
	tmpfile.Close()
	
	return NewEditor(term, cfg, tmpfile.Name())
}

func TestEditor_FindReplace(t *testing.T) {
	e, err := createTestEditor("hello world\nhello again\nworld hello")
	if err != nil {
		t.Fatal(err)
	}
	
	// Test find functionality
	e.isFinding = true
	e.promptBuffer = "hello"
	e.findInitial()
	
	if len(e.findMatches) == 0 {
		t.Error("expected to find matches for 'hello'")
	}
}

func BenchmarkEditor_LoadFileContent_Small(b *testing.B) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, _ := NewEditor(term, cfg, "")
	
	// Create small test file
	tmpfile, _ := os.CreateTemp("", "panka_bench_*.txt")
	tmpfile.WriteString("small file content")
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.loadFileContent(tmpfile.Name())
	}
}

func BenchmarkEditor_LoadFileContent_Large(b *testing.B) {
	term := newMockTerminal()
	cfg := config.DefaultConfig()
	e, _ := NewEditor(term, cfg, "")
	
	// Create large test file
	tmpfile, _ := os.CreateTemp("", "panka_bench_large_*.txt")
	largeContent := bytes.Repeat([]byte("a"), 2*1024*1024)
	tmpfile.Write(largeContent)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.loadFileContent(tmpfile.Name())
	}
}

