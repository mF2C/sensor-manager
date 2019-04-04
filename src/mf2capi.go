package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// TODO: this will be removed. maybe.
const CimiAuthenticationHeaderKey = "Slipstream-Authn-Info"
const CimiAuthenticationBypassValue = "internal ADMIN"

var CimiAdditionalHeaders = []struct {
	Key   string
	Value string
}{
	{"Content-Type", "application/json"},
}
var LifecycleAdditionalHeaders = CimiAdditionalHeaders

type Mf2cConnectionParameters struct {
	Host     string
	Port     uint16
	Protocol string
	Headers  []struct {
		Key   string
		Value string
	}
}

func (receiver Mf2cConnectionParameters) buildUrl(endpoint string) string {
	return fmt.Sprintf("%s://%s:%d/%s", receiver.Protocol, receiver.Host, receiver.Port, strings.TrimLeft(endpoint, "/"))
}

func (receiver Mf2cConnectionParameters) execute(method string, endpoint string, body io.Reader) (*http.Response, error) {
	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}
	transport := http.Transport{
		TLSClientConfig: &tlsConfig,
	}
	client := &http.Client{
		Transport: &transport,
	}
	req, err := http.NewRequest(method, receiver.buildUrl(endpoint), body)
	if err != nil {
		return nil, err
	}
	for _, header := range receiver.Headers {
		req.Header.Add(header.Key, header.Value)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode/100 != 2 {
		byteBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("non-2xx status code at %s /%s: %s\n%s", method, strings.TrimLeft(endpoint, "/"), response.Status, string(byteBody))
	}
	return response, nil
}

func (receiver Mf2cConnectionParameters) get(endpoint string) (string, error) {
	response, err := receiver.execute("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	if response.StatusCode/100 != 2 {
		byteBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("non-2xx status code at GET /%s: %s\n%s", strings.TrimLeft(endpoint, "/"), response.Status, string(byteBody))
	}
	byteBody, err := ioutil.ReadAll(response.Body)
	return string(byteBody), err
}

func (receiver Mf2cConnectionParameters) getUnmarshal(endpoint string, target interface{}) error {
	response, err := receiver.get(endpoint)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(response), target)
	if err != nil {
		return err
	}
	return nil
}

func (receiver Mf2cConnectionParameters) post(endpoint string, data interface{}) error {
	marshaled, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = receiver.execute("POST", endpoint, bytes.NewReader(marshaled))
	if err != nil {
		return err
	}
	return nil
}

type CimiUser struct {
	Id           CimiIdentifier `json:"id"`
	ActiveSince  string         `json:"activeSince"`
	LastExecute  string         `json:"last_Execute"`
	Deleted      bool           `json:"deleted"`
	Password     string         `json:"password"`
	Method       string         `json:"method"`
	Updated      string         `json:"updated"`
	EmailAddress string         `json:"emailAddress"`
	Username     string         `json:"username"`
	Created      string         `json:"created"`
	State        string         `json:"state"`
	LastOnline   string         `json:"lastOnline"`
	IsSuperuser  bool           `json:"isSuperuser"`
}

type CimiUserList struct {
	Count uint       `json:"count"`
	Users []CimiUser `json:"users"`
}

type CimiUserCreationRequest struct {
	UserTemplate struct {
		Href           string `json:"href"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		PasswordRepeat string `json:"passwordRepeat"`
		EmailAddress   string `json:"emailAddress"`
	} `json:"userTemplate"`
}

type CimiSensor struct {
	HardwareModel        string
	Dimensions           []string
	ConnectionParameters map[string]interface{}
}

// used in PUT /api/service
type CimiService struct {
	Id CimiIdentifier `json:"id"`
	// required
	Name      string `json:"name"`
	Exec      string `json:"exec"`
	ExecType  string `json:"exec_type"`
	AgentType string `json:"agent_type"`
	// optional
	Description       string   `json:"description,omitempty"`
	ExecPorts         []uint16 `json:"exec_ports,omitempty"`
	NumAgents         uint     `json:"num_agents,omitempty"`
	CpuArch           string   `json:"cpu_arch,omitempty"`
	Os                string   `json:"os,omitempty"`
	MemoryMin         uint64   `json:"memory_min,omitempty"`
	StorageMin        uint64   `json:"storage_min,omitempty"`
	Disk              uint64   `json:"disk,omitempty"`
	RequiredResources []string `json:"req_resource,omitempty"`
	OptionalResources []string `json:"opt_resource,omitempty"`
	// others
	Created  string `json:"created,omitempty"`
	Updated  string `json:"updated,omitempty"`
	Category int    `json:"category,omitempty"`
}

// parsed from GET /api/service
// some fields not parsed
type CimiServiceList struct {
	Count    uint          `json:"count"`
	Services []CimiService `json:"services"`
}

type CimiHref struct {
	Href string `json:"href"`
}

// descriptive rename for type/uuid strings
type CimiIdentifier string

func (receiver CimiIdentifier) getType() string {
	split := strings.Split(string(receiver), "/")
	if len(split) != 2 {
		return string(receiver)
	} else {
		return split[0]
	}
}

func (receiver CimiIdentifier) getUuid() string {
	split := strings.Split(string(receiver), "/")
	if len(split) != 2 {
		return string(receiver)
	} else {
		return split[1]
	}
}

type CimiAgent struct {
	Agent        CimiHref `json:"agent"`
	Ports        []uint16 `json:"ports"`
	ContainerId  string   `json:"container_id"`
	Status       string   `json:"status"`
	NumCpus      uint16   `json:"num_cpus"`
	Allow        bool     `json:"allow"`
	AgentParam   string   `json:"agent_param"`
	Url          string   `json:"url"`
	MasterCompss bool     `json:"master_compss"`
}

type CimiServiceInstance struct {
	Id CimiIdentifier `json:"id"`
	// inputs, apparently, but not used yet (or ever)
	User      CimiIdentifier `json:"user"`
	Service   CimiIdentifier `json:"service"`
	Agreement CimiIdentifier `json:"agreement"`
	Status    string         `json:"status"`
	Agents    []CimiAgent    `json:"agents"`
	// other data
	ParentDeviceId CimiIdentifier `json:"parent_device_id,omitempty"`
	ParentDeviceIp string         `json:"parent_device_ip,omitempty"`
	Updated        string         `json:"updated,omitempty"`
	Created        string         `json:"created,omitempty"`
	DeviceId       string         `json:"device_id,omitempty"`
	DeviceIp       string         `json:"device_ip,omitempty"`
	ServiceType    string         `json:"service_type,omitempty"`
}

type CimiServiceInstanceList struct {
	Count            uint                  `json:"count"`
	ServiceInstances []CimiServiceInstance `json:"serviceInstances"`
}

type LifecycleServiceStartRequest struct {
	ServiceId CimiIdentifier `json:"service_id"`
	UserId    CimiIdentifier `json:"user_id"`
	// this seems to be optional for now
	AgreementId CimiIdentifier `json:"agreement_id"`
}

func getSensorsFromCimi(connectionParams Mf2cConnectionParameters) ([]CimiSensor, error) {
	// TODO: mocks until cimi integration
	const cimiSensorModel = `["hello-world-sensor"]`
	const cimiSensorTypes = `[["temperature", "humidity"]]`
	const cimiSensorConnection = `[{"gpioPin": 23, "simulated": true}]`

	var parsedSensorModels []string
	err := json.Unmarshal([]byte(cimiSensorModel), &parsedSensorModels)
	if err != nil {
		return nil, err
	}
	var parsedSensorTypes [][]string
	err = json.Unmarshal([]byte(cimiSensorTypes), &parsedSensorTypes)
	if err != nil {
		return nil, err
	}
	var parsedSensorConnections []map[string]interface{}
	err = json.Unmarshal([]byte(cimiSensorConnection), &parsedSensorConnections)
	if err != nil {
		return nil, err
	}

	if len(parsedSensorModels) != len(parsedSensorTypes) || len(parsedSensorTypes) != len(parsedSensorConnections) {
		return nil, fmt.Errorf("array lengths of sensor definitions do not match")
	}

	result := make([]CimiSensor, len(parsedSensorModels))
	for i := 0; i < len(parsedSensorModels); i++ {
		result[i].HardwareModel = parsedSensorModels[i]
		result[i].Dimensions = parsedSensorTypes[i]
		result[i].ConnectionParameters = parsedSensorConnections[i]
	}
	return result, nil
}

func getCimiUser(connectionParams Mf2cConnectionParameters, username string) (*CimiUser, error) {
	var parsedResponse CimiUserList
	err := connectionParams.getUnmarshal("/api/user", &parsedResponse)
	if err != nil {
		return nil, err
	}
	for _, cu := range parsedResponse.Users {
		if cu.Username == username {
			return &cu, nil
		}
	}
	return nil, nil
}

func getSensorDriverService(connectionParams Mf2cConnectionParameters, container SensorDriverContainer) (*CimiService, error) {
	var parsedResponse CimiServiceList
	err := connectionParams.getUnmarshal("/api/service", &parsedResponse)
	if err != nil {
		return nil, err
	}
	for _, cs := range parsedResponse.Services {
		if cs.Name == container.getCimiServiceName() {
			return &cs, nil
		}
	}
	return nil, nil
}

func getSensorDriverServiceInstance(connectionParams Mf2cConnectionParameters, cimiService CimiService) (*CimiServiceInstance, error) {
	var parsedResponse CimiServiceInstanceList
	err := connectionParams.getUnmarshal("/api/service-instance", &parsedResponse)
	if err != nil {
		return nil, err
	}

	for _, csi := range parsedResponse.ServiceInstances {
		if csi.Service == cimiService.Id {
			return &csi, nil
		}
	}
	return nil, nil
}

func createCimiUser(connectionParams Mf2cConnectionParameters, username string, password string) error {
	req := CimiUserCreationRequest{}
	// cleaner to instantiate this way, nested structs are weird
	req.UserTemplate.Href = "user-template/self-registration"
	req.UserTemplate.Username = username
	req.UserTemplate.Password = password
	req.UserTemplate.PasswordRepeat = password
	req.UserTemplate.EmailAddress = fmt.Sprintf("%s@example.com", username)
	return connectionParams.post("/api/user", &req)
}

func createSensorDriverService(connectionParams Mf2cConnectionParameters, container SensorDriverContainer) error {
	// TODO: pass envvars, probably through docker-compose vars (will need to host a server for lifecycle)
	return connectionParams.post("/api/service", CimiService{
		Name:      container.getCimiServiceName(),
		Exec:      fmt.Sprintf("%s:%s", container.DockerImagePath, container.DockerImageVersion),
		ExecType:  "docker",
		AgentType: "normal",
	})
}

func startSensorDriverService(connectionParams Mf2cConnectionParameters, user CimiUser, service CimiService) error {
	return connectionParams.post("/api/v2/lm/service", LifecycleServiceStartRequest{
		ServiceId:   service.Id,
		UserId:      user.Id,
		AgreementId: "this-is-not-needed-yet-right?",
	})
}
