package sensormanager

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// TODO: this will be removed. maybe.
const CimiAuthenticationHeaderKey = "Slipstream-Authn-Info"
const CimiAuthenticationBypassValue = "internal ADMIN"

const CimiSlaTemplateName = "sensor-manager-sla"

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

type CimiSlaTemplate struct {
	Id CimiIdentifier `json:"id"`
	// required
	Name    string `json:"name"`
	State   string `json:"state"`
	Details struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Provider struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"provider"`
		Client struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"client"`
		Creation   string `json:"creation"`
		Expiration string `json:"expiration"`
		Guarantees []struct {
			Name       string `json:"name"`
			Constraint string `json:"constraint"`
		} `json:"guarantees"`
	} `json:"details"`
	// optional
	Created string `json:"created,omitifempty"`
	Updated string `json:"updated,omitifempty"`
}

type CimiSlaTemplateList struct {
	Count     uint              `json:"count"`
	Templates []CimiSlaTemplate `json:"templates"`
}

// used in PUT /api/service
type CimiService struct {
	Id CimiIdentifier `json:"id"`
	// required
	Name         string     `json:"name"`
	Exec         string     `json:"exec"`
	ExecType     string     `json:"exec_type"`
	AgentType    string     `json:"agent_type"`
	NumAgents    uint       `json:"num_agents"`
	SlaTemplates []CimiHref `json:"sla_templates"`
	// optional
	Description       string   `json:"description,omitempty"`
	ExecPorts         []uint16 `json:"exec_ports,omitempty"`
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

type CimiDeviceDynamic struct {
	Device                      CimiHref                  `json:"device"`
	RamFree                     float32                   `json:"ramFree"`
	RamFreePercent              float32                   `json:"ramFreePercent"`
	StorageFree                 float32                   `json:"storageFree"`
	StorageFreePercent          float32                   `json:"storageFreePercent"`
	PowerRemainingStatus        string                    `json:"powerRemainingStatus"`
	PowerRemainingStatusSeconds string                    `json:"powerRemainingStatusSeconds"`
	EthernetAddress             string                    `json:"ethernetAddress"`
	WifiAddress                 string                    `json:"wifiAddress"`
	EthernetThroughputInfo      []string                  `json:"ethernetThroughputInfo"`
	WifiThroughputInfo          []string                  `json:"wifiThroughputInfo"`
	Sensors                     []CimiDeviceDynamicSensor `json:"sensors"`
	MyLeaderId                  CimiHref                  `json:"myLeaderID"`
}

type CimiDeviceDynamicList struct {
	Count          uint                `json:"count"`
	DeviceDynamics []CimiDeviceDynamic `json:"deviceDynamics"`
}

type CimiDeviceDynamicSensor struct {
	SensorType       string `json:"sensorType"`
	SensorModel      string `json:"sensorModel"`
	SensorConnection string `json:"sensorConnection"`
}

type LifecycleServiceStartRequest struct {
	ServiceId CimiIdentifier `json:"service_id"`
	UserId    CimiIdentifier `json:"user_id"`
	// this seems to be optional for now
	AgreementId CimiIdentifier `json:"agreement_id"`
}

func getSensorsFromCimi(connectionParams Mf2cConnectionParameters) ([]CimiSensor, error) {
	var parsedResponse CimiDeviceDynamicList
	err := connectionParams.getUnmarshal("/api/device-dynamic", &parsedResponse)
	if err != nil {
		return nil, err
	}

	if len(parsedResponse.DeviceDynamics) == 0 {
		return nil, fmt.Errorf("no device-dynamic resources exist")
	}

	// TODO: expand to multiple
	if len(parsedResponse.DeviceDynamics) != 1 {
		return nil, fmt.Errorf("more than one device-dynamic resource is not supported in this version, got: %d", len(parsedResponse.DeviceDynamics))
	}
	singleDeviceDynamic := parsedResponse.DeviceDynamics[0]

	// TODO: the values are returned in a bad format. sometimes wrong.
	//       this is just a note that upstream needs to change
	result := make([]CimiSensor, len(singleDeviceDynamic.Sensors))
	for i := 0; i < len(singleDeviceDynamic.Sensors); i++ {
		// nothing to parse here, it's a string already
		result[i].HardwareModel = singleDeviceDynamic.Sensors[i].SensorModel

		var sensorTypes []string
		err = json.Unmarshal([]byte(singleDeviceDynamic.Sensors[i].SensorType), &sensorTypes)
		if err != nil {
			return nil, err
		}
		result[i].Dimensions = sensorTypes

		var sensorConnection map[string]interface{}
		err = json.Unmarshal([]byte(singleDeviceDynamic.Sensors[i].SensorConnection), &sensorConnection)
		if err != nil {
			return nil, err
		}
		result[i].ConnectionParameters = sensorConnection
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

func getSlaTemplate(connectionParams Mf2cConnectionParameters, templateName string) (*CimiSlaTemplate, error) {
	var parsedResponse CimiSlaTemplateList
	err := connectionParams.getUnmarshal("/api/sla-template", &parsedResponse)
	if err != nil {
		return nil, err
	}
	for _, st := range parsedResponse.Templates {
		if st.Name == templateName {
			return &st, nil
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

func createSlaTemplate(connectionParams Mf2cConnectionParameters, templateName string) error {
	req := CimiSlaTemplate{}
	req.Name = templateName
	req.State = "started"
	req.Details.Type = "template"
	req.Details.Name = templateName
	req.Details.Provider.Id = "mf2c"
	req.Details.Provider.Name = "mF2C Platform"
	req.Details.Client.Id = "c02"
	req.Details.Client.Name = "clint"
	req.Details.Creation = time.Now().UTC().Format(time.RFC3339Nano)
	req.Details.Expiration = time.Now().Add(100 * 365 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	// this is ugly, but it at least gives a compilation error if it doesn't match the destination type
	req.Details.Guarantees = []struct {
		Name       string `json:"name"`
		Constraint string `json:"constraint"`
	}{
		{Name: "TestGuarantee", Constraint: "execution_time < 1234567890"},
	}
	return connectionParams.post("/api/sla-template", &req)
}

// accepts a SensorDriverContainer as the dot
const DockerComposeTemplate = `
version: "3.5"
services:
  sensor-driver:
    image: {{.DockerImagePath}}:{{.DockerImageVersion}}
    networks:
      - assigned_driver_network
    environment:
{{range .Environment}}
      - '{{.Key}}={{.Value}}'
{{end}}
networks:
  assigned_driver_network:
    name: {{.DockerNetworkName}}
    external: true
`

func createSensorDriverService(connectionParams Mf2cConnectionParameters, container SensorDriverContainer, slaTemplate CimiSlaTemplate) error {
	tpl, err := template.New("docker-compose").Parse(DockerComposeTemplate)
	if err != nil {
		return err
	}
	buffer := bytes.Buffer{}
	err = tpl.Execute(&buffer, container)
	if err != nil {
		return err
	}

	return connectionParams.post("/api/service", CimiService{
		Name:         container.getCimiServiceName(),
		Exec:         "data:application/x-yaml," + buffer.String(),
		ExecType:     "docker-compose",
		AgentType:    "normal",
		NumAgents:    1,
		SlaTemplates: []CimiHref{{Href: string(slaTemplate.Id)}},
	})
}

func startSensorDriverService(connectionParams Mf2cConnectionParameters, user CimiUser, service CimiService) error {
	return connectionParams.post("/api/v2/lm/service", LifecycleServiceStartRequest{
		ServiceId:   service.Id,
		UserId:      user.Id,
		AgreementId: "this-is-not-needed-yet-right?",
	})
}
