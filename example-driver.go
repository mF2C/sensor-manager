package main

import (
	"encoding/json"
	"fmt"
	"log"
	sensormanager "mf2c-sensor-manager/sensor-manager"
	"os"
	"strconv"
	"time"
)

func main() {
	sensorManagerHost, present := os.LookupEnv("SENSOR_MANAGER_HOST")
	if !present {
		panic(fmt.Errorf("sensor manager host not specified"))
	}

	sensorManagerPort, err := strconv.Atoi(os.Getenv("SENSOR_MANAGER_PORT"))
	if err != nil {
		panic(err)
	}

	sensorManagerPathSuffix, present := os.LookupEnv("SENSOR_MANAGER_PATH_SUFFIX")
	if !present {
		panic(fmt.Errorf("sensor manager path suffix not specified (empty is a valid value)"))
	}

	sensorManagerUsername, present := os.LookupEnv("SENSOR_MANAGER_USERNAME")
	if !present {
		panic(fmt.Errorf("sensor manager username not specified"))
	}

	sensorManagerPassword, present := os.LookupEnv("SENSOR_MANAGER_PASSWORD")
	if !present {
		panic(fmt.Errorf("sensor manager password not specified"))
	}

	sensorManagerTopic, present := os.LookupEnv("SENSOR_MANAGER_TOPIC")
	if !present {
		panic(fmt.Errorf("sensor manager topic not specified"))
	}

	sensorManagerConnectionInfoString := os.Getenv("SENSOR_CONNECTION_INFO")
	var sensorManagerConnectionInfo map[string]interface{}
	err = json.Unmarshal([]byte(sensorManagerConnectionInfoString), &sensorManagerConnectionInfo)
	if err != nil {
		panic(err)
	}

	log.Printf("Connecting to %s:%d", sensorManagerHost, sensorManagerPort)
	log.Printf("Using username %s for topic %s", sensorManagerUsername, sensorManagerTopic)
	log.Printf("Connection parameters: %+v", sensorManagerConnectionInfo)

	mqttClient := sensormanager.ConnectMqttClient(fmt.Sprintf("ws://%s:%d%s", sensorManagerHost, sensorManagerPort, sensorManagerPathSuffix), "example-driver", sensorManagerUsername, sensorManagerPassword)
	log.Print("WARNING: a successful connection does not mean writes will succeed - the MQTT server silently drops unauthorised writes!")

	for i := 1; true; i++ {
		reading := sensormanager.IncomingSensorMessage{
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

		sendtoken := mqttClient.Publish(sensorManagerTopic, 0, false, readingJson)
		if sendtoken.Wait() && sendtoken.Error() != nil {
			log.Print(sendtoken.Error())
		} else {
			log.Print("Value published.")
		}
		time.Sleep(1 * time.Second)
	}
}
