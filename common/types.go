package common

import "regexp"

type (
	Rules []Rule
	Rule  struct {
		Name   string
		Table  string
		Regexp regexp.Regexp
		Fields []map[string]interface{}
	}

	Field struct {
		Name string
		Type string
	}

	ModelField struct {
		Name string
		Type string
	}

	Model struct {
		Table     string
		Statement string
		Fields    []ConvertableField
	}

	ConvertableField interface {
		Convert(value string) (interface{}, error)
		GetName() string
	}

	StringModelField struct {
		ModelField
	}

	TimestampModelField struct {
		Layout string
		ModelField
	}

	ListModelField struct {
		Values  map[string]int
		Default *int
		ModelField
	}

	IpToIntModelField struct {
		ModelField
	}

	Int16ModelField struct {
		ModelField
	}

	UInt16ModelField struct {
		ModelField
	}
)

func (f *ModelField) GetName() string {
	return f.Name
}
