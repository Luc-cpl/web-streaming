package render

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

var path = "views/files-min/"

func Render(w http.ResponseWriter, data interface{}, templateFile string, files map[string]string) {
	tmpl := template.New("")

	var html string
	if templateFile != "" {
		byt, _ := ioutil.ReadFile(path + templateFile)
		html = string(byt)
	}

	for key, element := range files {
		if element != "" {
			file, err := ioutil.ReadFile(path + element)
			if err == nil {
				if strings.EqualFold(filepath.Ext(path+element), ".html") {
					html += string(file)
				} else {
					html += `{{define "` + key + `"}}` + string(file) + `{{end}}`
				}
			}
		}

	}
	tmpl, _ = tmpl.Parse(html)

	if data != nil {
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}
