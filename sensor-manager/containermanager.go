package sensormanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

const CimiUsername = "sensor-manager-user"
const CimiPassword = "sensor-manager-password"

type SensorDriverContainer struct {
	SensorHardwareModel string
	DockerImagePath     string
	DockerImageVersion  string
	DockerNetworkName   string
	// contains SENSOR_MANAGER_ and SENSOR_ vars
	Environment []struct {
		Key   string
		Value string
	}
}

func (receiver SensorDriverContainer) getCimiServiceName() string {
	return fmt.Sprintf("sensor-driver-%s", receiver.SensorHardwareModel)
}

// reads a mapping file (json) for mappings; the whole file is reread each time to allow on-the-fly updates
func getDriverContainerForSensor(sensorContainerMapFilename string, sensor CimiSensor, authDb *AuthDatabase, mqttHost string, mqttPort uint16, sensorDriverDockerNetworkName string, mqttPathSuffix string) (*SensorDriverContainer, error) {
	hwContainerMap := map[string]struct {
		Image   string `json:"image"`
		Version string `json:"version"`
	}{}

	contents, err := ioutil.ReadFile(sensorContainerMapFilename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(contents, &hwContainerMap)
	if err != nil {
		return nil, err
	}

	mapping, ok := hwContainerMap[sensor.HardwareModel]
	if !ok {
		return nil, fmt.Errorf("no sensor driver container mapping for sensor %s", sensor.HardwareModel)
	}

	connectionParamsJson, err := json.Marshal(sensor.ConnectionParameters)
	if err != nil {
		return nil, err
	}
	env := []struct {
		Key   string
		Value string
	}{
		{"SENSOR_MANAGER_HOST", mqttHost},
		{"SENSOR_MANAGER_PORT", fmt.Sprintf("%d", mqttPort)},
		{"SENSOR_MANAGER_PATH_SUFFIX", mqttPathSuffix},
		{"SENSOR_MANAGER_USERNAME", SensorDriverUsername},
		{"SENSOR_MANAGER_PASSWORD", authDb.SensorDriverAccessToken},
		{"SENSOR_MANAGER_TOPIC", TopicSensorReceive},
		{"SENSOR_CONNECTION_INFO", string(connectionParamsJson)},
	}

	return &SensorDriverContainer{
		SensorHardwareModel: sensor.HardwareModel,
		DockerImagePath:     mapping.Image,
		DockerImageVersion:  mapping.Version,
		DockerNetworkName:   sensorDriverDockerNetworkName,
		Environment:         env,
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
		user, err = getCimiUser(connectionParams, CimiUsername)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, fmt.Errorf("created a CIMI user for %s but it was not present on lookup", CimiUsername)
		}
	}
	return user, nil
}

func getOrCreateCimiSlaTemplate(connectionParams Mf2cConnectionParameters, templateName string) (*CimiSlaTemplate, error) {
	slaTemplate, err := getSlaTemplate(connectionParams, templateName)
	if err != nil {
		return nil, err
	}
	if slaTemplate == nil {
		log.Printf("SLA template %s does not exist, creating.", templateName)
		err = createSlaTemplate(connectionParams, templateName)
		if err != nil {
			return nil, err
		}
		slaTemplate, err = getSlaTemplate(connectionParams, templateName)
		if err != nil {
			return nil, err
		}
		if slaTemplate == nil {
			return nil, fmt.Errorf("created a SLA template for %s but it was not present on lookup", templateName)
		}
	}
	return slaTemplate, err
}

func getOrCreateService(connectionParams Mf2cConnectionParameters, sensorDriverContainer SensorDriverContainer, slaTemplate CimiSlaTemplate) (*CimiService, error) {
	cimiService, err := getSensorDriverService(connectionParams, sensorDriverContainer)
	if err != nil {
		return nil, err
	}
	if cimiService == nil {
		log.Printf("Service for %s does not exist, creating.", sensorDriverContainer.SensorHardwareModel)
		err = createSensorDriverService(connectionParams, sensorDriverContainer, slaTemplate)
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

func getOrCreateServiceInstance(cimiConnectionParams Mf2cConnectionParameters, lifecycleConnectionParams Mf2cConnectionParameters, cimiUser CimiUser, cimiService CimiService) (*CimiServiceInstance, error) {
	cimiServiceInstance, err := getSensorDriverServiceInstance(cimiConnectionParams, cimiService)
	if err != nil {
		return nil, err
	}
	if cimiServiceInstance == nil {
		log.Printf("Service instance for %s does not exist, creating.", cimiService.Name)
		err = startSensorDriverService(lifecycleConnectionParams, cimiUser, cimiService)
		if err != nil {
			return nil, err
		}
		cimiServiceInstance, err = getSensorDriverServiceInstance(cimiConnectionParams, cimiService)
		if err != nil {
			return nil, err
		}
		if cimiServiceInstance == nil {
			return nil, fmt.Errorf("started a CIMI service instance for service %s but it was not present on lookup", cimiService.Name)
		}
	}
	return cimiServiceInstance, nil
}

func StartContainerManager(wg *sync.WaitGroup, cimiTraefikHost string, cimiTraefikPort uint16, lifecycleHost string, lifecyclePort uint16,
	mqttHost string, mqttPort uint16, authDb *AuthDatabase, sensorCheckIntervalSeconds uint, sensorContainerMapFilename string, sensorDriverDockerNetworkName string, mqttPathSuffix string) {
	defer wg.Done()
	log.Println("Starting container manager.")

	cimiConnectionParams := Mf2cConnectionParameters{
		Host:     cimiTraefikHost,
		Port:     cimiTraefikPort,
		Protocol: "https",
		Headers: append([]struct {
			Key   string
			Value string
		}{
			{Key: CimiAuthenticationHeaderKey, Value: CimiAuthenticationBypassValue},
		}, CimiAdditionalHeaders...),
	}

	lifecycleConnectionParams := Mf2cConnectionParameters{
		Host:     lifecycleHost,
		Port:     lifecyclePort,
		Protocol: "http",
		Headers:  LifecycleAdditionalHeaders,
	}

	// go is such a nice and readable language wow, very simple and much good
	var cimiUser *CimiUser
	var err error
	for {
		cimiUser, err = getOrCreateUser(cimiConnectionParams)
		if err != nil {
			log.Printf("Error establishing CIMI user: %s", err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	var slaTemplate *CimiSlaTemplate
	for {
		slaTemplate, err = getOrCreateCimiSlaTemplate(cimiConnectionParams, CimiSlaTemplateName)
		if err != nil {
			log.Printf("Error establishing CIMI user: %s", err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	knownSensors := map[string]CimiSensor{}
	for {
		sensors, err := getSensorsFromCimi(cimiConnectionParams)
		if err != nil {
			log.Printf("Error getting sensors from CIMI: %s", err)
			time.Sleep(time.Duration(sensorCheckIntervalSeconds) * time.Second)
			continue
		}

		// golang is much production language and also does not support breaking out of specific nested loops is bad and you should feel bad
		for _, s := range sensors {
			_, present := knownSensors[s.HardwareModel]
			if !present {
				knownSensors[s.HardwareModel] = s
				log.Printf("Adding a new sensor container: %s", s.HardwareModel)
				sensorDriverContainer, err := getDriverContainerForSensor(sensorContainerMapFilename, s, authDb, mqttHost, mqttPort, sensorDriverDockerNetworkName, mqttPathSuffix)
				if err != nil {
					log.Printf("Error adding a new sensor container: %s", err)
					break
				}
				log.Printf("Ensuring the service for %s exists.", s.HardwareModel)
				cimiService, err := getOrCreateService(cimiConnectionParams, *sensorDriverContainer, *slaTemplate)
				if err != nil {
					log.Printf("Error creating the sensor driver service: %s", err)
					break
				}
				log.Printf("Ensuring the service instance for %s exists.", s.HardwareModel)
				_, err = getOrCreateServiceInstance(cimiConnectionParams, lifecycleConnectionParams, *cimiUser, *cimiService)
				if err != nil {
					log.Printf("Error spawning the sensor driver service: %s", err)
					break
				}
			}
		}

		time.Sleep(time.Duration(sensorCheckIntervalSeconds) * time.Second)
	}
}
