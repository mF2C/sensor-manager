package main

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"strconv"
	"time"
)

func runTestPublish(mqttClient mqtt.Client) {
	log.Println("Starting in testpublish mode.")
	publishMessagesIndefinitely(mqttClient, "testpublish", 1*time.Second)
}

func runProduction(mqttClient mqtt.Client, authDatabase AuthDatabase, httpServerPort uint16) {
	log.Println("Starting in production mode.")
	startMessageTransformations(mqttClient, authDatabase)
	startBlockingHttpServer(authDatabase, httpServerPort)
}

func getEnvMandatoryString(envName string) string {
	value, exists := os.LookupEnv(envName)
	if !exists {
		panic(fmt.Errorf("environment variable %s is mandatory", envName))
	} else {
		log.Printf("Read env var %s as %s.", envName, value)
		return value
	}
}

func getEnvMandatoryInt(envName string) int {
	stringVal := getEnvMandatoryString(envName)
	intVal, err := strconv.Atoi(stringVal)
	if err != nil {
		panic(fmt.Errorf("could not parse integer from %s for env var %s", stringVal, envName))
	}
	return intVal
}

func main() {
	testpublish := flag.Bool("testpublish", false, "Test mode: publish messages to MQTT indefinitely.")
	flag.Parse()

	mqttHost := getEnvMandatoryString("MQTT_HOST")
	mqttPort := getEnvMandatoryInt("MQTT_PORT")
	sensorManagerClientId := getEnvMandatoryString("CLIENT_ID")
	httpServerPort := getEnvMandatoryInt("HTTP_PORT")
	authDatabaseFilename := getEnvMandatoryString("AUTH_DB_FILE")

	mqttAddress := fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort)

	authDatabase := loadOrCreateAuthDatabase(authDatabaseFilename)
	mqttClient := connectMqttClient(mqttAddress, sensorManagerClientId, authDatabase.SystemServiceToken)

	if *testpublish {
		runTestPublish(mqttClient)
	} else {
		runProduction(mqttClient, authDatabase, uint16(httpServerPort))
	}
}
