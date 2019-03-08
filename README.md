# mF2C Sensor Manager

## Sensor driver communication interface

The sensor driver is spawned with connection parameters in environment variables:

```
SENSOR_MANAGER_HOST=<host>
SENSOR_MANAGER_PORT=<port>
``` 

It must publish sensor values to an MQTT server at that endpoint to the `/sensor-manager/sensor-incoming` topic.
No credentials required. The values must conform to the following JSON structure:

```json
{
    "SensorId":   "string, an ID that differentiates this piece of hardware sensor from others",
    "SensorType": "string, the type of the sensor (hardware)",
    "Quantity":   "string, the SI dimension, e.g. humidity",
    "Value":      "float, the measurement value",
    "Unit":       "string, the base SI unit"
}
```
