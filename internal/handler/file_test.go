package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"suseoaa/internal/service"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFileHandler_UploadImage_InvalidScene(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&service.FileService{})
	r := gin.New()
	r.POST("/upload/image", h.UploadImage)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("scene", "../hack")
	part, _ := writer.CreateFormFile("file", "test.png")
	_, _ = part.Write([]byte("fake image content"))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/upload/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid scene, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestFileHandler_UploadFile_InvalidScene(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFileHandler(&service.FileService{})
	r := gin.New()
	r.POST("/upload/file", h.UploadFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("scene", "file/nested")
	part, _ := writer.CreateFormFile("file", "test.txt")
	_, _ = part.Write([]byte("fake file content"))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/upload/file", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid scene, got %d, body: %s", w.Code, w.Body.String())
	}
}
