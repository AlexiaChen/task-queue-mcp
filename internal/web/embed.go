package web

import "embed"

// StaticFS holds the embedded static files
//
//go:embed static/*
var StaticFS embed.FS
