package utils

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
)

func GenerateInsertQuery(tableName string, model interface{}) string {
	modelType := reflect.TypeOf(model)
	var columns, placeholders string

	for i := 0; i < modelType.NumField(); i++ {
		dbTag := modelType.Field(i).Tag.Get("db")
		fmt.Println("dbTag", dbTag)
		dbTag = strings.TrimSuffix(dbTag, ",omitempty")
		if dbTag == "id" || dbTag == "" {
			continue
		}
		if columns != "" {
			columns += ", "
			placeholders += ", "
		}
		columns += dbTag
		placeholders += "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, columns, placeholders)
}

func GetStructValues(model interface{}) []interface{} {
	modelValue := reflect.ValueOf(model)
	modelType := modelValue.Type()
	values := []interface{}{}
	for i := 0; i < modelType.NumField(); i++ {
		dbTag := modelType.Field(i).Tag.Get("db")
		dbTag = strings.TrimSuffix(dbTag, ",omitempty")
		if dbTag != "" && dbTag != "id" {
			values = append(values, modelValue.Field(i).Interface())
		}
	}
	log.Println("Values: ", values)
	return values
}

func AddSorting(r *http.Request, query string) string {
	sortStr := ""
	sortParams := r.URL.Query()["sortby"]
	if len(sortParams) > 0 {
		for i, param := range sortParams {
			parts := strings.Split(param, ":")
			if len(parts) != 2 {
				continue
			}
			field, order := parts[0], parts[1]
			if isValidField(field) && isValidOrder(order) {
				if i > 0 {
					query += ","
				}
				sortStr += " " + field + " " + order
			}
		}
	}
	if sortStr != "" {
		query += " ORDER BY" + sortStr
	}
	return query
}

func AddFilters(r *http.Request, query string, args []interface{}) (string, []interface{}) {
	params := map[string]string{
		"id":         "id",
		"first_name": "first_name",
		"last_name":  "last_name",
		"email":      "email",
		"class":      "class",
		"subject":    "subject",
	}

	for param, _ := range params {
		value := r.URL.Query().Get(param)

		if value != "" {
			query += " AND " + param + "=?"
			args = append(args, value)
		}
	}
	return query, args
}

func isValidField(field string) bool {
	fields := map[string]bool{
		"id":         true,
		"first_name": true,
		"last_name":  true,
		"email":      true,
		"class":      true,
		"subject":    true,
	}
	_, exists := fields[field]
	return exists
}

func isValidOrder(order string) bool {
	return order == "asc" || order == "desc"
}
