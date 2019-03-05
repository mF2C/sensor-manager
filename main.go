package main

import (
	"github.com/eclipse/paho.mqtt.golang"
	flag "github.com/spf13/pflag"
	"log"
	"time"
)

func main() {
	testpublish := flag.Bool("testpublish", false, "Publish messages to MQTT indefinitely.")
	flag.Parse()

	authDatabase := loadOrCreateAuthDatabase("authdb.json")

	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("TOPIC: %s\n", msg.Topic())
		log.Printf("MSG: %s\n", msg.Payload())
	}

	address := "tcp://172.21.0.68:1883"
	log.Printf("Connecting to MQTT server at %s", address)
	mqttClient := connectMqttClient(address, "sensor-manager", messageHandler, authDatabase.SystemServiceToken)

	if *testpublish {
		log.Println("Testpublish mode activated.")
		publishMessagesIndefinitely(mqttClient, "testpublish", 1*time.Second)
	}

	log.Println("Starting message transformations")
	startMessageTransformations(mqttClient)
	log.Println("Starting HTTP server on port 8080.")
	startBlockingHttpServer(authDatabase, 8080)
}
