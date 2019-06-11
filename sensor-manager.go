package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"hash/crc64"
	"log"
	"math/rand"
	sensormanager "mf2c-sensor-manager/sensor-manager"
	"os"
	"strconv"
	"sync"
	"time"
)

func runSensorSimulator(mqttHost string, mqttPort uint16, sensorDriverPassword string) {
	log.Println("Starting in sensor simulation mode.")
	mqttClient := sensormanager.ConnectMqttClient(fmt.Sprintf("ws://%s:%d", mqttHost, mqttPort), "sensor-simulator", sensormanager.SensorDriverUsername, sensorDriverPassword)
	sensormanager.PublishMessagesIndefinitely(mqttClient, sensormanager.TopicSensorReceive, 1*time.Second)
}

func runProduction(mqttHost string, mqttPort uint16, cimiTraefikHost string, cimiTraefikPort uint16, lifecycleHost string, lifecyclePort uint16,
	authDatabase sensormanager.AuthDatabase, httpServerPort uint16, sensorCheckIntervalSeconds uint, sensorContainerMapFilename string, sensorDriverDockerNetworkName string, mqttPathSuffix string) {
	log.Println("Starting in production mode.")
	// the server needs to start beforehand, as message transformations connect to MQTT and thus require auth
	wg := sync.WaitGroup{}
	wg.Add(3)
	go sensormanager.StartBlockingHttpServer(&wg, &authDatabase, httpServerPort)
	mqttClient := sensormanager.ConnectMqttClient(fmt.Sprintf("ws://%s:%d", mqttHost, mqttPort), "sensor-manager", sensormanager.SuperuserUsername, authDatabase.AdministratorAccessToken)
	go sensormanager.StartMessageTransformations(&wg, &authDatabase, mqttClient)
	go sensormanager.StartContainerManager(&wg, cimiTraefikHost, cimiTraefikPort, lifecycleHost, lifecyclePort, mqttHost, mqttPort, &authDatabase, sensorCheckIntervalSeconds, sensorContainerMapFilename, sensorDriverDockerNetworkName, mqttPathSuffix)
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
		sensorContainerMapFilename := getEnvMandatoryString("SENSOR_CONTAINER_MAP_FILE")
		sensorDriverDockerNetworkName := getEnvMandatoryString("SENSOR_DRIVER_DOCKER_NETWORK_NAME")
		mqttPathSuffix := getEnvMandatoryString("MQTT_PATH_SUFFIX")

		rand.Seed(int64(crc64.Checksum([]byte(applicationSecret), crc64.MakeTable(crc64.ECMA))))
		authDatabase := sensormanager.LoadOrCreateAuthDatabase(authDatabaseFilename, administratorAccessToken, sensorDriverAccessToken)

		runProduction(
			mqttHost, uint16(mqttPort),
			cimiHost, uint16(cimiPort),
			lifecycleHost, uint16(lifecyclePort),
			authDatabase,
			uint16(httpServerPort),
			uint(sensorsCheckIntervalSeconds),
			sensorContainerMapFilename,
			sensorDriverDockerNetworkName,
			mqttPathSuffix,
		)
	}
}
