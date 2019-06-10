package sensormanager

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"sync"
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
	// the timestamp in RFC 3339
	Timestamp string
	Value     float64
	// the SI unit (deg. celsius, RH%, kg, etc), without SI prefixes (except kg, which is a base unit)
	Unit string
}

// only defines values not known prior to the request
// e.g. not the sensor type, because this sensor's data has been requested
type OutgoingClientMessage struct {
	Timestamp string
	Value     float64
	Unit      string
}

func ConnectMqttClient(address string, clientId string, username string, password string) mqtt.Client {
	log.Printf("Building a new MQTT client with id %s.", clientId)
	defaultMessageHandler := func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("client %s got: %s -> %s", clientId, msg.Topic(), msg.Payload())
	}

	mqttClientOptions := mqtt.NewClientOptions()
	mqttClientOptions.AddBroker(address)
	mqttClientOptions.SetClientID(clientId)
	mqttClientOptions.SetKeepAlive(2 * time.Second)
	mqttClientOptions.SetDefaultPublishHandler(defaultMessageHandler)
	mqttClientOptions.SetPingTimeout(1 * time.Second)
	if username != "" {
		mqttClientOptions.SetUsername(username)
	}
	if password != "" {
		mqttClientOptions.SetPassword(password)
	}

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

func validateIncomingMessage(incoming IncomingSensorMessage) bool {
	_, err := time.Parse(time.RFC3339Nano, incoming.Timestamp)
	return err == nil
}

func transformMessage(incoming IncomingSensorMessage) OutgoingClientMessage {
	return OutgoingClientMessage{
		Timestamp: incoming.Timestamp,
		Value:     incoming.Value,
		Unit:      incoming.Unit,
	}
}

func StartMessageTransformations(wg *sync.WaitGroup, authDb *AuthDatabase, subscribeClient mqtt.Client) {
	defer wg.Done()
	log.Println("Starting message transformations.")
	if token := subscribeClient.Subscribe(TopicSensorReceive, 0, func(receiveClient mqtt.Client, message mqtt.Message) {
		log.Printf("Got sensor driver message.")
		unmarshaled := IncomingSensorMessage{}
		err := json.Unmarshal(message.Payload(), &unmarshaled)
		if err != nil {
			log.Println(err)
		} else {
			if !validateIncomingMessage(unmarshaled) {
				log.Printf("Invalid timestamp format for incoming message, skipping: %s", unmarshaled.Timestamp)
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
					log.Printf("Message transformation successful, publishing on the outgoing topic: %s", outTopicName)
					receiveClient.Publish(outTopicName, 0, false, transformedRemarshaled)
				}
			}
		}
	}); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	} else {
		log.Print("No error subscribing to the sensor receive topic.")
	}
}
