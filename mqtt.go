package main

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"time"
)

// sensors are on a level below this, with their UUID
const TopicSensorReceive = "/sensor-manager/sensor-incoming"
const TopicClientPublishRoot = "/sensor-manager/values/"

type IncomingSensorMessage struct {
	// an ID that differentiates this piece of hardware sensor from others
	SensorId string
	// the type of the sensor: AM2302 etc
	SensorType string
	// the SI dimension (temperature, humidity, weight, etc)
	Quantity string
	Value    float64
	// the SI unit (deg. celsius, RH%, kg, etc), without SI prefixes (except kg, which is a base unit)
	Unit string
}

// only defines values not known prior to the request
// e.g. not the sensor type, because this sensor's data has been requested
type OutgoingClientMessage struct {
	Value float64
	Unit  string
}

func connectMqttClient(address string, id string, systemServiceToken string) mqtt.Client {
	log.Println("Building a new MQTT client.")
	defaultMessageHandler := func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("%s -> %s", msg.Topic(), msg.Payload())
	}

	mqttClientOptions := mqtt.NewClientOptions().AddBroker(address).SetClientID(id)
	mqttClientOptions.SetKeepAlive(2 * time.Second)
	mqttClientOptions.SetDefaultPublishHandler(defaultMessageHandler)
	mqttClientOptions.SetPingTimeout(1 * time.Second)
	mqttClientOptions.SetUsername(SystemTokenUsername)
	mqttClientOptions.SetPassword(systemServiceToken)

	mqttClient := mqtt.NewClient(mqttClientOptions)
	log.Printf("Connecting to MQTT server at %s", address)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return mqttClient
}

func transformMessage(incoming IncomingSensorMessage) OutgoingClientMessage {
	return OutgoingClientMessage{
		Value: incoming.Value,
		Unit:  incoming.Unit,
	}
}

func startMessageTransformations(subscribeClient mqtt.Client, authDb AuthDatabase) {
	log.Println("Starting message transformations.")
	if token := subscribeClient.Subscribe(TopicSensorReceive, 0, func(receiveClient mqtt.Client, message mqtt.Message) {
		unmarshaled := IncomingSensorMessage{}
		err := json.Unmarshal(message.Payload(), &unmarshaled)
		if err != nil {
			log.Println(err)
		} else {
			transformedRemarshaled, err := json.Marshal(transformMessage(unmarshaled))
			if err != nil {
				log.Println(err)
			} else {
				outTopicName, err := authDb.getTopicForSensor(unmarshaled.SensorId)
				if err != nil {
					newTopicName, err := authDb.addSensorTopic(unmarshaled.SensorId, unmarshaled.Quantity)
					if err != nil {
						panic(err)
					}
					// go is such agile and also very good language, wow
					outTopicName = newTopicName
				}
				log.Printf("Message transformation successful publishing on the outgoing topic: %s", outTopicName)
				receiveClient.Publish(outTopicName, 0, false, transformedRemarshaled)
			}
		}
	}); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	}
}
