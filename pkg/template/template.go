package template

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/suzuki-shunsuke/buildflow/pkg/util"
)

type Template struct {
	Template *template.Template
	Text     string
}

func Compile(tpl string) (Template, error) {
	tmpl, err := template.New("text").Funcs(util.GetTemplateUtil()).Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return Template{}, err
	}
	return Template{
		Template: tmpl,
		Text:     tpl,
	}, nil
}

func (tpl Template) GetRaw() string {
	return tpl.Text
}

func (tpl Template) Render(params interface{}) (string, error) {
	if tpl.Template == nil {
		return "", nil
	}
	buf := &bytes.Buffer{}
	if err := tpl.Template.Execute(buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tpl *Template) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var src string
	if err := unmarshal(&src); err != nil {
		return err
	}
	t, err := Compile(src)
	if err != nil {
		return err
	}
	*tpl = t
	return nil
}
