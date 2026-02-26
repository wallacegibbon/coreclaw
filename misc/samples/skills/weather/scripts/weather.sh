#!/bin/sh
# Weather script - concise output for AI

MAX_RETRIES=3

while [ $# -gt 0 ]; do
	case "$1" in
		-n|--max-retries)
			MAX_RETRIES="$2"
			shift 2
			;;
		-*)
			echo "Usage: weather [-n N] <location>"
			exit 1
			;;
		*)
			break
			;;
	esac
done

if [ -z "$1" ]; then
	echo "Usage: weather [-n N] <location>"
	exit 1
fi

fetch_weather() {
	LOCATION=$(echo "$*" | tr ' ' '+')
	URL="https://wttr.in/${LOCATION}?format=j1"
	DATA=$(curl -s "$URL")

	if [ -z "$DATA" ] || echo "$DATA" | grep -q "404"; then
		echo "Error: Location not found" >&2
		return 1
	fi

	# Current: temp|Condition|humidity%|wind km/h
	TEMP=$(echo "$DATA" | jq -r '.current_condition[0].temp_C')
	COND=$(echo "$DATA" | jq -r '.current_condition[0].weatherDesc[0].value')
	HUMID=$(echo "$DATA" | jq -r '.current_condition[0].humidity')
	WIND=$(echo "$DATA" | jq -r '.current_condition[0].windspeedKmph')

	echo "$* | ${TEMP}°C, $COND | Humidity: $HUMID% | Wind: ${WIND}km/h"

	# Forecast: date|condition|temp
	echo "$DATA" | jq -r '.weather[0,1,2] | "\(.date) | \(.hourly[3].weatherDesc[0].value) | \(.hourly[3].tempC)°C"'
}

i=1
while [ $i -le "$MAX_RETRIES" ]; do
	if fetch_weather "$@"; then
		exit 0
	fi
	if [ "$i" -lt "$MAX_RETRIES" ]; then
		sleep 1
	fi
	i=$((i + 1))
done

echo "Error: Failed after $MAX_RETRIES attempts" >&2
exit 1
