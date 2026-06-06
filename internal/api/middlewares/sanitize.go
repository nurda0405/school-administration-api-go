package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"restapi/pkg/utils"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

func XSSMiddleware(next http.Handler) http.Handler {
	fmt.Println("--------Inside XSSMiddleware----------")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Original Path: ", r.URL.Path)
		sanitizedPath, err := clean(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.URL.Path = sanitizedPath.(string)
		fmt.Println("Sanitized Path: ", r.URL.Path)

		fmt.Println("Original Query: ", r.URL.Query())
		params := r.URL.Query()
		sanitizedQuery := make(map[string][]string)
		for k, v := range params {
			sanitizedKey, err := clean(k)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var sanitizedVals []string
			for _, val := range v {
				sanitizedVal, err := clean(val)
				if err != nil {
					http.Error(w, "Invalid query values", http.StatusBadRequest)
					return
				}
				sanitizedVals = append(sanitizedVals, sanitizedVal.(string))

			}
			sanitizedQuery[sanitizedKey.(string)] = sanitizedVals
		}

		r.URL.Path = sanitizedPath.(string)
		r.URL.RawQuery = url.Values(sanitizedQuery).Encode()
		fmt.Println("Sanitized Query: ", r.URL.Query())

		if r.Header.Get("Content-Type") == "application/json" {
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, utils.ErrorHandler(err, "Error reading request body").Error(), http.StatusBadRequest)
					return
				}
				fmt.Println("Original Body: ", string(bodyBytes))

				bodyString := strings.TrimSpace(string(bodyBytes))
				r.Body = io.NopCloser(bytes.NewReader([]byte(bodyString)))

				if len(bodyString) > 0 {
					var inputData interface{}
					err = json.NewDecoder(bytes.NewReader([]byte(bodyString))).Decode(&inputData)
					if err != nil {
						http.Error(w, "Invalid JSON body", http.StatusBadRequest)
						return
					}
					sanitizedData, err := clean(inputData)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					sanitizedBody, err := json.Marshal(sanitizedData)
					if err != nil {
						http.Error(w, "Error sanitizing body", http.StatusBadRequest)
						return
					}

					r.Body = io.NopCloser(bytes.NewReader([]byte(sanitizedBody)))
					fmt.Println("Sanitized Body: ", string(sanitizedBody))
				}
			}
		} else if r.Header.Get("Content-Type") != "" {
			http.Error(w, "Expected application/json Content-Type", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clean(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			v[k] = sanitizeValue(val)
		}
		return v, nil
	case []interface{}:
		for i, val := range v {
			v[i] = sanitizeValue(val)
		}
		return v, nil
	case string:
		return sanitizeString(v), nil
	default:
		return nil, fmt.Errorf("Unsupported value type: %T", data)
	}
}

func sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for k, val := range v {
			v[k] = sanitizeValue(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = sanitizeValue(val)
		}
		return v
	case string:
		return sanitizeString(v)
	default:
		return v
	}
}
func sanitizeString(value string) string {
	return bluemonday.UGCPolicy().Sanitize(value)
}
