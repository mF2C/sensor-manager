package sensormanager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

const MqttAuthAccessTypeSubscribe = 1
const MqttAuthAccessTypePublish = 2

type MqttAuthParams struct {
	ClientId   string
	Username   string
	Password   string
	Topic      string
	AccessType int
}

func getParamsFromRequest(req *http.Request) MqttAuthParams {
	accessTypeString := req.PostFormValue("acc")
	accessType := -1
	if accessTypeString != "" {
		accessTypeWhyGoWhy, err := strconv.Atoi(accessTypeString)
		accessType = accessTypeWhyGoWhy
		if err != nil {
			log.Println(fmt.Errorf("cannot convert access type '%s' to int, defaulting to -1", accessTypeString))
			accessType = -1
		}
	}

	return MqttAuthParams{
		ClientId:   req.PostFormValue("clientid"),
		Username:   req.PostFormValue("username"),
		Password:   req.PostFormValue("password"),
		Topic:      req.PostFormValue("topic"),
		AccessType: accessType,
	}
}

func StartBlockingHttpServer(wg *sync.WaitGroup, authDb *AuthDatabase, port uint16) {
	http.HandleFunc("/auth", func(writer http.ResponseWriter, request *http.Request) {
		authParams := getParamsFromRequest(request)
		if authDb.isAuthenticated(authParams.Username, authParams.Password) {
			writer.WriteHeader(200)
			log.Printf("/auth (200) -> %+v", authParams)
		} else {
			writer.WriteHeader(403)
			log.Printf("/auth (403) -> %+v", authParams)
		}
	})
	http.HandleFunc("/superuser", func(writer http.ResponseWriter, request *http.Request) {
		// system users are superusers
		authParams := getParamsFromRequest(request)
		if authDb.isSuperuserPreauthenticated(authParams.Username) {
			writer.WriteHeader(200)
			log.Printf("/superuser (200) -> %+v", authParams)
		} else {
			writer.WriteHeader(403)
			log.Printf("/superuser (403) -> %+v", authParams)
		}
	})
	http.HandleFunc("/acl", func(writer http.ResponseWriter, request *http.Request) {
		authParams := getParamsFromRequest(request)
		if authDb.isAuthorized(authParams.Username, authParams.Topic, authParams.AccessType) {
			writer.WriteHeader(200)
			log.Printf("/acl (200) -> %+v", authParams)
		} else {
			writer.WriteHeader(403)
			log.Printf("/acl (403) -> %+v", authParams)
		}
	})
	// TODO: this returns everything to everyone, needs auth through cimi
	http.HandleFunc("/topics", func(writer http.ResponseWriter, request *http.Request) {
		serialized, err := json.Marshal(authDb.Topics)
		if err != nil {
			panic(err)
		}
		_, err = writer.Write(serialized)
		if err != nil {
			panic(err)
		}
	})
	defer wg.Done()
	log.Printf("Starting HTTP server on port %d.", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
