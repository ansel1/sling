package sling

import (
	"encoding/json"
	"encoding/xml"

	goquery "github.com/google/go-querystring/query"
	"strings"
	"fmt"
)

func MarshalIndentedJSON(v interface{}) ([]byte, string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	return data, jsonContentType, err
}

func MarshalJSON(v interface{}) ([]byte, string, error) {
	data, err := json.Marshal(v)
	return data, jsonContentType, err
}

func UnmarshalJSON(data []byte, _ string, v interface{}) error {
	return json.Unmarshal(data, v)
}

func MarshalIndentedXML(v interface{}) ([]byte, string, error) {
	data, err := xml.MarshalIndent(v, "", "  ")
	return data, xmlContentType, err
}

func MarshalXML(v interface{}) ([]byte, string, error) {
	data, err := xml.Marshal(v)
	return data, xmlContentType, err
}

func UnmarshalXML(data []byte, _ string, v interface{}) error {
	return xml.Unmarshal(data, v)
}

func MarshalFormURLEncoded(v interface{}) ([]byte, string, error) {
	values, err := goquery.Values(v)
	if err != nil {
		return nil, "", err
	}
	return []byte(values.Encode()), formContentType, nil
}

func UnmarshalMulti(data []byte, contentType string, v interface{}) error {
	switch {
	case strings.Contains(contentType, jsonContentType):
		return UnmarshalJSON(data, contentType, v)
	case strings.Contains(contentType, xmlContentType):
		return UnmarshalXML(data, contentType, v)
	}
	return fmt.Errorf("unsupported content type: %s", contentType)
}
