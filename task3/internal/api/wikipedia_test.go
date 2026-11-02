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

// Создаёт клиента, направленного на локальный тестовый сервер.
func newWikipediaTestClient(
	t *testing.T,
	statusCode int,
	body string,
	expectedParams map[string]string,
) *WikipediaClient {
	t.Helper()

	const userAgent = "PlacesLab-Test/0.1"

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

			if r.URL.Path != "/w/api.php" {
				t.Errorf(
					"path: got %q, want /w/api.php",
					r.URL.Path,
				)
			}

			if got := r.Header.Get("User-Agent"); got != userAgent {
				t.Errorf(
					"User-Agent: got %q, want %q",
					got,
					userAgent,
				)
			}

			params := r.URL.Query()

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

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)

			if _, err := w.Write([]byte(body)); err != nil {
				t.Errorf("write response: %v", err)
			}
		}),
	)

	t.Cleanup(server.Close)

	return NewWikipediaClient(
		server.Client(),
		config.WikipediaConfig{
			APIURL:    server.URL + "/w/api.php",
			UserAgent: userAgent,
		},
	)
}

func TestFindNearbyPlacesHTTP(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       []model.PlaceRef
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"query": {
					"geosearch": [
						{
							"pageid": 123,
							"title": "Тестовый парк",
							"lat": 55.04,
							"lon": 82.93,
							"dist": 450.5
						}
					]
				}
			}`,
			want: []model.PlaceRef{
				{
					PageID: 123,
					Title:  "Тестовый парк",
					Coordinates: model.Coordinates{
						Lat: 55.04,
						Lon: 82.93,
					},
					DistanceM: 450.5,
				},
			},
		},
		{
			name:       "empty result",
			statusCode: http.StatusOK,
			body:       `{"query":{"geosearch":[]}}`,
			want:       []model.PlaceRef{},
		},
		{
			name:       "API error with HTTP 200",
			statusCode: http.StatusOK,
			body: `{
				"error": {
					"code": "badvalue",
					"info": "Invalid coordinate"
				}
			}`,
			wantErr: true,
		},
		{
			name:       "missing query",
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    true,
		},
		{
			name:       "HTTP error",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"message":"Unavailable"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			body:       `{"query":`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newWikipediaTestClient(
				t,
				tt.statusCode,
				tt.body,
				map[string]string{
					"action":      "query",
					"list":        "geosearch",
					"gscoord":     "55.03|82.92",
					"gsradius":    "5000",
					"gslimit":     "10",
					"gsnamespace": "0",
					"format":      "json",
				},
			)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			got, err := client.FindNearbyPlaces(
				ctx,
				model.Coordinates{
					Lat: 55.03,
					Lon: 82.92,
				},
				5000,
				10,
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
					"unexpected places:\ngot:  %#v\nwant: %#v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestLoadPlaceDescriptionHTTP(t *testing.T) {
	place := model.PlaceRef{
		PageID: 123,
		Title:  "Старое название",
		Coordinates: model.Coordinates{
			Lat: 55.04,
			Lon: 82.93,
		},
		DistanceM: 450.5,
	}

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantTitle  string
		wantText   string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"query": {
					"pages": {
						"123": {
							"pageid": 123,
							"title": "Тестовый парк",
							"extract": "  Описание парка. \n"
						}
					}
				}
			}`,
			wantTitle: "Тестовый парк",
			wantText:  "Описание парка.",
		},
		{
			name:       "empty extract",
			statusCode: http.StatusOK,
			body: `{
				"query": {
					"pages": {
						"123": {
							"pageid": 123,
							"title": "Тестовый парк",
							"extract": ""
						}
					}
				}
			}`,
			wantTitle: "Тестовый парк",
			wantText:  "",
		},
		{
			name:       "missing page",
			statusCode: http.StatusOK,
			body: `{
				"query": {
					"pages": {
						"123": {
							"pageid": 123,
							"missing": ""
						}
					}
				}
			}`,
			wantErr: true,
		},
		{
			name:       "requested page not returned",
			statusCode: http.StatusOK,
			body:       `{"query":{"pages":{}}}`,
			wantErr:    true,
		},
		{
			name:       "API error with HTTP 200",
			statusCode: http.StatusOK,
			body: `{
				"error": {
					"code": "badvalue",
					"info": "Invalid page ID"
				}
			}`,
			wantErr: true,
		},
		{
			name:       "missing query",
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    true,
		},
		{
			name:       "HTTP error",
			statusCode: http.StatusTooManyRequests,
			body:       `{"message":"Too many requests"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			body:       `{"query":`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newWikipediaTestClient(
				t,
				tt.statusCode,
				tt.body,
				map[string]string{
					"action":      "query",
					"prop":        "extracts|pageimages",
					"pageids":     "123",
					"exintro":     "1",
					"explaintext": "1",
					"piprop":      "thumbnail",
					"pithumbsize": "480",
					"format":      "json",
				},
			)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			got, err := client.LoadPlaceDescription(ctx, place)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Координаты, расстояние и PageID должны сохраниться.
			wantPlace := place
			wantPlace.Title = tt.wantTitle

			want := model.PlaceDescription{
				Place:   wantPlace,
				Summary: tt.wantText,
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf(
					"unexpected description:\ngot:  %#v\nwant: %#v",
					got,
					want,
				)
			}
		})
	}
}
