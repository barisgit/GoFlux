package upload

import (
	"bytes"
	"io"
	"mime/multipart"
	"strings"
	"testing"
)

func TestNewFile(t *testing.T) {
	// Create a test multipart file header
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("test", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("test content"))
	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	defer form.RemoveAll()

	// Get the file header
	fileHeaders := form.File["test"]
	if len(fileHeaders) == 0 {
		t.Fatal("No file headers found")
	}
	fh := fileHeaders[0]

	// Test NewFile
	file, err := NewFile(fh)
	if err != nil {
		t.Fatalf("NewFile failed: %v", err)
	}
	defer file.Close()

	if file.Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", file.Filename)
	}

	if file.Size <= 0 {
		t.Errorf("Expected positive size, got %d", file.Size)
	}

	if file.ContentType == "" {
		t.Error("Expected non-empty content type")
	}

	if file.Content == nil {
		t.Error("Expected non-nil content reader")
	}
}

func TestNewFileList(t *testing.T) {
	// Create multiple test files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// File 1
	part1, _ := writer.CreateFormFile("files", "file1.txt")
	part1.Write([]byte("content1"))

	// File 2
	part2, _ := writer.CreateFormFile("files", "file2.jpg")
	part2.Write([]byte("content2"))

	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	defer form.RemoveAll()

	fileHeaders := form.File["files"]
	if len(fileHeaders) != 2 {
		t.Fatalf("Expected 2 file headers, got %d", len(fileHeaders))
	}

	// Test NewFileList
	fileList, err := NewFileList(fileHeaders)
	if err != nil {
		t.Fatalf("NewFileList failed: %v", err)
	}
	defer fileList.CloseAll()

	if len(fileList) != 2 {
		t.Errorf("Expected 2 files, got %d", len(fileList))
	}

	if fileList[0].Filename != "file1.txt" {
		t.Errorf("Expected first file to be 'file1.txt', got '%s'", fileList[0].Filename)
	}

	if fileList[1].Filename != "file2.jpg" {
		t.Errorf("Expected second file to be 'file2.jpg', got '%s'", fileList[1].Filename)
	}
}

func TestFileReadAll(t *testing.T) {
	content := "test file content"
	file := &File{
		Filename:    "test.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
		Content:     io.NopCloser(strings.NewReader(content)),
	}
	defer file.Close()

	data, err := file.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if string(data) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(data))
	}
}

func TestFileExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.txt", ".txt"},
		{"image.jpg", ".jpg"},
		{"document.pdf", ".pdf"},
		{"noextension", ""},
		{"file.tar.gz", ".gz"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			file := &File{Filename: tt.filename}
			ext := file.Extension()
			if ext != tt.expected {
				t.Errorf("Expected extension '%s', got '%s'", tt.expected, ext)
			}
		})
	}
}

func TestFileIsImage(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"text/plain", false},
		{"application/pdf", false},
		{"application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			file := &File{ContentType: tt.contentType}
			isImage := file.IsImage()
			if isImage != tt.expected {
				t.Errorf("Expected IsImage() to be %v for content type '%s'", tt.expected, tt.contentType)
			}
		})
	}
}

func TestFileIsDocument(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/pdf", true},
		{"application/msword", true},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"text/plain", true},
		{"image/jpeg", false},
		{"application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			file := &File{ContentType: tt.contentType}
			isDoc := file.IsDocument()
			if isDoc != tt.expected {
				t.Errorf("Expected IsDocument() to be %v for content type '%s'", tt.expected, tt.contentType)
			}
		})
	}
}

func TestFileClose(t *testing.T) {
	// Test closing a file with content
	file := &File{
		Content: io.NopCloser(strings.NewReader("test")),
	}

	err := file.Close()
	if err != nil {
		t.Errorf("Expected no error when closing file, got %v", err)
	}

	// Test closing a file without content
	fileNoContent := &File{}
	err = fileNoContent.Close()
	if err != nil {
		t.Errorf("Expected no error when closing file without content, got %v", err)
	}
}

func TestFileListCount(t *testing.T) {
	fileList := FileList{
		&File{Filename: "file1.txt"},
		&File{Filename: "file2.txt"},
		&File{Filename: "file3.txt"},
	}

	count := fileList.Count()
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	emptyList := FileList{}
	count = emptyList.Count()
	if count != 0 {
		t.Errorf("Expected count 0 for empty list, got %d", count)
	}
}

func TestFileListCloseAll(t *testing.T) {
	// Create file list with closeable content
	fileList := FileList{
		&File{
			Filename: "file1.txt",
			Content:  io.NopCloser(strings.NewReader("content1")),
		},
		&File{
			Filename: "file2.txt",
			Content:  io.NopCloser(strings.NewReader("content2")),
		},
	}

	err := fileList.CloseAll()
	if err != nil {
		t.Errorf("Expected no error when closing all files, got %v", err)
	}

	// Test with empty list
	emptyList := FileList{}
	err = emptyList.CloseAll()
	if err != nil {
		t.Errorf("Expected no error when closing empty list, got %v", err)
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename          string
		headerContentType string
		expected          string
	}{
		// Test header content type takes precedence
		{"test.txt", "custom/type", "custom/type"},

		// Test extension-based detection
		{"image.jpg", "", "image/jpeg"},
		{"image.jpeg", "", "image/jpeg"},
		{"image.png", "", "image/png"},
		{"image.gif", "", "image/gif"},
		{"document.pdf", "", "application/pdf"},
		{"text.txt", "", "text/plain"},
		{"data.json", "", "application/json"},
		{"config.xml", "", "application/xml"},
		{"archive.zip", "", "application/zip"},
		{"video.mp4", "", "video/mp4"},
		{"audio.mp3", "", "audio/mpeg"},
		{"unknown.xyz", "", "application/octet-stream"},
		{"noextension", "", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Create a mock file header
			fh := &multipart.FileHeader{
				Filename: tt.filename,
			}

			if tt.headerContentType != "" {
				fh.Header = make(map[string][]string)
				fh.Header.Set("Content-Type", tt.headerContentType)
			}

			contentType := detectContentType(fh)
			if contentType != tt.expected {
				t.Errorf("Expected content type '%s', got '%s'", tt.expected, contentType)
			}
		})
	}
}

func TestNewFileUploadResponse(t *testing.T) {
	// Create test files
	files := FileList{
		&File{
			Filename:    "file1.txt",
			Size:        100,
			ContentType: "text/plain",
		},
		&File{
			Filename:    "file2.jpg",
			Size:        200,
			ContentType: "image/jpeg",
		},
	}

	message := "Files uploaded successfully"
	response := NewFileUploadResponse(files, message)

	if response.Body.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, response.Body.Message)
	}

	if response.Body.TotalCount != 2 {
		t.Errorf("Expected total count 2, got %d", response.Body.TotalCount)
	}

	if response.Body.TotalSize != 300 {
		t.Errorf("Expected total size 300, got %d", response.Body.TotalSize)
	}

	if len(response.Body.Files) != 2 {
		t.Errorf("Expected 2 file infos, got %d", len(response.Body.Files))
	}

	// Check first file info
	fileInfo1 := response.Body.Files[0]
	if fileInfo1.Filename != "file1.txt" {
		t.Errorf("Expected filename 'file1.txt', got '%s'", fileInfo1.Filename)
	}
	if fileInfo1.Size != 100 {
		t.Errorf("Expected size 100, got %d", fileInfo1.Size)
	}
	if fileInfo1.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", fileInfo1.ContentType)
	}

	// Check that uploaded_at is set
	if response.Body.UploadedAt == "" {
		t.Error("Expected uploaded_at to be set")
	}
}

func TestFileUploadError(t *testing.T) {
	code := "FILE_TOO_LARGE"
	message := "File exceeds maximum size limit"
	field := "upload"

	err := NewFileUploadError(code, message, field)

	if err.Code != code {
		t.Errorf("Expected code '%s', got '%s'", code, err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Field != field {
		t.Errorf("Expected field '%s', got '%s'", field, err.Field)
	}

	// Test Error() method
	errorStr := err.Error()
	if errorStr != message {
		t.Errorf("Expected error string '%s', got '%s'", message, errorStr)
	}

	// Test GetStatus for different codes
	statusTests := []struct {
		code           string
		expectedStatus int
	}{
		{"FILE_TOO_LARGE", 413},
		{"INVALID_TYPE", 415}, // This matches the actual constant
		{"TOO_MANY_FILES", 400},
		{"NO_FILE", 400},         // This matches the actual constant
		{"INVALID_CONTENT", 415}, // This matches the actual constant
		{"UNKNOWN_ERROR", 400},   // Unknown codes default to 400
	}

	for _, tt := range statusTests {
		t.Run(tt.code, func(t *testing.T) {
			err := &FileUploadError{Code: tt.code}
			status := err.GetStatus()
			if status != tt.expectedStatus {
				t.Errorf("Expected status %d for code '%s', got %d", tt.expectedStatus, tt.code, status)
			}
		})
	}

	// Test GetHeaders - it's fine for it to return nil
	headers := err.GetHeaders()
	if headers != nil {
		t.Log("GetHeaders returned headers (this is fine, but not required)")
	}
}

func TestGetFileFromForm(t *testing.T) {
	// Create a test multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("upload", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	defer form.RemoveAll()

	// Test getting existing file
	file, err := GetFileFromForm(form, "upload")
	if err != nil {
		t.Fatalf("GetFileFromForm failed: %v", err)
	}
	defer file.Close()

	if file.Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", file.Filename)
	}

	// Test getting non-existent file
	_, err = GetFileFromForm(form, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetFormValue(t *testing.T) {
	// Create a test multipart form with text fields
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "John Doe")
	writer.WriteField("age", "30")
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}
	defer form.RemoveAll()

	// Test getting existing values
	name := GetFormValue(form, "name")
	if name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", name)
	}

	age := GetFormValue(form, "age")
	if age != "30" {
		t.Errorf("Expected age '30', got '%s'", age)
	}

	// Test getting non-existent value
	email := GetFormValue(form, "email")
	if email != "" {
		t.Errorf("Expected empty string for non-existent field, got '%s'", email)
	}
}
