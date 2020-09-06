package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/microcosm-cc/bluemonday"
	"net/http"
)

var sanitizer *bluemonday.Policy

func init() {
	sanitizer = bluemonday.UGCPolicy()
	sanitizer.AllowURLSchemes("ob")
}

func sanitizedStringResponse(w http.ResponseWriter, response string) {
	ret, err := sanitizeJSON([]byte(response))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, string(ret))
}

func sanitizedJSONResponse(w http.ResponseWriter, i interface{}) {
	out, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret, err := sanitizeJSON(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprint(w, string(ret))
}

func sanitizedProtobufResponse(w http.ResponseWriter, m proto.Message) {
	out, err := sanitizeProtobuf(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprint(w, string(out))
}
func marshalAndSanitizeJSON(i interface{}) ([]byte, error) {
	out, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return nil, err
	}
	return sanitizeJSON(out)
}

func sanitizeJSON(s []byte) ([]byte, error) {
	d := json.NewDecoder(bytes.NewReader(s))
	d.UseNumber()

	var i interface{}
	err := d.Decode(&i)
	if err != nil {
		return nil, err
	}
	sanitize(i)

	return json.MarshalIndent(i, "", "    ")
}

func sanitizeProtobuf(m proto.Message) ([]byte, error) {
	marshaler := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "    ",
		OrigName:     false,
	}

	out, err := marshaler.MarshalToString(m)
	if err != nil {
		return nil, err
	}

	return sanitizeJSON([]byte(out))
}

func sanitize(data interface{}) {
	switch d := data.(type) {
	case map[string]interface{}:
		for k, v := range d {
			switch tv := v.(type) {
			case string:
				d[k] = sanitizer.Sanitize(tv)
			case map[string]interface{}:
				sanitize(tv)
			case []interface{}:
				sanitize(tv)
			case nil:
				delete(d, k)
			}
		}
	case []interface{}:
		if len(d) > 0 {
			switch d[0].(type) {
			case string:
				for i, s := range d {
					d[i] = sanitizer.Sanitize(s.(string))
				}
			case map[string]interface{}:
				for _, t := range d {
					sanitize(t)
				}
			case []interface{}:
				for _, t := range d {
					sanitize(t)
				}
			}
		}
	}
}
