package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"
	"fmt"

	"github.com/desertbit/glue"
	"github.com/so0k/ecs-sample/api"
	"github.com/so0k/ecs-sample/data"
	"github.com/so0k/ecs-sample/hub"
	"github.com/so0k/ecs-sample/ui"
)

func main() {

	var err error
	//parse command line arguments
	fEnvFile := flag.String("env-file", "", "path to environment file")
	flag.Parse()

	if *fEnvFile != "" {
		err = LoadEnvFile(*fEnvFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	//try connecting to S3
	log.Println("Initializing S3...")
	err = data.InitBucket(os.Getenv("S3_BUCKET_NAME"))
	if err != nil {
		log.Fatal(err)
	}

	maxAttempts := 20
	//try connecting to mongodb with exponential back-off
	for attempts :=1; attempts <= maxAttempts; attempts++ {
		log.Println(fmt.Sprintf("Connecting to mongodb (%d/%d)...",attempts,maxAttempts))
		err = data.OpenSession(os.Getenv("MONGO_URL"))
		if err == nil {
			break
		}
		log.Println(err.Error() + ", sleeping...")
		time.Sleep(time.Duration(attempts)*time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}

	//try connecting to redis with exponential back-off
	for attempts :=1; attempts <= maxAttempts; attempts++ {
		log.Println(fmt.Sprintf("Connecting to redis (%d/%d)...",attempts,maxAttempts))
		err = hub.Connect(os.Getenv("REDIS_URL"))
		if err == nil {
			break
		}
		log.Println(err.Error() + ", sleeping...")
		time.Sleep(time.Duration(attempts)*time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}
	//init the Hub
	hub.InitHub()

	glueSrv := glue.NewServer(glue.Options{
		HTTPSocketType: glue.HTTPSocketTypeNone,
	})
	glueSrv.OnNewSocket(hub.HandleSocket)

	http.Handle("/", ui.Router)
	http.Handle("/api/", http.StripPrefix("/api", api.Router))
	http.Handle("/assets/", http.StripPrefix("/assets", ui.AssetsFS))
	http.Handle("/glue/", glueSrv)

	port := os.Getenv("PORT")
	//if PORT is blank, use 80 as default
	if  port == ""{
		port = "80"
	}

	log.Printf("Listening on :%s", port)
	err = http.ListenAndServe("0.0.0.0:"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
