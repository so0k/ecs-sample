package ui

import (
	"net/http"
	"os"
)

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	err := TplIndex.Execute(w, TplIndexValues{hostname})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func init() {
	Router.NewRoute().
		Methods("GET").
		Path("/").
		HandlerFunc(ServeIndex)
}
