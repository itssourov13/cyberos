package v1

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var uiFS embed.FS

func uiFileServer() http.Handler {
	sub, err := fs.Sub(uiFS, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(sub))
}
