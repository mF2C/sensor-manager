package main

import (
	"encoding/json"
	"github.com/eclipse/paho.mqtt.golang"
	"time"
)

func publishMessagesIndefinitely(client mqtt.Client, topic string, interval time.Duration) {
	for i := 0; true; i++ {
		message := OutgoingClientMessage{
			Value: float64(i),
			Unit: "iteration",
		}
		marshaled, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}
		token := client.Publish(topic, 0, false, marshaled)
		token.Wait()
		time.Sleep(interval)
	}
}
