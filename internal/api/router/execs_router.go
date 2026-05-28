package router

import (
	"net/http"
	"restapi/internal/api/handlers"
)

func execsRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /execs", handlers.GetExecsHandler)
	mux.HandleFunc("POST /execs", handlers.AddExecsHandler)
	mux.HandleFunc("PATCH /execs", handlers.PatchExecsHandler)

	mux.HandleFunc("GET /execs/{id}", handlers.GetOneExecHandler)
	mux.HandleFunc("PATCH /execs/{id}", handlers.PatchOneExecHandler)
	mux.HandleFunc("DELETE /execs/{id}", handlers.DeleteOneExecHandler)

	mux.HandleFunc("POST /execs/{id}/updatepassword", handlers.DeleteOneStudentHandler)
	mux.HandleFunc("POST /login", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /logout", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /forgotpassword", handlers.AddStudentsHandler)
	mux.HandleFunc("POST /resetpassword/reset/{resetcode}", handlers.AddStudentsHandler)

	return mux
}
