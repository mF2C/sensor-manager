# mF2C Sensor Manager

Provides the ability for applications to subscribe to sensor data in conjunction with CIMI. 

## Data interface

Authentication details will be handled through CIMI and have yet to be implemented. They will consist of sensor-specific
credentials in the form of a username and password, which can be considered sensor manager-specific tokens.

The data interface is through MQTT. A call to CIMI (not yet implemented) returns a topic and connection credentials
that allow subscribing to it. As details about the sensors are obtained through CIMI from the categorisation module,
the returned data is in the bare format of

```json
{
    "Timestamp": "string, RFC 3339, when the measurement was taken",
    "Value":     "float, the measurement value",
    "Unit":      "string, the SI base unit of the value"
}
```

This data is not stored and is only available through QoS level 0.

## Sensor driver communication interface

The sensor driver is spawned with connection parameters in environment variables:

```
SENSOR_MANAGER_HOST=<host>
SENSOR_MANAGER_PORT=<port>
SENSOR_MANAGER_USERNAME=<username>
SENSOR_MANAGER_PASSWORD=<password>
SENSOR_MANAGER_TOPIC=<topic>
``` 

It must publish sensor values to an MQTT server at that endpoint to the specified topic, authenticated with 
the username and password in the environment variables. The values must conform to the following 
JSON structure:

```json
{
    "SensorId":   "string, an ID that differentiates this piece of hardware sensor from others",
    "SensorType": "string, the type of the sensor (hardware)",
    "Quantity":   "string, the SI dimension, e.g. humidity",
    "Timestamp":  "string, RFC 3339, when the measurement was taken",
    "Value":      "float, the measurement value",
    "Unit":       "string, the SI base unit of the value"
}
```

## About the MQTT backend

The sensor manager requires an MQTT server capable of username-password-based authentication and per-topic authorization.
This is achieved with [https://github.com/jpmens/mosquitto-auth-plug/](https://github.com/jpmens/mosquitto-auth-plug/),
which provides a versatile authentication plugin to Eclipse Mosquitto. We use the HTTP authentication backend, served
by the sensor manager, which allows it control over all communication.
