package docs

import (
	"embed"
	"net/http"
)

//go:embed api/*
var fs embed.FS

var Handler = http.FileServer(http.FS(fs))
