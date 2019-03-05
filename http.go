package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

const AccessTypeSubscribe = 1
const AccessTypePublish = 2

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

func startBlockingHttpServer(authDb AuthDatabase, port uint16) {
	http.HandleFunc("/auth", func(writer http.ResponseWriter, request *http.Request) {
		authParams := getParamsFromRequest(request)
		if authDb.isAuthenticated(authParams.Username, authParams.Password) {
			writer.WriteHeader(200)
		} else {
			writer.WriteHeader(403)
		}
	})
	http.HandleFunc("/superuser", func(writer http.ResponseWriter, request *http.Request) {
		// system users are superusers
		authParams := getParamsFromRequest(request)
		if authParams.Username == SystemTokenUsername && authParams.Password == authDb.SystemServiceToken {
			writer.WriteHeader(200)
		} else {
			writer.WriteHeader(403)
		}
	})
	http.HandleFunc("/acl", func(writer http.ResponseWriter, request *http.Request) {
		authParams := getParamsFromRequest(request)
		// no one is authorized to write to topics
		if authParams.AccessType == AccessTypeSubscribe && authDb.isAuthorized(authParams.Username, authParams.Password, authParams.Topic) {
			writer.WriteHeader(200)
		} else {
			writer.WriteHeader(403)
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
