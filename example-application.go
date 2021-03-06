package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io/ioutil"
	"log"
	sensormanager "mf2c-sensor-manager/sensor-manager"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	sensorManagerApiHost, present := os.LookupEnv("SENSOR_MANAGER_API_HOST")
	if !present {
		panic(fmt.Errorf("sensor manager API host not specified"))
	}

	sensorManagerApiPort, err := strconv.Atoi(os.Getenv("SENSOR_MANAGER_API_PORT"))
	if err != nil {
		panic(err)
	}

	sensorManagerApiPathPrefix, present := os.LookupEnv("SENSOR_MANAGER_API_PATH_PREFIX")
	if !present {
		panic(fmt.Errorf("sensor manager API path prefix not specified (empty is a valid value)"))
	}

	sensorManagerMqttHost, present := os.LookupEnv("SENSOR_MANAGER_MQTT_HOST")
	if !present {
		panic(fmt.Errorf("sensor manager MQTT host not specified"))
	}

	sensorManagerMqttPort, err := strconv.Atoi(os.Getenv("SENSOR_MANAGER_MQTT_PORT"))
	if err != nil {
		panic(err)
	}

	sensorManagerMqttPathSuffix, present := os.LookupEnv("SENSOR_MANAGER_MQTT_PATH_SUFFIX")
	if !present {
		panic(fmt.Errorf("sensor manager MQTT path suffix not specified (empty is a valid value)"))
	}

	sensorManagerUrl := fmt.Sprintf("http://%s:%d%s", sensorManagerApiHost, sensorManagerApiPort, sensorManagerApiPathPrefix)
	topicsResponse, err := http.Get(sensorManagerUrl + "/topics")
	if err != nil {
		panic(err)
	}
	topicsBody, err := ioutil.ReadAll(topicsResponse.Body)
	if err != nil {
		panic(err)
	}

	var topics map[string]sensormanager.SensorTopic
	err = json.Unmarshal(topicsBody, &topics)

	log.Print("Got available topics:")
	log.Print(topics)

	if len(topics) == 0 {
		log.Print("No available topics to listen to, exiting.")
	}
	var firstTopic sensormanager.SensorTopic
	for _, value := range topics {
		firstTopic = value
		break
	}
	log.Printf("Listening for sensor values on the first topic: %s", firstTopic.Name)
	mqttClient := sensormanager.ConnectMqttClient(
		fmt.Sprintf("ws://%s:%d%s", sensorManagerMqttHost, sensorManagerMqttPort, sensorManagerMqttPathSuffix),
		"example-application",
		firstTopic.Username,
		firstTopic.Password,
	)

	if token := mqttClient.Subscribe(firstTopic.Name, 0, func(receiveClient mqtt.Client, message mqtt.Message) {
		log.Printf("Got message on topic %s: %s", message.Topic(), string(message.Payload()))
	}); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	} else {
		log.Print("No error subscribing to the data topic.")
	}
	log.Print("Sleeping for a hundred years to allow background processing of messages.")
	time.Sleep(100 * 365 * 24 * time.Hour)
}
