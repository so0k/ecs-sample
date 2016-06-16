package ui

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/so0k/ecs-sample/data"
)

func ServeUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname, _ := os.Hostname()
	upl, err := data.GetUploadByShortID(vars["shortID"])
	if err != nil {
		ServeInternalServerError(w, r)
		return
	}
	if upl == nil {
		ServeNotFound(w, r)
		return
	}

	err = TplUploadView.Execute(w, TplUploadViewValues{
		Upload: upl,
		Hostname: hostname,
	})
	if err != nil {
		ServeInternalServerError(w, r)
		return
	}
}

func init() {
	Router.NewRoute().
		Methods("GET").
		Path("/u/{shortID}").
		HandlerFunc(ServeUpload)
}
