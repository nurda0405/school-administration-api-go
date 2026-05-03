package sqlconnect

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/internal/repository/sqlconnect"
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
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return nil, err
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
		// 	http.Error(w, "Sql query error", http.StatusInternalServerError)
		return nil, err
	}
	defer rows.Close()

	// teacherList := make([]models.Teacher, 0)
	for rows.Next() {
		var teacher models.Teacher
		err = rows.Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
		// if err != nil {
		// 	http.Error(w, "Error scanning database results", http.StatusInternalServerError)
		// 	return
		// }
		teachers = append(teachers, teacher)
	}
	return teachers, nil
}

func GetTeacherByID(id int) (models.Teacher, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// 	http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	defer db.Close()

	var teacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacher.ID, &teacher.FirstName, &teacher.LastName, &teacher.Email, &teacher.Class, &teacher.Subject)
	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return models.Teacher{}, err
	} else if err != nil {
		// http.Error(w,1 "Sql query error", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return teacher, nil
}

func AddNewTeachersHandler(r *http.Request) ([]models.Teacher, error) {
	database, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error in connecting to a database", http.StatusInternalServerError)
		return nil, err
	}
	defer database.Close()

	stmt, err := database.Prepare("INSERT INTO teachers (first_name, last_name, email, class, subject) VALUES (?,?,?,?,?)")
	if err != nil {
		// http.Error(w, "Error in preparing sql statement", http.StatusInternalServerError)
		return nil, err
	}
	defer stmt.Close()

	var newTeachers []models.Teacher
	err = json.NewDecoder(r.Body).Decode(&newTeachers)
	if err != nil {
		// http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return nil, err
	}
	for i, newTeacher := range newTeachers {
		res, err := stmt.Exec(newTeacher.FirstName, newTeacher.LastName, newTeacher.Email, newTeacher.Class, newTeacher.Subject)
		if err != nil {
			// http.Error(w, "Error in executing sql statement", http.StatusInternalServerError)
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			// http.Error(w, "Error in getting the id of the last object", http.StatusInternalServerError)
			return nil, err
		}
		newTeachers[i].ID = int(id)
	}
	return newTeachers, nil
}

func UpdateTeacher(updatedTeacher models.Teacher, id int) (models.Teacher, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error in connecting to database", http.StatusInternalServerError)
		return models.Teacher{}, err
	}

	defer db.Close()
	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName,
		&existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)

	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusBadRequest)
		return models.Teacher{}, err
	} else if err != nil {
		// http.Error(w, "Unable to retrieve data", http.StatusInternalServerError)
	}

	_, err = db.Exec("UPDATE teachers SET first_name = ?, last_name = ?, email = ?, class = ?, subject = ? WHERE id = ?", updatedTeacher.FirstName, updatedTeacher.LastName,
		updatedTeacher.Email, updatedTeacher.Class, updatedTeacher.Subject, updatedTeacher.ID)

	updatedTeacher.ID = existingTeacher.ID
	if err != nil {
		log.Println(err)
		// http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return existingTeacher, nil
}

func PatchTeachers(updates []map[string]interface{}) error {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error in connecting to database", http.StatusInternalServerError)
		return err
	}
	defer db.Close()

	tx, err := db.Begin() //transaction
	if err != nil {
		log.Println(err)
		// http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return err
	}

	for _, update := range updates {
		idFloat, ok := update["id"].(float64)
		id := int(idFloat)

		if !ok {
			log.Println(err)
			tx.Rollback()
			// http.Error(w, "Invalid teacher ID in update", http.StatusBadRequest)
			return err
		}

		// id, err := strconv.Atoi(idStr)
		// if err != nil {
		// 	tx.Rollback()
		// 	http.Error(w, "Error converting ID to int", http.StatusBadRequest)
		// 	return
		// }

		var teacherFromDb models.Teacher
		err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&teacherFromDb.ID, &teacherFromDb.FirstName,
			&teacherFromDb.LastName, &teacherFromDb.Email, &teacherFromDb.Class, &teacherFromDb.Subject)
		log.Println(teacherFromDb.ID)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				// http.Error(w, "Teacher not found", http.StatusNotFound)
				return err
			}
			log.Println(err)
			// http.Error(w, "Error retrivieng teacher", http.StatusInternalServerError)
			return err
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
							return fmt.Errorf("Can not convert %v to %v", v.Type(), fieldVal.Type())
						}
						break
					}
				}
			}

			_, err = tx.Exec("UPDATE teachers SET first_name = ?, last_name = ?, class = ?, email = ?, subject = ? WHERE id = ?", teacherFromDb.FirstName, teacherFromDb.LastName,
				teacherFromDb.Class, teacherFromDb.Email, teacherFromDb.Subject, teacherFromDb.ID)
			if err != nil {
				tx.Rollback()
				// http.Error(w, "Error updating teacher", http.StatusInternalServerError)
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		// http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		return err
	}

	return nil
}

func PatchOneTeacher(id int, updates map[string]interface{}) (models.Teacher, error) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error in connecting to database", http.StatusInternalServerError)
		return models.Teacher{}, err
	}

	defer db.Close()
	var existingTeacher models.Teacher
	err = db.QueryRow("SELECT id, first_name, last_name, email, class, subject FROM teachers WHERE id = ?", id).Scan(&existingTeacher.ID, &existingTeacher.FirstName,
		&existingTeacher.LastName, &existingTeacher.Email, &existingTeacher.Class, &existingTeacher.Subject)

	if err == sql.ErrNoRows {
		// http.Error(w, "Teacher not found", http.StatusBadRequest)
		return models.Teacher{}, err
	} else if err != nil {
		// http.Error(w, "Unable to retrieve data", http.StatusInternalServerError)
		return models.Teacher{}, err
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
		// http.Error(w, "Error updating teacher", http.StatusInternalServerError)
		return models.Teacher{}, err
	}
	return existingTeacher, nil
}

func DeleteOneTeacher(id int) error {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		// http.Error(w, "Error in connecting to database", http.StatusInternalServerError)
		return err
	}

	defer db.Close()

	result, err := db.Exec("DELETE FROM teachers WHERE id = ?", id)

	if err != nil {
		// http.Error(w, "Error deleting teacher", http.StatusInternalServerError)
		return err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		// http.Error(w, "Error retrieving result", http.StatusInternalServerError)
		return err
	}

	if rowsAffected == 0 {
		// http.Error(w, "Teacher not found", http.StatusNotFound)
		return err
	}
	return nil
}
