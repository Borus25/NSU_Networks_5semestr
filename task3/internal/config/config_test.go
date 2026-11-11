package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEndpoint(t *testing.T) {
	for _, endpoint := range []string{"https://example.org/api", "http://localhost:1234/api"} {
		if err := validateEndpoint("ENDPOINT", endpoint); err != nil {
			t.Fatal(err)
		}
	}
	for _, endpoint := range []string{"", "file:///etc/passwd", "example.org/api", "https://user:pass@example.org/api", "https://example.org/api?key=SECRET", "https://example.org/api#x"} {
		if err := validateEndpoint("ENDPOINT", endpoint); err == nil {
			t.Fatalf("accepted %q", endpoint)
		} else if strings.Contains(err.Error(), "SECRET") {
			t.Fatal("URL secret leaked")
		}
	}
}
func TestLoadWithEnvironmentAndMalformedDotenv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GRAPHHOPPER_API_KEY", " graph ")
	t.Setenv("OPENWEATHER_API_KEY", " weather ")
	t.Setenv("WIKIPEDIA_USER_AGENT", "PlacesTest/1.0 (test@example.org)")
	t.Setenv("GRAPHHOPPER_GEOCODING_URL", "https://graphhopper.com/api/1/geocode")
	t.Setenv("OPENWEATHER_WEATHER_URL", "https://api.openweathermap.org/data/2.5/weather")
	t.Setenv("OPENWEATHER_GEOCODING_URL", "https://api.openweathermap.org/geo/1.0/direct")
	t.Setenv("WIKIPEDIA_API_URL", "https://ru.wikipedia.org/w/api.php")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GraphHopper.APIKey != "graph" || cfg.OpenWeather.APIKey != "weather" {
		t.Fatal("keys not trimmed")
	}
	if cfg.HTTPTimeout <= 0 || cfg.OperationTimeout < cfg.HTTPTimeout {
		t.Fatal("bad timeouts")
	}
	t.Setenv("GRAPHHOPPER_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("missing key accepted")
	}
	t.Setenv("GRAPHHOPPER_API_KEY", "graph")
	if err := os.WriteFile(filepath.Join(".", ".env"), []byte("THIS IS NOT A VALID ENV LINE !!!"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("invalid .env accepted")
	}
}
func TestDotenvDoesNotOverrideEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GRAPHHOPPER_API_KEY", "environment")
	t.Setenv("OPENWEATHER_API_KEY", "weather")
	t.Setenv("WIKIPEDIA_USER_AGENT", "test")
	if err := os.WriteFile(".env", []byte("GRAPHHOPPER_API_KEY=file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GraphHopper.APIKey != "environment" {
		t.Fatal(".env overwrote environment")
	}
}
