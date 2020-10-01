package config

import "github.com/suzuki-shunsuke/buildflow/pkg/template"

type Template struct {
	Text     string
	Template template.Template
}

func (tmpl *Template) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var src string
	if err := unmarshal(&src); err != nil {
		return err
	}
	t, err := template.Compile(src)
	if err != nil {
		return err
	}
	tmpl.Text = src
	tmpl.Template = t
	return nil
}

func (tmpl Template) New(params interface{}) (Template, error) {
	txt, err := tmpl.Template.Render(params)
	return Template{
		Text:     txt,
		Template: tmpl.Template,
	}, err
}
