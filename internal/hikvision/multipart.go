package hikvision

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
)

// MultipartWriter wraps multipart.Writer to provide a simpler interface
// for building multipart/form-data requests matching Hikvision Web UI format.
type MultipartWriter struct {
	writer *multipart.Writer
	buffer *bytes.Buffer
}

// NewMultipartWriter creates a new MultipartWriter that writes to the given buffer.
func NewMultipartWriter(buffer *bytes.Buffer) *MultipartWriter {
	return &MultipartWriter{
		writer: multipart.NewWriter(buffer),
		buffer: buffer,
	}
}

// WriteField writes a form field with the given name and value.
func (mw *MultipartWriter) WriteField(name, value string) error {
	// Write the boundary + Content-Disposition header manually
	// to match the exact format Hikvision Web UI uses
	boundary := mw.writer.Boundary()
	part := fmt.Sprintf("\r\n--%s\r\n", boundary)
	part += fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"\r\n\r\n", name)
	part += value

	_, err := mw.buffer.WriteString(part)
	return err
}

// WriteFile writes a file field with the given name, filename, content type, and data.
func (mw *MultipartWriter) WriteFile(fieldName, filename, contentType string, data []byte) error {
	boundary := mw.writer.Boundary()
	part := fmt.Sprintf("\r\n--%s\r\n", boundary)
	part += fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldName, filename)
	part += fmt.Sprintf("Content-Type: %s\r\n\r\n", contentType)

	if _, err := mw.buffer.WriteString(part); err != nil {
		return err
	}
	if _, err := mw.buffer.Write(data); err != nil {
		return err
	}

	return nil
}

// FormDataContentType returns the Content-Type header value for the multipart form data.
func (mw *MultipartWriter) FormDataContentType() string {
	return mw.writer.FormDataContentType()
}

// Close finalizes the multipart message by writing the closing boundary.
func (mw *MultipartWriter) Close() error {
	boundary := mw.writer.Boundary()
	_, err := mw.buffer.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	return err
}

// ensure interface compliance
var _ io.Closer = (*MultipartWriter)(nil)
