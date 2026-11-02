package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"task3/internal/config"
	"task3/internal/model"
)

func TestSearchLocationsHTTP(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       []model.Location
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"hits": [
					{
						"name": "Тестовый объект",
						"country": "Россия",
						"state": "Тестовая область",
						"city": "Тестовый город",
						"street": "Тестовая улица",
						"housenumber": "10",
						"point": {
							"lat": 55.03,
							"lng": 82.92
						}
					}
				]
			}`,
			want: []model.Location{
				{
					Name:        "Тестовый объект",
					Country:     "Россия",
					Region:      "Тестовая область",
					City:        "Тестовый город",
					Street:      "Тестовая улица",
					HouseNumber: "10",
					Coordinates: model.Coordinates{
						Lat: 55.03,
						Lon: 82.92,
					},
				},
			},
		},
		{
			name:       "empty result",
			statusCode: http.StatusOK,
			body:       `{"hits":[]}`,
			want:       []model.Location{},
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
			body:       `{"hits":`,
			wantErr:    true,
		},
		{
			name:       "missing coordinates",
			statusCode: http.StatusOK,
			body:       `{"hits":[{"name":"Объект без координат"}]}`,
			wantErr:    true,
		},
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

					if r.URL.Path != "/geocode" {
						t.Errorf(
							"path: got %q, want /geocode",
							r.URL.Path,
						)
					}

					params := r.URL.Query()

					expectedParams := map[string]string{
						"q":      "Новосибирск, Цветной проезд",
						"key":    "test-key",
						"locale": "ru",
						"limit":  "5",
					}

					for name, want := range expectedParams {
						if got := params.Get(name); got != want {
							t.Errorf(
								"parameter %s: got %q, want %q",
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

			client := NewGraphHopperClient(
				server.Client(),
				config.GraphHopperConfig{
					APIKey:       "test-key",
					GeocodingURL: server.URL + "/geocode",
				},
			)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			got, err := client.SearchLocations(
				ctx,
				"Новосибирск, Цветной проезд",
			)

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
					"unexpected locations:\ngot:  %#v\nwant: %#v",
					got,
					tt.want,
				)
			}
		})
	}
}
