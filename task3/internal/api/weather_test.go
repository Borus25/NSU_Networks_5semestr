package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"task3/internal/config"
	"task3/internal/model"
)

func TestLoadWeatherHTTP(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       model.Weather
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"main": {
					"temp": -5.5,
					"feels_like": -10.2,
					"humidity": 75
				},
				"wind": {
					"speed": 3.4
				},
				"weather": [
					{
						"description": "небольшой снег"
					}
				]
			}`,
			want: model.Weather{
				TemperatureC: -5.5,
				FeelsLikeC:   -10.2,
				HumidityPct:  75,
				WindSpeedMPS: 3.4,
				Description:  "небольшой снег",
			},
		},
		{
			name:       "zero values are valid",
			statusCode: http.StatusOK,
			body: `{
				"main": {
					"temp": 0,
					"feels_like": 0,
					"humidity": 0
				},
				"wind": {
					"speed": 0
				},
				"weather": [
					{
						"description": "ясно"
					}
				]
			}`,
			want: model.Weather{
				Description: "ясно",
			},
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"message":"Invalid API key"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			body:       `{"main":`,
			wantErr:    true,
		},
		{
			name:       "missing main",
			statusCode: http.StatusOK,
			body: `{
				"wind": {"speed": 2},
				"weather": [{"description": "ясно"}]
			}`,
			wantErr: true,
		},
		{
			name:       "missing wind",
			statusCode: http.StatusOK,
			body: `{
				"main": {
					"temp": 10,
					"feels_like": 9,
					"humidity": 50
				},
				"weather": [{"description": "ясно"}]
			}`,
			wantErr: true,
		},
		{
			name:       "empty weather array",
			statusCode: http.StatusOK,
			body: `{
				"main": {
					"temp": 10,
					"feels_like": 9,
					"humidity": 50
				},
				"wind": {"speed": 2},
				"weather": []
			}`,
			wantErr: true,
		},
	}

	coordinates := model.Coordinates{
		Lat: 55.03,
		Lon: 82.92,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					if r.Method != http.MethodGet {
						t.Errorf(
							"method: got %s, want GET",
							r.Method,
						)
					}

					if r.URL.Path != "/weather" {
						t.Errorf(
							"path: got %q, want /weather",
							r.URL.Path,
						)
					}

					params := r.URL.Query()

					for name, want := range map[string]string{
						"appid": "test-key",
						"units": "metric",
						"lang":  "ru",
					} {
						if got := params.Get(name); got != want {
							t.Errorf(
								"parameter %s: got %q, want %q",
								name,
								got,
								want,
							)
						}
					}

					// Сравниваем числовые значения:
					// разное текстовое представление допустимо.
					for name, want := range map[string]float64{
						"lat": coordinates.Lat,
						"lon": coordinates.Lon,
					} {
						got, err := strconv.ParseFloat(
							params.Get(name),
							64,
						)
						if err != nil {
							t.Errorf(
								"parameter %s is not a number: %v",
								name,
								err,
							)
							continue
						}

						if got != want {
							t.Errorf(
								"parameter %s: got %v, want %v",
								name,
								got,
								want,
							)
						}
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)
					w.WriteHeader(tt.statusCode)

					if _, err := w.Write([]byte(tt.body)); err != nil {
						t.Errorf("write response: %v", err)
					}
				}),
			)
			defer server.Close()

			client := NewOpenWeatherClient(
				server.Client(),
				config.OpenWeatherConfig{
					APIKey:     "test-key",
					WeatherURL: server.URL + "/weather",
				},
			)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			got, err := client.LoadWeather(ctx, coordinates)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf(
					"unexpected weather:\ngot:  %#v\nwant: %#v",
					got,
					tt.want,
				)
			}
		})
	}
}
