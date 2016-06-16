package api

import (
	"net/http"
    "log"
)

func ServeBadRequest(w http.ResponseWriter, r *http.Request) {
    log.Println("serving 400")
	http.Error(w, "Bad Request", http.StatusBadRequest)
}

func ServeNotFound(w http.ResponseWriter, r *http.Request) {
    log.Println("serving 404")
	http.Error(w, "Not Found", http.StatusNotFound)
}

func ServeInternalServerError(w http.ResponseWriter, r *http.Request) {
    log.Println("serving 500")
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
