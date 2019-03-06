#!/bin/sh

# will warn if variables are unset
set -e
set -u

echo "Configuring mosquitto to use the HTTP auth backend at ${MOSQUITTO_AUTH_HTTP_HOST}:${MOSQUITTO_AUTH_HTTP_PORT}"
cat >/etc/mosquitto/mosquitto.conf.d/mosquitto-auto.conf <<EOF
auth_opt_http_ip            ${MOSQUITTO_AUTH_HTTP_HOST}
auth_opt_http_port          ${MOSQUITTO_AUTH_HTTP_PORT}
# auth_opt_http_hostname
auth_opt_http_getuser_uri   /auth
auth_opt_http_superuser_uri /superuser
auth_opt_http_aclcheck_uri  /acl
auth_opt_http_with_tls      false
auth_opt_http_retry_count   5
EOF

/usr/local/sbin/mosquitto -c /etc/mosquitto/mosquitto.conf
