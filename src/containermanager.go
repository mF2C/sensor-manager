package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const CimiUsername = "sensor-manager-user"
const CimiPassword = "sensor-manager-password"

// # TODO envvar this interval
const SensorCheckIntervalSeconds = 5

type SensorDriverContainer struct {
	SensorHardwareModel string
	DockerImagePath     string
	DockerImageVersion  string
}

func (receiver SensorDriverContainer) getCimiServiceName() string {
	return fmt.Sprintf("sensor-driver-%s", receiver.SensorHardwareModel)
}

func (receiver SensorDriverContainer) getCimiServiceIdentifier() CimiIdentifier {
	return CimiIdentifier(fmt.Sprintf("service/%s", receiver.getCimiServiceName()))
}

func getDriverContainerForSensor(sensor CimiSensor) (*SensorDriverContainer, error) {
	// TODO: this is hardcoded, use a database
	hwContainerMap := map[string]struct {
		image   string
		version string
	}{
		"DHT22": {"hello-world", "latest"},
	}

	mapping, ok := hwContainerMap[sensor.HardwareModel]
	if !ok {
		return nil, fmt.Errorf("no sensor driver container mapping for sensor %s", sensor.HardwareModel)
	}
	return &SensorDriverContainer{
		SensorHardwareModel: sensor.HardwareModel,
		DockerImagePath:     mapping.image,
		DockerImageVersion:  mapping.version,
	}, nil
}

func getOrCreateUser(connectionParams Mf2cConnectionParameters) (*CimiUser, error) {
	log.Println("Initialising user.")
	user, err := getCimiUser(connectionParams, CimiUsername)
	if err != nil {
		return nil, err
	}
	if user == nil {
		err = createCimiUser(connectionParams, CimiUsername, CimiPassword)
		if err != nil {
			return nil, err
		}
		user, err := getCimiUser(connectionParams, CimiUsername)
		if err != nil {
			return nil, err
		}
		if user == nil {
			log.Printf("Started a CIMI user for %s but it was not present on lookup.", CimiUsername)
			os.Exit(1)
		}
	}
	return user, nil
}

func getOrCreateService(connectionParams Mf2cConnectionParameters, sensorDriverContainer SensorDriverContainer) (*CimiService, error) {
	cimiService, err := getSensorDriverService(connectionParams, sensorDriverContainer)
	if err != nil {
		return nil, err
	}
	if cimiService == nil {
		log.Printf("Service for %s does not exist, creating.", sensorDriverContainer.SensorHardwareModel)
		err = createSensorDriverService(connectionParams, sensorDriverContainer)
		if err != nil {
			return nil, err
		}
		cimiService, err = getSensorDriverService(connectionParams, sensorDriverContainer)
		if err != nil {
			return nil, err
		}
		if cimiService == nil {
			return nil, fmt.Errorf("created a CIMI service for %s but it was not present on lookup", sensorDriverContainer.SensorHardwareModel)
		}
	}
	return cimiService, nil
}

func getOrCreateServiceInstance(connectionParams Mf2cConnectionParameters, sensorDriverContainer SensorDriverContainer, cimiUser CimiUser, cimiService CimiService) (*CimiServiceInstance, error) {
	cimiServiceInstance, err := getSensorDriverServiceInstance(connectionParams, sensorDriverContainer)
	if err != nil {
		return nil, err
	}
	if cimiServiceInstance == nil {
		log.Printf("Service instance for %s does not exist, creating.", sensorDriverContainer.SensorHardwareModel)
		err = startSensorDriverService(connectionParams, cimiUser, cimiService)
		if err != nil {
			return nil, err
		}
		cimiServiceInstance, err := getSensorDriverServiceInstance(connectionParams, sensorDriverContainer)
		if err != nil {
			return nil, err
		}
		if cimiServiceInstance == nil {
			return nil, fmt.Errorf("started a CIMI service instance for %s but it was not present on lookup", sensorDriverContainer.SensorHardwareModel)
		}
	}
	return cimiServiceInstance, nil
}

func startContainerManager(wg *sync.WaitGroup, cimiTraefikHost string, cimiTraefikPort uint16, lifecycleHost string, lifecyclePort uint16, authDb *AuthDatabase) {
	defer wg.Done()
	log.Println("Starting container manager.")

	// TODO: envvar these
	cimiConnectionParams := Mf2cConnectionParameters{
		Host:     cimiTraefikHost,
		Port:     cimiTraefikPort,
		Protocol: "https",
		Headers: append([]struct {
			Key   string
			Value string
		}{{Key: CimiAuthenticationHeaderKey, Value: CimiAuthenticationBypassValue}}, CimiAdditionalHeaders...),
	}

	lifecycleConnectionParams := Mf2cConnectionParameters{
		Host:     lifecycleHost,
		Port:     lifecyclePort,
		Protocol: "http",
		Headers: LifecycleAdditionalHeaders,
	}

	cimiUser, err := getOrCreateUser(cimiConnectionParams)
	if err != nil {
		panic(err)
	}

	knownSensors := map[string]CimiSensor{}
	for {
		sensors, err := getSensorsFromCimi(cimiConnectionParams)
		if err != nil {
			panic(err)
		}
		for _, s := range sensors {
			_, present := knownSensors[s.HardwareModel]
			if !present {
				log.Printf("Adding a new sensor container: %s", s.HardwareModel)
				sensorDriverContainer, err := getDriverContainerForSensor(s)
				if err != nil {
					panic(err)
				}
				log.Printf("Ensuring the service for %s exists.", s.HardwareModel)
				cimiService, err := getOrCreateService(cimiConnectionParams, *sensorDriverContainer)
				if err != nil {
					panic(err)
				}
				log.Printf("Ensuring the service instance for %s exists.", s.HardwareModel)
				_, err = getOrCreateServiceInstance(lifecycleConnectionParams, *sensorDriverContainer, *cimiUser, *cimiService)
				if err != nil {
					panic(err)
				}
			}
		}

		time.Sleep(SensorCheckIntervalSeconds * time.Second)
	}
}
