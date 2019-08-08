package main

import (
	"html/template"
	"io"

	"go.shu.run/lu"
)

const html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
	<title>IPGeo</title>
	<style>
	label { font-size:14px; font-weight:normal; }
	span { font-size:16px; font-weight:bold; }
	</style>
</head>
<body>
	<div style="text-align:center;">
		<label>IP:</label>
		<span>{{ .IP }}</span>
		{{ if eq .Code "internet" }}

		{{ with .Continent }}
		<label>Continent:</label>
		<span>{{ .Name }}</span>
		{{ end }}

		{{ with .Country }}
		<label>Country:</label>
		<span>{{ .Name }}</span>
		{{ end }}

		{{ with .City }}
		<label>City:</label>
		<span>{{ .Name }}</span>
		{{ end }}

		{{ else if eq .Code "internal" "local" }}
		<label>Code:</label>
		<label>{{ .Code }}</label>
		{{ else }}
		<label>Error:</label>
		<label>{{ .Msg }}</label>
		{{ end }}
	</div>
</body>
</html>`

type iprTemplate struct {
	templates *template.Template
}

func (t *iprTemplate) Render(w io.Writer, name string, data interface{}, c lu.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
