package upload

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	resp "github.com/ismael/qr-restaurant/internal/shared/response"
	"github.com/ismael/qr-restaurant/pkg/storage"
	"golang.org/x/net/context"
)

const maxUploadSize = 5 * 1024 * 1024 // 5MB

var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type Handler struct {
	storageClient *storage.Client
	cfg           *config.Config
}

func NewHandler(storageClient *storage.Client, cfg *config.Config) *Handler {
	return &Handler{storageClient: storageClient, cfg: cfg}
}

type UploadResponse struct {
	URL      string `json:"url"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType string `json:"mime_type"`
}

func (h *Handler) UploadImage(c *gin.Context) {
	// Check if storage is configured
	if h.storageClient == nil {
		resp.Error(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "File storage is not configured")
		return
	}

	// Limit request body size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		resp.Error(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "File exceeds the 5MB maximum size")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		resp.ValidationError(c, "file field is required", nil)
		return
	}
	defer file.Close()

	contextStr := c.Request.FormValue("context")
	if contextStr == "" {
		contextStr = "menu_item"
	}
	allowedContexts := map[string]bool{"logo": true, "banner": true, "menu_item": true}
	if !allowedContexts[contextStr] {
		resp.ValidationError(c, "context must be one of: logo, banner, menu_item", nil)
		return
	}

	// Validate MIME type
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		resp.InternalError(c)
		return
	}
	file.Seek(0, io.SeekStart)

	mimeType := http.DetectContentType(buf)
	ext, allowed := allowedMIME[mimeType]
	if !allowed {
		resp.ValidationError(c, "Only JPEG, PNG, and WebP images are allowed", nil)
		return
	}

	// Validate file extension
	originalExt := strings.ToLower(filepath.Ext(header.Filename))
	if originalExt != ext {
		resp.ValidationError(c, "File extension does not match content type", nil)
		return
	}

	// Generate key and upload
	key := storage.GenerateKey(contextStr, header.Filename)
	ctx := context.Background()

	url, err := h.storageClient.Upload(ctx, key, file, mimeType)
	if err != nil {
		resp.InternalError(c)
		return
	}

	resp.Created(c, UploadResponse{
		URL:       url,
		SizeBytes: header.Size,
		MimeType:  mimeType,
	})
}
