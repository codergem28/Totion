package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// setupTestNotesDir creates a temporary directory for testing and sets NotesDir
func setupTestNotesDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "totion-app-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save original NotesDir
	originalNotesDir := NotesDir
	NotesDir = tmpDir

	// Cleanup function to restore original
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
		NotesDir = originalNotesDir
	})

	return tmpDir
}

// createTestNoteFile creates a test note file
func createTestNoteFile(t *testing.T, name, content string) string {
	filePath := filepath.Join(NotesDir, name+".md")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test note: %v", err)
	}
	return filePath
}

func TestInitialModel(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)

	model := InitialModel()

	if model.CreateFileInputVisible != false {
		t.Errorf("Expected CreateFileInputVisible to be false, got %v", model.CreateFileInputVisible)
	}

	if model.ListVisible != false {
		t.Errorf("Expected ListVisible to be false, got %v", model.ListVisible)
	}

	if model.CurrentNote != nil {
		t.Errorf("Expected CurrentNote to be nil, got %v", model.CurrentNote)
	}

	if model.ErrMsg != "" {
		t.Errorf("Expected ErrMsg to be empty, got %s", model.ErrMsg)
	}

	if model.AutoCompleteEnabled != false {
		t.Errorf("Expected AutoCompleteEnabled to be false, got %v", model.AutoCompleteEnabled)
	}

	if model.Suggestion != "" {
		t.Errorf("Expected Suggestion to be empty, got %s", model.Suggestion)
	}

	if model.Ctx == nil {
		t.Error("Expected Ctx to be set, got nil")
	}

	if model.Width == 0 || model.Height == 0 {
		t.Errorf("Expected Width and Height to be set, got Width=%d, Height=%d", model.Width, model.Height)
	}
}

func TestModel_OpenOrCreateFile(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)

	t.Run("create new file", func(t *testing.T) {
		model := InitialModel()
		filePath := filepath.Join(NotesDir, "newfile.md")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if model.CurrentNote == nil {
			t.Error("Expected CurrentNote to be set after opening file")
		}

		if model.ErrMsg != "" {
			t.Errorf("Expected ErrMsg to be empty, got %s", model.ErrMsg)
		}

		// Verify file was created
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Expected file to be created, but it doesn't exist")
		}

		if model.CurrentNote != nil {
			model.CurrentNote.Close()
		}
	})

	t.Run("open existing file", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "existing", "existing content")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if model.CurrentNote == nil {
			t.Fatal("Expected CurrentNote to be set")
		}

		content := model.NoteContent.Value()
		if content != "existing content" {
			t.Errorf("Expected content 'existing content', got '%s'", content)
		}

		if model.CurrentNote != nil {
			model.CurrentNote.Close()
		}
	})

	t.Run("handle file with content", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "testfile", "line 1\nline 2\nline 3")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		content := model.NoteContent.Value()
		if !strings.Contains(content, "line 1") {
			t.Errorf("Expected content to contain 'line 1', got '%s'", content)
		}

		if model.CurrentNote != nil {
			model.CurrentNote.Close()
		}
	})
}

func TestModel_SaveNote(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)

	t.Run("save note with content", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "savetest", "original content")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		// Modify content
		model.NoteContent.SetValue("updated content")

		// Save note
		model.SaveNote()

		// Verify file was saved
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != "updated content" {
			t.Errorf("Expected content 'updated content', got '%s'", string(content))
		}

		if model.CurrentNote != nil {
			t.Error("Expected CurrentNote to be nil after saving")
		}

		if model.NoteContent.Value() != "" {
			t.Errorf("Expected NoteContent to be cleared, got '%s'", model.NoteContent.Value())
		}
	})

	t.Run("save note when CurrentNote is nil", func(t *testing.T) {
		model := InitialModel()

		// Should not panic or error when CurrentNote is nil
		model.SaveNote()

		if model.ErrMsg != "" {
			t.Errorf("Expected no error message, got '%s'", model.ErrMsg)
		}
	})

	t.Run("save empty note", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "empty", "original")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		model.NoteContent.SetValue("")
		model.SaveNote()

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != "" {
			t.Errorf("Expected empty content, got '%s'", string(content))
		}
	})
}

func TestModel_Update(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)

	t.Run("window size message", func(t *testing.T) {
		model := InitialModel()
		originalWidth := model.Width
		originalHeight := model.Height

		newModel, cmd := model.Update(tea.WindowSizeMsg{
			Width:  120,
			Height: 40,
		})

		updatedModel := newModel.(Model)
		if updatedModel.Width != 120 {
			t.Errorf("Expected Width 120, got %d", updatedModel.Width)
		}

		if updatedModel.Height != 40 {
			t.Errorf("Expected Height 40, got %d", updatedModel.Height)
		}

		// WindowSizeMsg returns nil command (correct behavior)
		if cmd != nil {
			t.Error("Expected nil command for WindowSizeMsg, got command")
		}

		// Verify dimensions changed
		if updatedModel.Width == originalWidth {
			t.Error("Width should have changed")
		}

		if updatedModel.Height == originalHeight {
			t.Error("Height should have changed")
		}
	})

	t.Run("suggestion message with error", func(t *testing.T) {
		model := InitialModel()

		newModel, cmd := model.Update(suggestionMsg{
			suggestion: "",
			err:        context.DeadlineExceeded,
		})

		updatedModel := newModel.(Model)
		if updatedModel.ErrMsg == "" {
			t.Error("Expected error message to be set")
		}

		if !strings.Contains(updatedModel.ErrMsg, "Suggestion error") {
			t.Errorf("Expected error message to contain 'Suggestion error', got '%s'", updatedModel.ErrMsg)
		}

		if updatedModel.Suggestion != "" {
			t.Errorf("Expected Suggestion to be empty on error, got '%s'", updatedModel.Suggestion)
		}

		if cmd != nil {
			t.Error("Expected no command, got command")
		}
	})

	t.Run("suggestion message success", func(t *testing.T) {
		model := InitialModel()

		newModel, cmd := model.Update(suggestionMsg{
			suggestion: "This is a suggestion",
			err:        nil,
		})

		updatedModel := newModel.(Model)
		if updatedModel.Suggestion != "This is a suggestion" {
			t.Errorf("Expected Suggestion 'This is a suggestion', got '%s'", updatedModel.Suggestion)
		}

		if updatedModel.ErrMsg != "" {
			t.Errorf("Expected no error message, got '%s'", updatedModel.ErrMsg)
		}

		if cmd != nil {
			t.Error("Expected no command, got command")
		}
	})

	t.Run("init command", func(t *testing.T) {
		model := InitialModel()
		cmd := model.Init()

		if cmd == nil {
			t.Error("Expected Init to return a command, got nil")
		}
	})
}

func TestModel_generateSuggestionCmd(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)
	
	t.Run("nil client", func(t *testing.T) {
		model := InitialModel()
		model.Client = nil

		cmd := model.generateSuggestionCmd()
		msg := cmd()

		sMsg, ok := msg.(suggestionMsg)
		if !ok {
			t.Fatalf("Expected suggestionMsg, got %T", msg)
		}

		if sMsg.err == nil {
			t.Error("Expected error when client is nil")
		}

		if !strings.Contains(sMsg.err.Error(), "AI client not available") {
			t.Errorf("Expected error about AI client, got '%v'", sMsg.err)
		}
	})
}

func TestModel_View(t *testing.T) {
	tmpDir := setupTestNotesDir(t)
	defer os.RemoveAll(tmpDir)

	t.Run("default view", func(t *testing.T) {
		model := InitialModel()
		view := model.View()

		if !strings.Contains(view, "Welcome to the TOTION") {
			t.Error("Expected view to contain welcome message")
		}

		if !strings.Contains(view, "Ctrl+N to create") {
			t.Error("Expected view to contain create note instruction")
		}
	})

	t.Run("view with error message", func(t *testing.T) {
		model := InitialModel()
		model.ErrMsg = "Test error message"

		view := model.View()
		if !strings.Contains(view, "Test error message") {
			t.Error("Expected view to contain error message")
		}
	})

	t.Run("view when note is open", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "viewtest", "test content")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		view := model.View()
		if !strings.Contains(view, "Ctrl+S: Save Note") {
			t.Error("Expected view to contain save help when note is open")
		}

		if model.CurrentNote != nil {
			model.CurrentNote.Close()
		}
	})

	t.Run("view with suggestion", func(t *testing.T) {
		model := InitialModel()
		filePath := createTestNoteFile(t, "sugtest", "content")

		err := model.OpenOrCreateFile(filePath)
		if err != nil {
			t.Fatalf("Failed to open file: %v", err)
		}

		model.AutoCompleteEnabled = true
		model.Suggestion = "suggested text"

		view := model.View()
		if !strings.Contains(view, "Suggestion:") {
			t.Error("Expected view to contain suggestion")
		}

		if !strings.Contains(view, "Tab to accept") {
			t.Error("Expected view to contain tab instruction")
		}

		if model.CurrentNote != nil {
			model.CurrentNote.Close()
		}
	})
}
