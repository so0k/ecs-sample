package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"log"

	"gopkg.in/mgo.v2/bson"

	"github.com/AdRoll/goamz/s3"
	"github.com/so0k/ecs-sample/data"
)

type Upload struct {
	ID      string        `json:"id"`
	ShortID string        `json:"shortID"`
	Content UploadContent `json:"content"`
}

type UploadContent struct {
	URL string `json:"url"`
}

func HandleUploadCreate(w http.ResponseWriter, r *http.Request) {
	f, h, err := r.FormFile("file")
	if err != nil {
		log.Println("Error reading HttpRequest")
		ServeBadRequest(w, r)
		return
	}

	b := bytes.Buffer{}
	n, err := io.Copy(&b, io.LimitReader(f, data.MaxUploadContentSize+10))
	if err != nil {
		log.Println("Error receiving file")
		ServeInternalServerError(w, r)
		return
	}
	if n > data.MaxUploadContentSize {
		log.Println("Max Upload Content Size exceeded")
		ServeBadRequest(w, r)
		return
	}

	id := bson.NewObjectId()
	upl := data.Upload{
		ID:   id,
		Kind: data.Image,
		Content: data.Blob{
			Path: "/uploads/" + id.Hex(),
			Size: n,
		},
	}

	err = data.Bucket.Put(upl.Content.Path, b.Bytes(), h.Header.Get("Content-Type"), s3.Private, s3.Options{})
	if err != nil {
		log.Println("Error storing file to s3: ",err.Error())
		ServeInternalServerError(w, r)
		return
	}

	err = upl.Put()
	if err != nil {
		log.Println("Error storing data in Mongo")
		ServeInternalServerError(w, r)
		return
	}

	res := Upload{
		ID:      upl.ID.Hex(),
		ShortID: upl.ShortID,
		Content: UploadContent{
			URL: upl.Content.SignedURL(),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println("Error Serializing response")
		ServeInternalServerError(w, r)
		return
	}
}

func init() {
	Router.NewRoute().
		Methods("POST").
		Path("/uploads").
		HandlerFunc(HandleUploadCreate)
}
