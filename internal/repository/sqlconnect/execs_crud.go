package sqlconnect

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"restapi/internal/models"
	"restapi/pkg/utils"
	"strconv"
	"time"

	"github.com/go-mail/mail/v2"
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
		encodedHash, err, _ := utils.HashPassword(newExec.Password)
		if err != nil {
			return nil, err
		}

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

func GetUserByUsername(username string) (*models.Exec, error) {
	db, err := ConnectDB()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal server error")
	}
	defer db.Close()

	user := &models.Exec{}
	err = db.QueryRow("SELECT id, first_name, last_name, email, username, password, inactive_status, role FROM execs WHERE username = ?",
		username).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Username, &user.Password, &user.InactiveStatus, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.ErrorHandler(err, "User not found")
		}
		return nil, utils.ErrorHandler(err, "Database error")
	}
	return user, nil
}

func UpdatePasswordInDb(userID int, currentPassword, updatedPassword string) (error, int) {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Internal server error"), http.StatusInternalServerError
	}

	var username string
	var userPassword string
	var role string

	err = db.QueryRow("SELECT username, password, role FROM execs WHERE id = ?", userID).Scan(&username, &userPassword, &role)
	if err != nil {
		return utils.ErrorHandler(err, "User not found"), http.StatusBadRequest
	}

	err = utils.VerifyPassword(currentPassword, userPassword)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	hashedPassword, err, statusCode := utils.HashPassword(updatedPassword)
	if err != nil {
		return err, statusCode
	}

	currentTime := time.Now().Format(time.RFC3339)
	_, err = db.Exec("UPDATE execs SET password = ?, password_changed_at = ? WHERE id = ?", hashedPassword, currentTime, userID)
	if err != nil {
		return errors.New("Error updating password"), http.StatusInternalServerError
	}
	return nil, http.StatusAccepted
}

func ForgotPasswordDbHandler(email string) (error, int) {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Internal Server Error"), http.StatusInternalServerError
	}
	defer db.Close()

	var exec models.Exec
	err = db.QueryRow("SELECT id FROM execs WHERE email = ?", email).Scan(&exec.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("Not found"), http.StatusNotFound
		}
		return utils.ErrorHandler(err, "Internal server error"), http.StatusInternalServerError
	}

	duration, err := strconv.Atoi(os.Getenv("RESET_TOKEN_EXP_DURATION"))
	if err != nil {
		return utils.ErrorHandler(err, "Failed to send password reset email"), http.StatusInternalServerError
	}

	mins := time.Duration(duration)
	expiry := time.Now().Add(mins * time.Minute).Format(time.RFC3339)

	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return utils.ErrorHandler(err, "Failed to send password reset email"), http.StatusInternalServerError
	}
	token := hex.EncodeToString(tokenBytes)
	hashedToken := sha256.Sum256(tokenBytes)
	hashedTokenString := hex.EncodeToString(hashedToken[:])

	_, err = db.Exec("UPDATE execs SET password_reset_code = ?, password_code_expires = ? WHERE id = ?", hashedTokenString, expiry, exec.ID)
	if err != nil {
		return utils.ErrorHandler(err, "Internal error"), http.StatusInternalServerError
	}

	resetURL := fmt.Sprintf("https://localhost:3000/execs/resetpassword/reset/%s", token)
	message := fmt.Sprintf(
		`<p>You can reset your password via this link:</p> <a href="%s">%s</a> <p>If you didn't request a password reset, ignore this email. This link is only valid for %v minutes.</p>`, resetURL, resetURL, duration)

	m := mail.NewMessage()
	m.SetHeader("From", "schooladmin@school.com")
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Your password reset link")
	m.SetHeader("html", message)

	d := mail.NewDialer("localhost", 1025, "", "")
	err = d.DialAndSend(m)
	if err != nil {
		return utils.ErrorHandler(err, "Failed to send password reset email"), http.StatusInternalServerError
	}
	return nil, 0
}

func ResetPasswordDbHandler(hashedTokenString string, newPassword string) (error, int) {
	db, err := ConnectDB()
	if err != nil {
		return utils.ErrorHandler(err, "Internal server error"), http.StatusInternalServerError
	}
	defer db.Close()

	var user models.Exec
	query := "SELECT id, email FROM execs WHERE password_reset_code = ? AND password_code_expires > ?"
	err = db.QueryRow(query, hashedTokenString, time.Now().Format(time.RFC3339)).Scan(&user.ID, &user.Email)
	if err != nil {
		return errors.New("Password reset code is incorrect or expired"), http.StatusBadRequest
	}

	hashedPassword, err, statusCode := utils.HashPassword(newPassword)
	if err != nil {
		return err, statusCode
	}

	updateQuery := "UPDATE execs SET password = ?, password_reset_code = NULL, password_code_expires = NULL, password_changed_at = ?"
	_, err = db.Exec(updateQuery, hashedPassword, time.Now().Format(time.RFC3339))
	if err != nil {
		return utils.ErrorHandler(err, "Internal server error"), http.StatusInternalServerError
	}
	return nil, 0
}
