#!/bin/sh

# this deploys the sensor manager as an application to mF2C
# most commands courtesy of the hello world script

if [ -z $1 ]; then
    echo "Missing deployment target."
    echo "Usage: $0 <mf2c-IP>"
    exit 1
fi

mf2c_ip="$1"

set -e
set -u

alias mf2c-curl-post="curl -XPOST -ksS -H 'slipstream-authn-info:internal ADMIN' -H 'content-type:application/json' "

mf2c-curl-post "https://$mf2c_ip/api/user" -d '{
    "userTemplate": {
        "href": "user-template/self-registration",
        "password": "whydoesthisneedtoberepeatedintheapicallthisisridiculous",
        "passwordRepeat" : "whydoesthisneedtoberepeatedintheapicallthisisridiculous",
        "emailAddress": "tralala@example.com",
        "username": "sensormanagerouter"
    }
}'

# Create SLA template
SLA_TEMPLATE_ID=$(mf2c-curl-post "https://$mf2c_ip/api/sla-template" -d '{
    "name": "sensor-manager-sla",
    "state": "started",
    "details":{
        "type": "template",
        "name": "sensor-manager-sla",
        "provider": { "id": "mf2c", "name": "mF2C Platform" },
        "client": { "id": "c02", "name": "A client" },
        "creation": "2018-01-16T17:09:45.01Z",
        "expiration": "2100-01-17T17:09:45.01Z",
        "guarantees": [
            {
                "name": "uselessguarantee",
                "constraint": "execution_time < 1234567890"
            }
        ]
    }
}' | jq -es 'if . == [] then null else .[] | .["resource-id"] end') && \
    echo "SLA template $SLA_TEMPLATE_ID created successfully." || \
        (echo "Failed to create new SLA template: $SLA_TEMPLATE_ID" && exit 2)
SLA_TEMPLATE_ID=`echo ${SLA_TEMPLATE_ID} | tr -d '"'`

compose_file_contents_base64="$(base64 docker-compose.yml)"
SERVICE_ID=$(mf2c-curl-post "https://$mf2c_ip/api/service" -d '{
    "name": "sensor-manager-service",
    "description": "The sensor manager.",
    "exec": "data:application/x-yaml;base64,'"$compose_file_contents_base64"'",
    "exec_type": "docker-compose",
    "sla_templates": ["'"$SLA_TEMPLATE_ID"'"],
    "agent_type": "normal",
    "num_agents": 1,
    "cpu_arch": "x86-64",
    "os": "linux",
    "storage_min": 0,
    "req_resource": [],
    "opt_resource": []
}' | jq -es 'if . == [] then null else .[] | .["resource-id"] end') && \
    echo "Service $SERVICE_ID created successfully." || \
        (echo "Failed to create new service: $SERVICE_ID" && exit 3)
SERVICE_ID=`echo ${SERVICE_ID} | tr -d '"'`

mf2c-curl-post "http://$mf2c_ip:46000/api/v2/lm/service" -d '{"service_id": "'"$SERVICE_ID"'"}'
echo "Sensor manager service spawned successfully."
