package template

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/suzuki-shunsuke/buildflow/pkg/util"
)

type Template struct {
	tpl *template.Template
}

func Compile(tpl string) (Template, error) {
	tmpl, err := template.New("text").Funcs(util.GetTemplateUtil()).Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return Template{}, err
	}
	return Template{
		tpl: tmpl,
	}, nil
}

func (tpl Template) Render(params interface{}) (string, error) {
	if tpl.tpl == nil {
		return "", nil
	}
	buf := &bytes.Buffer{}
	if err := tpl.tpl.Execute(buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
