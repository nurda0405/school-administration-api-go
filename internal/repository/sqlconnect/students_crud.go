package sqlconnect

import (
	"database/sql"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"
)

func GetStudentsDBHandler(students []models.Student, r *http.Request, limit, page int) ([]models.Student, int, error) {
	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return nil, 0, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	firstName := r.URL.Query().Get("first_name")
	lastName := r.URL.Query().Get("last_name")
	query := "SELECT id, first_name, last_name, email, class FROM students WHERE 1 = 1"
	var args []interface{}

	query, args = utils.AddFilters(r, query, args)

	offset := (page - 1) * limit
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	query = utils.AddSorting(r, query)

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
		return nil, 0, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	for rows.Next() {
		var student models.Student
		err = rows.Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
		if err != nil {
			return nil, 0, utils.ErrorHandler(err, "Error retrieving data")
		}
		students = append(students, student)
	}

	var totalStudents int
	err = db.QueryRow("SELECT COUNT(*) FROM students").Scan(&totalStudents)
	if err != nil {
		utils.ErrorHandler(err, "")
		totalStudents = 0
	}
	return students, totalStudents, nil
}

func GetStudentByID(id int) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	var student models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&student.ID, &student.FirstName, &student.LastName, &student.Email, &student.Class)
	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Error retrieving data")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	return student, nil
}

func AddNewStudentsHandler(newStudents []models.Student) ([]models.Student, error) {
	database, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer database.Close()

	stmt, err := database.Prepare(utils.GenerateInsertQuery("students", models.Student{}))
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer stmt.Close()

	for i, newStudent := range newStudents {
		values := utils.GetStructValues(newStudent)
		res, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newStudents[i].ID = int(id)
	}
	return newStudents, nil
}

func UpdateStudent(updatedStudent models.Student, id int) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	defer db.Close()
	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName,
		&existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)

	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Student not found")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	updatedStudent.ID = existingStudent.ID
	_, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", updatedStudent.FirstName, updatedStudent.LastName,
		updatedStudent.Email, updatedStudent.Class, updatedStudent.ID)

	if err != nil {
		log.Println(err)
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	return updatedStudent, nil
}

func PatchStudents(updates []map[string]interface{}) error {
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
			return utils.ErrorHandler(err, "Invalid student ID")
		}

		var studentFromDb models.Student
		err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&studentFromDb.ID, &studentFromDb.FirstName,
			&studentFromDb.LastName, &studentFromDb.Email, &studentFromDb.Class)
		log.Println(studentFromDb.ID)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return utils.ErrorHandler(err, "Student not found")
			}
			log.Println(err)
			return utils.ErrorHandler(err, "Error retrieving data")
		}

		studentVal := reflect.ValueOf(&studentFromDb).Elem()
		studentType := studentVal.Type()

		for key, val := range update {
			if key == "id" {
				continue
			}

			for i := 0; i < studentVal.NumField(); i++ {
				field := studentType.Field(i)
				if field.Tag.Get("json") == key+",omitempty" {
					fieldVal := studentVal.Field(i)
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

			_, err = tx.Exec("UPDATE students SET first_name = ?, last_name = ?, class = ?, email = ? WHERE id = ?", studentFromDb.FirstName, studentFromDb.LastName,
				studentFromDb.Class, studentFromDb.Email, studentFromDb.ID)
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

func PatchOneStudent(id int, updates map[string]interface{}) (models.Student, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	defer db.Close()
	var existingStudent models.Student
	err = db.QueryRow("SELECT id, first_name, last_name, email, class FROM students WHERE id = ?", id).Scan(&existingStudent.ID, &existingStudent.FirstName,
		&existingStudent.LastName, &existingStudent.Email, &existingStudent.Class)

	if err == sql.ErrNoRows {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	} else if err != nil {
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}

	studentVal := reflect.ValueOf(&existingStudent).Elem()
	studentType := studentVal.Type()

	for k, v := range updates {
		for i := 0; i < studentVal.NumField(); i++ {
			field := studentType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if studentVal.Field(i).CanSet() {
					fieldVal := studentVal.Field(i)
					fieldVal.Set(reflect.ValueOf(v).Convert(studentVal.Field(i).Type()))
				}
			}
		}
	}
	_, err = db.Exec("UPDATE students SET first_name = ?, last_name = ?, email = ?, class = ? WHERE id = ?", existingStudent.FirstName, existingStudent.LastName,
		existingStudent.Email, existingStudent.Class, existingStudent.ID)

	if err != nil {
		log.Println(err)
		return models.Student{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingStudent, nil
}

func DeleteStudents(ids []int) ([]int, error) {
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

	stmt, err := tx.Prepare("DELETE FROM students WHERE id = ?")
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

func DeleteOneStudent(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	defer db.Close()

	result, err := db.Exec("DELETE FROM students WHERE id = ?", id)

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "Student not found")
	}
	return nil
}
