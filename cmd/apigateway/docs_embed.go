package main

import (
	"bytes"
	_ "embed"
	"net/http"
	"time"
)

//go:embed swagger.html
var swaggerHTML string

//go:embed ../../api/openapi.yaml
var openAPISpec []byte

func swaggerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "swagger.html", time.Time{}, bytes.NewReader([]byte(swaggerHTML)))
}

func openAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	http.ServeContent(w, r, "openapi.yaml", time.Time{}, bytes.NewReader(openAPISpec))
}
