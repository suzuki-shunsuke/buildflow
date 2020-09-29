package template

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
)

type Template struct {
	tpl *template.Template
}

func Compile(tpl string) (Template, error) {
	tmpl, err := template.New("text").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return Template{}, err
	}
	return Template{
		tpl: tmpl,
	}, nil
}

func (tpl Template) Render(params interface{}) (string, error) {
	buf := &bytes.Buffer{}
	if err := tpl.tpl.Execute(buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
