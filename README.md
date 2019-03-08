# mF2C Sensor Manager

## Sensor driver communication interface

The sensor driver is spawned with connection parameters in environment variables:

```
SENSOR_MANAGER_HOST=<host>
SENSOR_MANAGER_PORT=<port>
SENSOR_MANAGER_USERNAME=<username>
SENSOR_MANAGER_PASSWORD=<password>
``` 

It must publish sensor values to an MQTT server at that endpoint to the `/sensor-manager/sensor-incoming` topic.
Authenticate with the username and password in the environment variables. The values must conform to the following 
JSON structure:

```json
{
    "SensorId":   "string, an ID that differentiates this piece of hardware sensor from others",
    "SensorType": "string, the type of the sensor (hardware)",
    "Quantity":   "string, the SI dimension, e.g. humidity",
    "Value":      "float, the measurement value",
    "Unit":       "string, the base SI unit"
}
```
