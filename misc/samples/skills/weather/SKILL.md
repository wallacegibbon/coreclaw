---
name: weather
description: Use this skill whenever the user wants to get weather information. This includes current weather, forecasts, temperature, humidity, wind, and weather conditions for any city or region.
---

# Weather Skill

Get weather information using the weather script.

## Usage

```bash
./scripts/weather.sh [-n N] "City name"
```

- `-n N` - Number of retries on failure (default: 3)
- **Note**: Use English city names (e.g., "Wuhan" not "武汉")

The script fetches weather data from wttr.in in JSON format.

## Output

Current: `location | temp°C, condition | Humidity: X% | Wind: Ykm/h`
Forecast: `date | condition | temp°C`

## Example

```bash
./scripts/weather.sh "New York"
```

Output:
```
New York | 18°C, Partly cloudy | Humidity: 65% | Wind: 12km/h
2026-02-23 | Partly Cloudy | 18°C
2026-02-24 | Sunny | 20°C
2026-02-25 | Light Rain | 16°C
```
