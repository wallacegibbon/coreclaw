#!/bin/bash
# Weather script - concise output for AI

if [ -z "$1" ]; then
	echo "Usage: weather <location>"
	exit 1
fi

LOCATION=$(echo "$*" | tr ' ' '+')
URL="https://wttr.in/${LOCATION}?format=j1"
DATA=$(curl -s "$URL")

if [ -z "$DATA" ] || echo "$DATA" | grep -q "404"; then
	echo "Error: Location not found"
	exit 1
fi

# Current: temp|Condition|humidity%|wind km/h
TEMP=$(echo "$DATA" | jq -r '.current_condition[0].temp_C')
COND=$(echo "$DATA" | jq -r '.current_condition[0].weatherDesc[0].value')
HUMID=$(echo "$DATA" | jq -r '.current_condition[0].humidity')
WIND=$(echo "$DATA" | jq -r '.current_condition[0].windspeedKmph')

echo "$* | ${TEMP}°C, $COND | Humidity: $HUMID% | Wind: ${WIND}km/h"

# Forecast: date|condition|temp
echo "$DATA" | jq -r '.weather[0,1,2] | "\(.date) | \(.hourly[3].weatherDesc[0].value) | \(.hourly[3].tempC)°C"'
