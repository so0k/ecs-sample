package ui

import (
	"html/template"

	"github.com/so0k/ecs-sample/data"
)

type TplIndexValues struct{
    Hostname string
}

var TplIndex = template.Must(template.ParseFiles("ui/templates/layout.html.tpl", "ui/templates/index.html.tpl"))

type TplUploadViewValues struct {
	Upload *data.Upload
    Hostname string
}

var TplUploadView = template.Must(template.ParseFiles("ui/templates/layout.html.tpl", "ui/templates/uploadView.html.tpl"))
