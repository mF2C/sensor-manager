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

	sensorManagerMqttHost, present := os.LookupEnv("SENSOR_MANAGER_MQTT_HOST")
	if !present {
		panic(fmt.Errorf("sensor manager MQTT host not specified"))
	}

	sensorManagerMqttPort, err := strconv.Atoi(os.Getenv("SENSOR_MANAGER_MQTT_PORT"))
	if err != nil {
		panic(err)
	}

	sensorManagerUrl := fmt.Sprintf("http://%s:%d", sensorManagerApiHost, sensorManagerApiPort)
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
		fmt.Sprintf("ws://%s:%d", sensorManagerMqttHost, sensorManagerMqttPort),
		"example-application",
		firstTopic.Username,
		firstTopic.Password,
	)

	if token := mqttClient.Subscribe(firstTopic.Name, 0, func(receiveClient mqtt.Client, message mqtt.Message) {
		log.Printf("Got message: %s", string(message.Payload()))
	}); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	} else {
		log.Print("No error subscribing to the data topic.")
	}
	log.Print("Sleeping for a hundred years to allow background processing of messages.")
	time.Sleep(100 * 365 * 24 * time.Hour)
}
