package main

import (
	"net/http"
	"html/template"
)

type FSData struct {
	CutCount    int
	CopyCount   int
	FileCount   int
	CutBuffer   []string
	CopyBuffer  []string
	File        *FileNode
}

var templates = make(map[string]*template.Template)

func renderTemplate(w http.ResponseWriter, tmpl string, data any) error {
	return templates[tmpl].ExecuteTemplate(w, "base.html", data)
}

func init() {
	templates["viewDirList"] = template.Must(template.New(
		"viewDirList.html",
	).ParseFiles("templates/base.html", "templates/viewDirList.html"))
}
