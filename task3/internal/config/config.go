package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ConfigDetails struct {
	OpenWeather OpenWeatherConfig
	GraphHopper GraphHopperConfig
	Wikipedia   WikipediaConfig

	HTTPTimeout      time.Duration
	OperationTimeout time.Duration
	DetailsWorkers   int
}

type OpenWeatherConfig struct {
	APIKey       string
	GeocodingURL string
	WeatherURL   string
}

type GraphHopperConfig struct {
	APIKey       string
	GeocodingURL string
}

type WikipediaConfig struct {
	APIURL    string
	UserAgent string
}

// Load загружает .env, читает окружение и проверяет настройки.
func Load() (ConfigDetails, error) {
	if err := godotenv.Load(".env"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return ConfigDetails{}, fmt.Errorf("load .env: %w", err)
		}
	}

	openWeatherKey, err := requiredEnv("OPENWEATHER_API_KEY")
	if err != nil {
		return ConfigDetails{}, err
	}

	graphHopperKey, err := requiredEnv("GRAPHHOPPER_API_KEY")
	if err != nil {
		return ConfigDetails{}, err
	}

	wikipediaUserAgent, err := requiredEnv("WIKIPEDIA_USER_AGENT")
	if err != nil {
		return ConfigDetails{}, err
	}
	cfg := ConfigDetails{
		OpenWeather: OpenWeatherConfig{
			APIKey: openWeatherKey,
			GeocodingURL: envOrDefault(
				"OPENWEATHER_GEOCODING_URL",
				"https://api.openweathermap.org/geo/1.0/direct",
			),
			WeatherURL: envOrDefault(
				"OPENWEATHER_WEATHER_URL",
				"https://api.openweathermap.org/data/2.5/weather",
			),
		},

		GraphHopper: GraphHopperConfig{
			APIKey: graphHopperKey,
			GeocodingURL: envOrDefault(
				"GRAPHHOPPER_GEOCODING_URL",
				"https://graphhopper.com/api/1/geocode",
			),
		},

		Wikipedia: WikipediaConfig{
			APIURL: envOrDefault(
				"WIKIPEDIA_API_URL",
				"https://ru.wikipedia.org/w/api.php",
			),
			UserAgent: wikipediaUserAgent,
		},

		HTTPTimeout:      10 * time.Second,
		OperationTimeout: 30 * time.Second,
		DetailsWorkers:   4,
	}

	if err = cfg.validate(); err != nil {
		return ConfigDetails{}, err
	}

	return cfg, nil
}

func (cfg ConfigDetails) validate() error {
	endpoints := []struct {
		name  string
		value string
	}{
		{
			name:  "OPENWEATHER_GEOCODING_URL",
			value: cfg.OpenWeather.GeocodingURL,
		},
		{
			name:  "OPENWEATHER_WEATHER_URL",
			value: cfg.OpenWeather.WeatherURL,
		},
		{
			name:  "GRAPHHOPPER_GEOCODING_URL",
			value: cfg.GraphHopper.GeocodingURL,
		},
		{
			name:  "WIKIPEDIA_API_URL",
			value: cfg.Wikipedia.APIURL,
		},
	}

	for _, endpoint := range endpoints {
		if err := validateEndpoint(endpoint.name, endpoint.value); err != nil {
			return err
		}
	}

	if cfg.HTTPTimeout <= 0 {
		return fmt.Errorf("HTTPTimeout must be > 0")
	}

	if cfg.OperationTimeout < cfg.HTTPTimeout {
		return fmt.Errorf("OperationTimeout must be >= HTTPTimeout")
	}

	if cfg.DetailsWorkers <= 0 {
		return fmt.Errorf("DetailsWorkers must be > 0")
	}

	return nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func envOrDefault(name string, fallback string) string {
	value, exists := os.LookupEnv(name)
	if !exists {
		return fallback
	}

	return strings.TrimSpace(value)
}

func validateEndpoint(name string, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s: invalid URL", name)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: expected http or https scheme", name)
	}

	if u.Hostname() == "" {
		return fmt.Errorf("%s: host is required", name)
	}

	if u.User != nil {
		return fmt.Errorf("%s: credentials must not be included in URL", name)
	}

	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return fmt.Errorf(
			"%s: expected endpoint without query parameters or fragment",
			name,
		)
	}

	return nil
}
