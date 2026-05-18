package router

import (
	"net/http"
)

func MainRouter() *http.ServeMux {
	sRouter := studentsRouter()
	tRouter := teachersRouter()

	tRouter.Handle("/", sRouter)
	return tRouter

	// mux := http.NewServeMux()
	// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

	// })
}
