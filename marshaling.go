package sling

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	goquery "github.com/google/go-querystring/query"
	"strings"
)

var DefaultMarshaler BodyMarshaler = &JSONMarshaler{}
var DefaultUnmarshaler BodyUnmarshaler = &MultiUnmarshaler{}

type BodyMarshaler interface {
	Marshal(v interface{}) (data []byte, contentType string, err error)
}

type BodyUnmarshaler interface {
	Unmarshal(data []byte, contentType string, v interface{}) error
}

type MarshalFunc func(v interface{}) ([]byte, string, error)

func (f MarshalFunc) Marshal(v interface{}) ([]byte, string, error) {
	return f(v)
}

type UnmarshalFunc func(data []byte, contentType string, v interface{}) error

func (f UnmarshalFunc) Unmarshal(data []byte, contentType string, v interface{}) error {
	return f(data, contentType, v)
}

type JSONMarshaler struct {
	Indent bool
}

func (m *JSONMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (m *JSONMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	return data, CONTENT_TYPE_JSON, err
}

type XMLMarshaler struct {
	Indent bool
}

func (*XMLMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return xml.Unmarshal(data, v)
}

func (m *XMLMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = xml.MarshalIndent(v, "", "  ")
	} else {
		data, err = xml.Marshal(v)
	}
	return data, CONTENT_TYPE_XML, err
}

type FormMarshaler struct{}

func (*FormMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	values, err := goquery.Values(v)
	if err != nil {
		return nil, "", err
	}
	return []byte(values.Encode()), CONTENT_TYPE_FORM, nil
}

type MultiUnmarshaler struct {
	jsonMar JSONMarshaler
	xmlMar  XMLMarshaler
}

func (m *MultiUnmarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	switch {
	case strings.Contains(contentType, CONTENT_TYPE_JSON):
		return m.jsonMar.Unmarshal(data, contentType, v)
	case strings.Contains(contentType, CONTENT_TYPE_XML):
		return m.xmlMar.Unmarshal(data, contentType, v)
	}
	return fmt.Errorf("unsupported content type: %s", contentType)
}
