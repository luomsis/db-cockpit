package dataquery

import (
	"context"
	"embed"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

//go:embed static/swagger-ui/*
var swaggerUIFS embed.FS

// SwaggerUIHandler returns a handler that serves the Swagger UI static files
func SwaggerUIHandler() func(ctx context.Context, c *app.RequestContext) {
	// Get the subdirectory containing swagger-ui files
	subFS, err := fs.Sub(swaggerUIFS, "static/swagger-ui")
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, c *app.RequestContext) {
		// Extract path after /swagger
		path := strings.TrimPrefix(string(c.URI().Path()), "/swagger")
		if path == "" || path == "/" {
			path = "/index.html"
		}
		// Remove leading slash
		path = strings.TrimPrefix(path, "/")

		// Read file from embedded FS
		data, err := fs.ReadFile(subFS, path)
		if err != nil {
			c.String(404, "File not found: "+path)
			return
		}

		// Set content type based on file extension
		ext := filepath.Ext(path)
		contentType := getContentType(ext)
		c.Response.Header.Set("Content-Type", contentType)

		// Write file content
		c.Response.SetBody(data)
	}
}

func getContentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}