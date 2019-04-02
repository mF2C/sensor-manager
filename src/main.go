package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"hash/crc64"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

func runSensorSimulator(mqttHost string, mqttPort uint16, sensorDriverPassword string) {
	log.Println("Starting in sensor simulation mode.")
	mqttClient := connectMqttClient(fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort), "sensor-simulator", SensorDriverUsername, sensorDriverPassword)
	publishMessagesIndefinitely(mqttClient, TopicSensorReceive, 1*time.Second)
}

func runProduction(mqttHost string, mqttPort uint16, cimiTraefikHost string, cimiTraefikPort uint16, lifecycleHost string, lifecyclePort uint16, authDatabase AuthDatabase, httpServerPort uint16, sensorCheckIntervalSeconds uint) {
	log.Println("Starting in production mode.")
	// the server needs to start beforehand, as message transformations connect to MQTT and thus require auth
	wg := sync.WaitGroup{}
	wg.Add(3)
	go startBlockingHttpServer(&wg, &authDatabase, httpServerPort)
	mqttClient := connectMqttClient(fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort), "sensor-manager", SuperuserUsername, authDatabase.AdministratorAccessToken)
	go startMessageTransformations(&wg, &authDatabase, mqttClient)
	go startContainerManager(&wg, cimiTraefikHost, cimiTraefikPort, lifecycleHost, lifecyclePort, &authDatabase, sensorCheckIntervalSeconds)
	wg.Wait()
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
	sensorDriverAccessToken := getEnvMandatoryString("SENSOR_DRIVER_ACCESS_TOKEN")

	if *simulateSensor {
		runSensorSimulator(mqttHost, uint16(mqttPort), sensorDriverAccessToken)
	} else {
		httpServerPort := getEnvMandatoryInt("HTTP_PORT")
		authDatabaseFilename := getEnvMandatoryString("AUTH_DB_FILE")
		administratorAccessToken := getEnvMandatoryString("ADMINISTRATOR_ACCESS_TOKEN")
		applicationSecret := getEnvMandatoryString("APPLICATION_SECRET")
		cimiHost := getEnvMandatoryString("CIMI_HOST")
		cimiPort := getEnvMandatoryInt("CIMI_PORT")
		lifecycleHost := getEnvMandatoryString("LIFECYCLE_HOST")
		lifecyclePort := getEnvMandatoryInt("LIFECYCLE_PORT")
		sensorsCheckIntervalSeconds := getEnvMandatoryInt("SENSORS_CHECK_INTERVAL_SECONDS")

		rand.Seed(int64(crc64.Checksum([]byte(applicationSecret), crc64.MakeTable(crc64.ECMA))))
		authDatabase := loadOrCreateAuthDatabase(authDatabaseFilename, administratorAccessToken, sensorDriverAccessToken)

		runProduction(mqttHost, uint16(mqttPort), cimiHost, uint16(cimiPort), lifecycleHost, uint16(lifecyclePort), authDatabase, uint16(httpServerPort), uint(sensorsCheckIntervalSeconds))
	}
}
