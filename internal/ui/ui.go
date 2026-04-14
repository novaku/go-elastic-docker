package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to load embedded ui assets")
	}
	return http.FileServer(http.FS(sub))
}
