package router

import (
	"net/http"
)

func MainRouter() *http.ServeMux {
	eRouter := execsRouter()
	sRouter := studentsRouter()
	tRouter := teachersRouter()

	sRouter.Handle("/", eRouter)
	tRouter.Handle("/", sRouter)
	return tRouter

}
