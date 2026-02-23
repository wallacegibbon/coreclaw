---
name: weather
description: Use this skill whenever the user wants to get weather information. This includes current weather, forecasts, temperature, humidity, wind, and weather conditions for any city or region.
---

# Weather Skill

Get weather information using the weather script.

## Usage

```bash
./scripts/weather.sh "New York"
./scripts/weather.sh "London"
./scripts/weather.sh "Tokyo"
```

The script fetches weather data from wttr.in in JSON format and displays it as a formatted table.

## Output

- Current conditions: temperature, feels-like, humidity, wind speed, UV index, visibility, pressure
- 3-day forecast: date, condition, temperature

## Requirements

- curl (for fetching data)
- jq (for parsing JSON)

## Example

```bash
./scripts/weather.sh "New York"
```

Output:
```
┌─────────────────────────────────────────┐
│ Weather: New York                     │
└─────────────────────────────────────────┘

Temperature:    -2°C (feels -10°C)
Condition:      Snow, freezing fog
Humidity:       96%
Wind:           45 km/h
UV Index:       0
Visibility:     0 km
Pressure:       999 mb

┌─────────────────────────────────────────┐
│ 3-Day Forecast                        │
└─────────────────────────────────────────┘

2026-02-23 | Blowing snow | -2°C
2026-02-24 | Sunny | -6°C
2026-02-25 | Moderate snow | -3°C
```
