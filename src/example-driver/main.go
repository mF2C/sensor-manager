package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"strconv"
	"time"
)

// here for clarity; authority in mqtt.go
type SensorReading struct {
	SensorId   string
	SensorType string
	Quantity   string
	Timestamp  string
	Value      float64
	Unit       string
}

// here for clarity, authority in mqtt.go
func connectMqttClient(address string, clientId string, username string, password string) mqtt.Client {
	log.Printf("Building a new MQTT client with id %s.", clientId)
	mqttClientOptions := mqtt.NewClientOptions()
	mqttClientOptions.AddBroker(address)
	mqttClientOptions.SetClientID(clientId)
	mqttClientOptions.SetKeepAlive(2 * time.Second)
	mqttClientOptions.SetPingTimeout(1 * time.Second)
	mqttClientOptions.SetUsername(username)
	mqttClientOptions.SetPassword(password)

	mqttClient := mqtt.NewClient(mqttClientOptions)
	log.Printf("Connecting to MQTT server at %s", address)
	connectionSuccessful := false
	for i := 1; !connectionSuccessful; i++ {
		log.Printf("    connection attempt %d...", i)
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("        unsuccessful: %s", token.Error())
			time.Sleep(1 * time.Second)
		} else {
			connectionSuccessful = true
		}
	}
	log.Println("Connection to MQTT server successful.")
	return mqttClient
}

func main() {
	sensorManagerHost := os.Getenv("SENSOR_MANAGER_HOST")
	sensorManagerPort, err := strconv.Atoi(os.Getenv("SENSOR_MANAGER_PORT"))
	if err != nil {
		panic(err)
	}
	sensorManagerUsername := os.Getenv("SENSOR_MANAGER_USERNAME")
	sensorManagerPassword := os.Getenv("SENSOR_MANAGER_PASSWORD")
	sensorManagerTopic := os.Getenv("SENSOR_MANAGER_TOPIC")

	sensorManagerConnectionInfoString := os.Getenv("SENSOR_CONNECTION_INFO")
	var sensorManagerConnectionInfo map[string]interface{}
	err = json.Unmarshal([]byte(sensorManagerConnectionInfoString), &sensorManagerConnectionInfo)
	if err != nil {
		panic(err)
	}

	log.Printf("Connecting to %s:%d", sensorManagerHost, sensorManagerPort)
	log.Printf("Using username %s for topic %s", sensorManagerUsername, sensorManagerTopic)
	log.Printf("Connection parameters: %+v", sensorManagerConnectionInfo)

	mqttClient := connectMqttClient(fmt.Sprintf("%s:%d", sensorManagerHost, sensorManagerPort), "example-driver", sensorManagerUsername, sensorManagerPassword)

	for i := 1; true; i++ {
		reading := SensorReading{
			SensorId:   "example-driver",
			SensorType: "example-driver",
			Quantity:   "example-count",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Value:      float64(i),
			Unit:       "times",
		}
		readingJson, err := json.Marshal(reading)
		if err != nil {
			panic(err)
		}

		mqttClient.Publish(sensorManagerTopic, 0, false, readingJson)
		time.Sleep(1 * time.Second)
	}
}
