package monitoring

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// LoadTemplate loads a template from the embedded filesystem
func LoadTemplate(name string) (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/"+name)
}

// MustLoadTemplate loads a template and panics if it fails
func MustLoadTemplate(name string) *template.Template {
	tmpl, err := LoadTemplate(name)
	if err != nil {
		panic(err)
	}
	return tmpl
}