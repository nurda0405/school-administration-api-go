package router

import (
	"net/http"
	"restapi/internal/api/handlers"
)

func execsRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /execs", handlers.GetStudentsHandler)
	mux.HandleFunc("POST /execs", handlers.AddStudentsHandler)
	mux.HandleFunc("PATCH /execs", handlers.PatchStudentsHandler)

	mux.HandleFunc("GET /execs/{id}", handlers.GetOneStudentHandler)
	mux.HandleFunc("PATCH /execs/{id}", handlers.PatchOneStudentHandler)
	mux.HandleFunc("DELETE /execs/{id}", handlers.DeleteOneStudentHandler)
	mux.HandleFunc("POST /execs/{id}/updatepassword", handlers.DeleteOneStudentHandler)

	mux.HandleFunc("POST /login", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /logout", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /forgotpassword", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /resetpassword/reset/{resetcode}", handlers.AddStudentsHandler)

	return mux
}
