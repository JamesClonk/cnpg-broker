package ui

import (
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/hako/durafmt"
	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func (h *Handler) RegisterRenderer(e *echo.Echo) {
	funcMap := template.FuncMap{
		"ToLower":    strings.ToLower,
		"TrimPrefix": strings.TrimPrefix,
		"Duration":   durafmt.Parse,
		"Bytes":      func(b int) string { return humanize.Bytes(uint64(b)) },
		"KBytes":     func(b int) string { return humanize.Bytes(uint64(b * 1000 * 1000 / 1024)) },
		"MBytes":     func(b int) string { return humanize.Bytes(uint64(b * 1000 * 1000 * 1000 / 1024)) },
		"GBytes":     func(b int) string { return humanize.Bytes(uint64(b * 1000 * 1000 * 1000 * 1000 / 1024)) },
		"Time":       humanize.Time,
	}
	renderer := &TemplateRenderer{
		templates: template.Must(template.New("ui").Funcs(sprig.FuncMap()).Funcs(funcMap).ParseGlob("public/*.html")),
	}
	e.Renderer = renderer
}
