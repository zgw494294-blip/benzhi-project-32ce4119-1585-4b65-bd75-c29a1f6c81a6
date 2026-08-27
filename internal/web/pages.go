package web

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

var pageTemplates = template.Must(template.ParseFS(assets, "assets/index.html", "assets/credential.html"))

func (s *Server) HandleWorkbenchPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if err := pageTemplates.ExecuteTemplate(w, "index.html", map[string]any{"Title": "湿地修复苗木投放核验工作台"}); err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
	}
}

func (s *Server) HandleCredentialPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if err := pageTemplates.ExecuteTemplate(w, "credential.html", map[string]any{"CredentialID": pathID(r, "id")}); err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
	}
}
