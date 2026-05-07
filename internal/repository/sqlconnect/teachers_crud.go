package sqlconnect

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"
	"strings"
)

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

func addSorting(r *http.Request, query string) string {
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

func addFilters(r *http.Request, query string, args []interface{}) (string, []interface{}) {
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

func GetTeachersDBHandler(teachers []models.Teacher, r *http.Request) ([]models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	firstName := r.URL.Query().Get("first_name")
	lastName := r.URL.Query().Get("last_name")
	query := "SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE 1 = 1"
	var args []interface{}

	query, args = addFilters(r, query, args)

	query = addSorting(r, query)

	if firstName != "" {
		query += " AND first_name = ?"
		args = append(args, firstName)
	}
	if lastName != "" {
		query += " AND last_name = ?"
		args = append(args, lastName)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	// teacherList := make([]models.Teacher, 0)
	for rows.Next() {
		var teacher models.Teacher
		err = rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error retrieving data")
		}
		teachers = append(teachers, teacher)
	}
	return teachers, nil
}

func GetTeacherByID(id int) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Error retrieving data")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	return teacher, nil
}

func AddNewTeachersHandler(newTeachers []models.Teacher) ([]models.Teacher, error) {
	database, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer database.Close()

	// stmt, err := database.Prepare("INSERT INTO teachers (first_name, last_name, email, class, subject) VALUES (?,?,?,?,?)")
	stmt, err := database.Prepare(GenerateInsertQuery(models.Teacher{}))
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer stmt.Close()

	for i, newTeacher := range newTeachers {
		// res, err := stmt.Exec(newTeacher.FirstName, newTeacher.LastName, newTeacher.Email, newTeacher.Class, newTeacher.Subject)
		values := GetStructValues(newTeacher)
		res, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newTeachers[i].ID = int(id)
	}
	return newTeachers, nil
}

func GenerateInsertQuery(model interface{}) string {
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
	fmt.Printf("INSERT INTO teachers (%s) VALUES (%s) \n", columns, placeholders)
	return fmt.Sprintf("INSERT INTO teachers (%s) VALUES (%s)", columns, placeholders)
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

func UpdateTeacher(updatedTeacher models.Teacher, id int) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	defer db.Close()
	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName,
		&existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)

	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Teacher not found")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	_, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", updatedTeacher.FirstName, updatedTeacher.LastName,
		updatedTeacher.Email, updatedTeacher.Class, updatedTeacher.Subject, updatedTeacher.ID)

	updatedTeacher.ID = existingTeacher.ID
	if err != nil {
		log.Println(err)
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingTeacher, nil
}

func PatchTeachers(updates []map[string]interface{}) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}
	defer db.Close()

	tx, err := db.Begin() //transaction
	if err != nil {
		log.Println(err)
		return utils.ErrorHandler(err, "Error updating data")
	}

	for _, update := range updates {
		idFloat, ok := update["id"].(float64)
		id := int(idFloat)

		if !ok {
			log.Println(err)
			tx.Rollback()
			return utils.ErrorHandler(err, "Invalid teacher ID")
		}

		var teacherFromDb models.Teacher
		err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacherFromDb.ID, &teacherFromDb.FirstName,
			&teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Class, &teacherFromDb.Subject)
		log.Println(teacherFromDb.ID)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return utils.ErrorHandler(err, "Teacher not found")
			}
			log.Println(err)
			return utils.ErrorHandler(err, "Error retrieving data")
		}

		teacherVal := reflect.ValueOf(&teacherFromDb).Elem()
		teacherType := teacherVal.Type()

		for key, val := range update {
			if key == "id" {
				continue
			}

			for i := 0; i < teacherVal.NumField(); i++ {
				field := teacherType.Field(i)
				if field.Tag.Get("json") == key+",omitempty" {
					fieldVal := teacherVal.Field(i)
					if fieldVal.CanSet() {
						v := reflect.ValueOf(val)
						if v.Type().ConvertibleTo(fieldVal.Type()) {
							fieldVal.Set(v.Convert(fieldVal.Type()))
						} else {
							tx.Rollback()
							log.Printf("Can not convert %v to %v", v.Type(), fieldVal.Type())
							return utils.ErrorHandler(err, "Error updating data")
						}
						break
					}
				}
			}

			_, err = tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, class = ?, email = ?, subject = ? WHERE id = ?", teacherFromDb.FirstName, teacherFromDb.LastName,
				teacherFromDb.Class, teacherFromDb.Email, teacherFromDb.Subject, teacherFromDb.ID)
			if err != nil {
				tx.Rollback()
				return utils.ErrorHandler(err, "Error updating data")
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return utils.ErrorHandler(err, "Error updating data")
	}

	return nil
}

func PatchOneTeacher(id int, updates map[string]interface{}) (models.Teacher, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	defer db.Close()
	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName,
		&existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)

	if err == sql.ErrNoRows {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	} else if err != nil {
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}

	teacherVal := reflect.ValueOf(&existingTeacher).Elem()
	teacherType := teacherVal.Type()

	for k, v := range updates {
		for i := 0; i < teacherVal.NumField(); i++ {
			field := teacherType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if teacherVal.Field(i).CanSet() {
					fieldVal := teacherVal.Field(i)
					fieldVal.Set(reflect.ValueOf(v).Convert(teacherVal.Field(i).Type()))
				}
			}
		}
	}
	_, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", existingTeacher.FirstName, existingTeacher.LastName,
		existingTeacher.Email, existingTeacher.Class, existingTeacher.Subject, existingTeacher.ID)

	if err != nil {
		log.Println(err)
		return models.Teacher{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingTeacher, nil
}

func DeleteTeachers(ids []int) ([]int, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}

	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}

	stmt, err := tx.Prepare("DELETE FROM teachers WHERE id = ?")
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	defer stmt.Close()

	deletedIDs := []int{}
	for _, id := range ids {
		res, err := stmt.Exec(id)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Error deleting data")
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Error deleting data")
		}
		if rowsAffected > 0 {
			deletedIDs = append(deletedIDs, id)
		}

		if rowsAffected == 0 {
			tx.Rollback()
			return nil, utils.ErrorHandler(err, "Error deleting data")
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return nil, utils.ErrorHandler(err, "Error deleting data")
	}
	return deletedIDs, nil
}

func DeleteOneTeacher(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	defer db.Close()

	result, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "Teacher not found")
	}
	return nil
}
