package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

const SuperuserUsername = "system"
const SensorDriverUsername = "sensor-driver"

// should be a multiple of 8
const GeneratedTokenLengthBytes = 32

type SensorTopic struct {
	SensorId string
	// the complete topic, not only the last part
	Name     string
	Quantity string
	Username string
	Password string
}

type AuthDatabase struct {
	// big ugly hack
	Filename string
	// maps sensor IDs to topics
	Topics map[string]SensorTopic
	// authenticates system services, also a big ugly hack
	AdministratorAccessToken string
	// authenticates sensor drivers, also a big ugly hack
	SensorDriverAccessToken string
}

func loadOrCreateAuthDatabase(filename string, administratorAccessToken string, sensorDriverAccessToken string) AuthDatabase {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Reading auth database file %s failed, creating anew.", filename)
		newAuthDb := AuthDatabase{
			Filename:                 filename,
			Topics:                   map[string]SensorTopic{},
			AdministratorAccessToken: administratorAccessToken,
			SensorDriverAccessToken:  sensorDriverAccessToken,
		}
		err = os.MkdirAll(path.Dir(filename), 0776)
		if err != nil {
			log.Println(fmt.Errorf("could not create auth database parent directories, panic"))
			panic(err)
		}
		newAuthDb.writeToFile()
		return newAuthDb
	}

	log.Printf("Auth database file %s read successfully.", filename)
	unmarshaled := AuthDatabase{}
	if json.Unmarshal(contents, &unmarshaled) != nil {
		log.Println(fmt.Errorf("failed to unmarshal database, panic"))
		panic(err)
	}
	// always overwrite
	unmarshaled.AdministratorAccessToken = administratorAccessToken
	unmarshaled.SensorDriverAccessToken = sensorDriverAccessToken
	unmarshaled.writeToFile()
	return unmarshaled
}

func buildTopicFromSensorId(unsafe string) string {
	sanitised := strings.Map(func(c rune) rune {
		if (47 <= c && c <= 57) || (65 <= c && c <= 90) || (97 <= c && c <= 122) {
			return c
		} else {
			return '_'
		}
	}, unsafe)
	return TopicClientPublishRoot + sanitised
}

func generateRandomString() string {
	base := make([]byte, GeneratedTokenLengthBytes)
	_, err := rand.Read(base)
	if err != nil {
		// the docs say this always returns nil
		// why even have the error ffs
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(base)
}

func constantTimeStringEqual(left string, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func generateUsernamePassword() (username string, password string) {
	return generateRandomString(), generateRandomString()
}

func (db AuthDatabase) writeToFile() {
	serialized, err := json.Marshal(db)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(db.Filename, serialized, 0660)
	if err != nil {
		log.Println(fmt.Errorf("error writing file detabase file"))
		log.Println(err)
	}
}

func (db AuthDatabase) addSensorTopic(sensorId string, quantity string) (topicName string, err error) {
	if _, ok := db.Topics[sensorId]; !ok {
		return "", fmt.Errorf("sensor ID already exists: %s", sensorId)
	}
	username, password := generateUsernamePassword()
	newTopic := SensorTopic{
		SensorId: sensorId,
		Name:     buildTopicFromSensorId(sensorId),
		Quantity: quantity,
		Username: username,
		Password: password,
	}
	db.Topics[sensorId] = newTopic
	db.writeToFile()
	log.Printf("Added topic %s for sensor %s", newTopic.Name, newTopic.SensorId)
	return newTopic.Name, nil
}

func (db AuthDatabase) getTopicForSensor(sensorId string) (topicName string, err error) {
	topic, ok := db.Topics[sensorId]
	if !ok {
		return "", fmt.Errorf("no topic for sensor %s", sensorId)
	} else {
		return topic.Name, nil
	}
}

// if any credential matches, the user is authenticated
func (db AuthDatabase) isAuthenticated(username string, password string) bool {
	if db.isSuperuser(username, password) || db.isSensorDriver(username, password) {
		return true
	}
	for _, dbTopic := range db.Topics {
		if constantTimeStringEqual(username, dbTopic.Username) && constantTimeStringEqual(password, dbTopic.Password) {
			return true
		}
	}
	return false
}

// if the (username, topic) tuple exists
// authentication with the password is done in isAuthenticated
// the password is not available here, as this is only called when authentication passes
func (db AuthDatabase) isAuthorized(username string, topic string) bool {
	if constantTimeStringEqual(username, SuperuserUsername) || (constantTimeStringEqual(username, SensorDriverUsername) && constantTimeStringEqual(topic, TopicSensorReceive)) {
		return true
	}
	for _, dbTopic := range db.Topics {
		if constantTimeStringEqual(topic, dbTopic.Name) && constantTimeStringEqual(username, dbTopic.Username) {
			return true
		}
	}
	return false
}

func (db AuthDatabase) isSuperuser(username string, password string) bool {
	return constantTimeStringEqual(username, SuperuserUsername) && constantTimeStringEqual(password, db.AdministratorAccessToken)
}

func (db AuthDatabase) isSensorDriver(username string, password string) bool {
	return constantTimeStringEqual(username, SensorDriverUsername) && constantTimeStringEqual(password, db.SensorDriverAccessToken)
}
