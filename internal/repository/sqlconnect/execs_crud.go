package sqlconnect

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"

	"golang.org/x/crypto/argon2"
)

func GetExecsDBHandler(execs []models.Exec, r *http.Request) ([]models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		// http.Error(w, "Error connecting to database", http.StatusInternalServerError)
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	firstName := r.URL.Query().Get("first_name")
	lastName := r.URL.Query().Get("last_name")
	query := "SELECT id, first_name, last_name, email, username, user_created_at, inactive_status, role FROM execs WHERE 1 = 1"
	var args []interface{}

	query, args = utils.AddFilters(r, query, args)

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
		return nil, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer rows.Close()

	for rows.Next() {
		var exec models.Exec
		err = rows.Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.InactiveStatus, &exec.Role)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error retrieving data")
		}
		execs = append(execs, exec)
	}
	return execs, nil
}

func GetExecByID(id int) (models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	defer db.Close()

	var exec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, user_created_at, inactive_status, role FROM execs WHERE id = ?", id).Scan(&exec.ID, &exec.FirstName, &exec.LastName, &exec.Email, &exec.Username, &exec.UserCreatedAt, &exec.InactiveStatus, &exec.Role)
	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Error retrieving data")
	} else if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error retrieving data")
	}
	return exec, nil
}

func AddNewExecsHandler(newExecs []models.Exec) ([]models.Exec, error) {
	database, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer database.Close()

	stmt, err := database.Prepare(utils.GenerateInsertQuery("execs", models.Exec{}))
	if err != nil {
		return nil, utils.ErrorHandler(err, "Error adding data")
	}
	defer stmt.Close()

	for i, newExec := range newExecs {
		if newExec.Password == "" {
			return nil, utils.ErrorHandler(errors.New("password is blank"), "please enter password")
		}
		salt := make([]byte, 16)
		_, err := rand.Read(salt)

		if err != nil {
			return nil, utils.ErrorHandler(errors.New("failed to generate salt"), "error adding data")
		}

		hash := argon2.IDKey([]byte(newExec.Password), salt, 1, 64*1024, 4, 32)
		saltBase64 := base64.StdEncoding.EncodeToString(salt)
		hashBase64 := base64.StdEncoding.EncodeToString(hash)

		encodedHash := fmt.Sprintf("%s.%s", saltBase64, hashBase64)
		newExec.Password = encodedHash
		newExecs[i].Password = encodedHash

		values := utils.GetStructValues(newExec)
		res, err := stmt.Exec(values...)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, utils.ErrorHandler(err, "Error adding data")
		}
		newExecs[i].ID = int(id)
	}
	return newExecs, nil
}

func PatchExecs(updates []map[string]interface{}) error {
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
			return utils.ErrorHandler(err, "Invalid exec ID")
		}

		var execFromDb models.Exec
		err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id = ?", id).Scan(&execFromDb.ID, &execFromDb.FirstName,
			&execFromDb.LastName, &execFromDb.Email, &execFromDb.Username)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return utils.ErrorHandler(err, "Exec not found")
			}
			log.Println(err)
			return utils.ErrorHandler(err, "Error retrieving data")
		}

		execVal := reflect.ValueOf(&execFromDb).Elem()
		execType := execVal.Type()

		for key, val := range update {
			if key == "id" {
				continue
			}

			for i := 0; i < execVal.NumField(); i++ {
				field := execType.Field(i)
				if field.Tag.Get("json") == key+",omitempty" {
					fieldVal := execVal.Field(i)
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

			_, err = tx.Exec("UPDATE execs SET first_name = ?, last_name = ?, email = ?, username = ? WHERE id = ?", execFromDb.FirstName, execFromDb.LastName,
				execFromDb.Email, execFromDb.Username, execFromDb.ID)

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

func PatchOneExec(id int, updates map[string]interface{}) (models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}

	defer db.Close()
	var existingExec models.Exec
	err = db.QueryRow("SELECT id, first_name, last_name, email, username FROM execs WHERE id = ?", id).Scan(&existingExec.ID, &existingExec.FirstName,
		&existingExec.LastName, &existingExec.Email, &existingExec.Username)

	if err == sql.ErrNoRows {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	} else if err != nil {
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}

	execVal := reflect.ValueOf(&existingExec).Elem()
	execType := execVal.Type()

	for k, v := range updates {
		for i := 0; i < execVal.NumField(); i++ {
			field := execType.Field(i)
			if field.Tag.Get("json") == k+",omitempty" {
				if execVal.Field(i).CanSet() {
					fieldVal := execVal.Field(i)
					fieldVal.Set(reflect.ValueOf(v).Convert(execVal.Field(i).Type()))
				}
			}
		}
	}
	_, err = db.Exec("UPDATE execs SET first_name = ?, last_name = ?, email = ?, username = ? WHERE id = ?", existingExec.FirstName, existingExec.LastName,
		existingExec.Email, existingExec.Username, existingExec.ID)

	if err != nil {
		log.Println(err)
		return models.Exec{}, utils.ErrorHandler(err, "Error updating data")
	}
	return existingExec, nil
}

func DeleteOneExec(id int) error {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	defer db.Close()

	result, err := db.Exec("DELETE FROM execs WHERE id = ?", id)

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return utils.ErrorHandler(err, "Error deleting data")
	}

	if rowsAffected == 0 {
		return utils.ErrorHandler(err, "Exec not found")
	}
	return nil
}
