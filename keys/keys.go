package keys

import (
	_"fmt"
	"strconv"
	"bytes"
	"github.com/alexanderbartels/flux"
)

var (
	allowedRequestParams = []string{
		"width",
		"height",
		"dpi",
	}
	nameParamSep         = "?"
	paramSep             = "&"
	assigment            = "="

	paramValidationFuncs = map[string]func(string) bool {
		"width": validatePositiveInteger,
		"height": validatePositiveInteger,
		"dpi": validatePositiveInteger,
	}
)

func validatePositiveInteger(value string) bool {
	valAsInt, err := strconv.Atoi(value)
	if err != nil || valAsInt < 0 {
		return false
	}
	return true
}

func validateParam(param, value string) bool {
	return paramValidationFuncs[param](value)
}

func getValueByName(params map[string]string, name string) string {
	if val, ok := params[name]; ok {
		if ok = validateParam(name, val); ok {
			// if param is available and valid, we can use it
			return val
		}
	}

	// we use always "0" as default value,
	// because every cache GetterFunc can have its own default values
	//
	// But all allowed Params should be in key to minimize cache entries
	return "0"
}

// creates a Key for the cache like an URL,
// but only the allowed request params are included
// example: test.jpg?width=500&height=300
func Generate(fileName string, params map[string]string) string {
	var buffer bytes.Buffer

	buffer.WriteString(fileName)
	buffer.WriteString(nameParamSep)

	for i, name := range allowedRequestParams {
		if i > 0 {
			buffer.WriteString(paramSep)
		}

		buffer.WriteString(name)
		buffer.WriteString(assigment)
		buffer.WriteString(getValueByName(params, name))
	}

	return buffer.String()
}

func Parse(key string) (fileName string, params map[string]string) {
	// TODO regex to parse the key efficiently
}
