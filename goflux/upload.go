package goflux

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// File represents an uploaded file with metadata and content access
type File struct {
	// Filename is the original name of the uploaded file
	Filename string `json:"filename" doc:"Original filename"`

	// Size is the file size in bytes
	Size int64 `json:"size" doc:"File size in bytes"`

	// ContentType is the MIME type of the uploaded file
	ContentType string `json:"content_type" doc:"MIME type of the file"`

	// Content provides access to the file data
	Content io.ReadCloser `json:"-"`
}

// FileList represents a list of uploaded files
type FileList []*File

// NewFile creates a File instance from a multipart.FileHeader
func NewFile(fh *multipart.FileHeader) (*File, error) {
	file, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}

	return &File{
		Filename:    fh.Filename,
		Size:        fh.Size,
		ContentType: detectContentType(fh),
		Content:     file,
	}, nil
}

// NewFileList creates a FileList from multiple multipart.FileHeader instances
func NewFileList(fhs []*multipart.FileHeader) (FileList, error) {
	files := make(FileList, 0, len(fhs))
	for _, fh := range fhs {
		file, err := NewFile(fh)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

// ReadAll reads all content from the file and returns it as bytes
func (f *File) ReadAll() ([]byte, error) {
	if f.Content == nil {
		return nil, fmt.Errorf("file content is not available")
	}
	return io.ReadAll(f.Content)
}

// Extension returns the file extension (e.g., ".jpg", ".pdf")
func (f *File) Extension() string {
	return strings.ToLower(filepath.Ext(f.Filename))
}

// IsImage returns true if the file appears to be an image
func (f *File) IsImage() bool {
	return strings.HasPrefix(f.ContentType, "image/")
}

// IsDocument returns true if the file appears to be a document
func (f *File) IsDocument() bool {
	contentType := strings.ToLower(f.ContentType)
	return strings.Contains(contentType, "pdf") ||
		strings.Contains(contentType, "document") ||
		strings.Contains(contentType, "text") ||
		strings.Contains(contentType, "msword") ||
		strings.Contains(contentType, "officedocument")
}

// Close closes the file content reader
func (f *File) Close() error {
	if f.Content != nil {
		return f.Content.Close()
	}
	return nil
}

// Count returns the number of files in the list
func (fl FileList) Count() int {
	return len(fl)
}

// CloseAll closes all files in the list
func (fl FileList) CloseAll() error {
	var lastErr error
	for _, file := range fl {
		if err := file.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// detectContentType tries to determine the content type from the file header
func detectContentType(fh *multipart.FileHeader) string {
	// First, check if content type is provided in headers
	if ct := fh.Header.Get("Content-Type"); ct != "" {
		return ct
	}

	// Fallback to guessing from file extension
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}

// FileUploadResponse provides a standard response structure for file uploads
type FileUploadResponseBody struct {
	Message    string            `json:"message" doc:"Upload result message"`
	Files      []*FileInfo       `json:"files" doc:"Information about uploaded files"`
	TotalCount int               `json:"total_count" doc:"Total number of files uploaded"`
	TotalSize  int64             `json:"total_size" doc:"Total size of all uploaded files"`
	UploadedAt string            `json:"uploaded_at" doc:"Upload timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty" doc:"Additional metadata"`
}

// FileInfo contains metadata about an uploaded file
type FileInfo struct {
	Filename    string `json:"filename" doc:"Original filename"`
	Size        int64  `json:"size" doc:"File size in bytes"`
	ContentType string `json:"content_type" doc:"MIME type"`
	URL         string `json:"url,omitempty" doc:"URL to access the file"`
	ID          string `json:"id,omitempty" doc:"Unique file identifier"`
}

// NewFileUploadResponse creates a standard response from uploaded files
func NewFileUploadResponse(files FileList, message string) *struct{ Body FileUploadResponseBody } {
	fileInfos := make([]*FileInfo, len(files))
	var totalSize int64

	for i, file := range files {
		fileInfos[i] = &FileInfo{
			Filename:    file.Filename,
			Size:        file.Size,
			ContentType: file.ContentType,
		}
		totalSize += file.Size
	}

	now := time.Now().Format(time.RFC3339)

	return &struct{ Body FileUploadResponseBody }{
		Body: FileUploadResponseBody{
			Message:    message,
			Files:      fileInfos,
			TotalCount: len(files),
			TotalSize:  totalSize,
			UploadedAt: now,
		},
	}
}

// FileUploadError represents an error that occurred during file upload
type FileUploadError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
}

func (e *FileUploadError) Error() string {
	return e.Message
}

// NewFileUploadError creates a new file upload error
func NewFileUploadError(code, message, field string) *FileUploadError {
	return &FileUploadError{
		Message: message,
		Code:    code,
		Field:   field,
	}
}

// Common file upload errors
var (
	ErrNoFileUploaded     = NewFileUploadError("NO_FILE", "No file was uploaded", "file")
	ErrFileTooLarge       = NewFileUploadError("FILE_TOO_LARGE", "File size exceeds limit", "file")
	ErrInvalidFileType    = NewFileUploadError("INVALID_TYPE", "File type not allowed", "file")
	ErrTooManyFiles       = NewFileUploadError("TOO_MANY_FILES", "Too many files uploaded", "file")
	ErrInvalidFileContent = NewFileUploadError("INVALID_CONTENT", "File content is invalid", "file")
)

// Implement huma.StatusError interface for proper HTTP status codes
func (e *FileUploadError) GetStatus() int {
	switch e.Code {
	case "NO_FILE":
		return 400 // Bad Request
	case "FILE_TOO_LARGE":
		return 413 // Request Entity Too Large
	case "INVALID_TYPE", "INVALID_CONTENT":
		return 415 // Unsupported Media Type
	case "TOO_MANY_FILES":
		return 400 // Bad Request
	default:
		return 400 // Bad Request
	}
}

// GetHeaders implements huma.HeadersError interface
func (e *FileUploadError) GetHeaders() map[string][]string {
	return nil
}

// GetFileFromForm extracts a single file from multipart form
func GetFileFromForm(form *multipart.Form, fieldName string) (*File, error) {
	files, exists := form.File[fieldName]
	if !exists || len(files) == 0 {
		return nil, ErrNoFileUploaded
	}

	if len(files) > 1 {
		return nil, fmt.Errorf("expected single file in field '%s', got %d", fieldName, len(files))
	}

	return NewFile(files[0])
}

// GetFormValue extracts a form value
func GetFormValue(form *multipart.Form, fieldName string) string {
	values, exists := form.Value[fieldName]
	if !exists || len(values) == 0 {
		return ""
	}
	return values[0]
}

func (p *Procedure) RegisterMultipartUpload(api huma.API, path string, handler interface{}, options ...func(*huma.Operation)) {
	// Now that our parsing system supports MultipartFormFiles, just use regular Post
	p.Post(api, path, handler, func(o *huma.Operation) {
		// Force multipart content type
		if o.RequestBody == nil {
			o.RequestBody = &huma.RequestBody{}
		}
		if o.RequestBody.Content == nil {
			o.RequestBody.Content = map[string]*huma.MediaType{}
		}
		if _, exists := o.RequestBody.Content["multipart/form-data"]; !exists {
			o.RequestBody.Content["multipart/form-data"] = &huma.MediaType{}
		}

		// Apply user options
		for _, opt := range options {
			opt(o)
		}
	})
}

// RegisterMultipartUpload creates a simple multipart file upload endpoint with minimal boilerplate
func RegisterMultipartUpload(api huma.API, path string, handler interface{}, options ...func(*huma.Operation)) {
	NewProcedure().RegisterMultipartUpload(api, path, handler, options...)
}
