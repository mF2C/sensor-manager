package main

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	flag "github.com/spf13/pflag"
	"hash/crc64"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func runSensorSimulator(mqttClient mqtt.Client) {
	log.Println("Starting in sensor simulation mode.")
	publishMessagesIndefinitely(mqttClient, "su", 1*time.Second)
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
	simulateSensor := flag.Bool("simulate-sensor", false, "Test mode: sensor simulation.")
	flag.Parse()

	mqttHost := getEnvMandatoryString("MQTT_HOST")
	mqttPort := getEnvMandatoryInt("MQTT_PORT")

	mqttAddress := fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort)

	if *simulateSensor {
		mqttClient := connectMqttClient(mqttAddress, "sensor-simulator", "", "")
		runSensorSimulator(mqttClient)
	} else {
		httpServerPort := getEnvMandatoryInt("HTTP_PORT")
		authDatabaseFilename := getEnvMandatoryString("AUTH_DB_FILE")
		administratorAccessToken := getEnvMandatoryString("ADMINISTRATOR_ACCESS_TOKEN")
		applicationSecret := getEnvMandatoryString("APPLICATION_SECRET")

		rand.Seed(int64(crc64.Checksum([]byte(applicationSecret), crc64.MakeTable(crc64.ECMA))))
		authDatabase := loadOrCreateAuthDatabase(authDatabaseFilename, administratorAccessToken)
		mqttClient := connectMqttClient(mqttAddress, "sensor-manager", SystemTokenUsername, authDatabase.AdministratorAccessToken)

		runProduction(mqttClient, authDatabase, uint16(httpServerPort))
	}
}
