package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"task3/internal/asynchttp"
	"task3/internal/config"
	"task3/internal/model"
	"task3/internal/service"

	"github.com/reactivex/rxgo/v2"
)

func apiItem(t *testing.T, ctx context.Context, source rxgo.Observable) rxgo.Item {
	t.Helper()
	select {
	case item, ok := <-source.Observe():
		if !ok {
			t.Fatal("empty API stream")
		}
		return item
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	return rxgo.Item{}
}

func TestAPIRequestsAndResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch r.URL.Path {
		case "/geocode":
			if q.Get("q") != "Цветной проезд & дом" || q.Get("key") != "key" || q.Get("locale") != "ru" || q.Get("limit") != "5" {
				t.Errorf("unexpected query %v", q)
			}
			_, _ = io.WriteString(w, `{"hits":[{"name":"Дом","city":"Новосибирск","point":{"lat":54.8,"lng":83.1}}]}`)
		case "/weather":
			if q.Get("lat") != "54.8" || q.Get("lon") != "83.1" || q.Get("appid") != "key" || q.Get("units") != "metric" || q.Get("lang") != "ru" {
				t.Errorf("unexpected query %v", q)
			}
			_, _ = io.WriteString(w, `{"main":{"temp":2,"feels_like":-1,"humidity":80},"wind":{"speed":3},"weather":[{"description":"облачно"}]}`)
		case "/wiki":
			if r.Header.Get("User-Agent") != "Task3Test/1.0 (test@example.org)" {
				t.Error("missing User-Agent")
			}
			if q.Get("list") == "geosearch" {
				if q.Get("gscoord") != "54.8|83.1" || q.Get("gsradius") != "1000" || q.Get("gslimit") != "2" {
					t.Errorf("unexpected query %v", q)
				}
				_, _ = io.WriteString(w, `{"query":{"geosearch":[{"pageid":42,"title":"Место","lat":54.81,"lon":83.11,"dist":23}]}}`)
			} else {
				if q.Get("pageids") != "42" || q.Get("prop") != "extracts|pageimages" || q.Get("explaintext") != "1" {
					t.Errorf("unexpected query %v", q)
				}
				_, _ = io.WriteString(w, `{"query":{"pages":{"42":{"pageid":42,"title":"Новое название","extract":" Описание "}}}}`)
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client := asynchttp.New(server.Client())
	gh := NewGraphHopperClient(client, config.GraphHopperConfig{APIKey: "key", GeocodingURL: server.URL + "/geocode"})
	weather := NewOpenWeatherClient(client, config.OpenWeatherConfig{APIKey: "key", WeatherURL: server.URL + "/weather"})
	wiki := NewWikipediaClient(client, config.WikipediaConfig{APIURL: server.URL + "/wiki", UserAgent: "Task3Test/1.0 (test@example.org)"})
	item := apiItem(t, ctx, gh.SearchLocations(ctx, "  Цветной проезд & дом  "))
	if item.Error() {
		t.Fatal(item.E)
	}
	loc := item.V.([]model.Location)[0]
	if loc.Name != "Дом" || loc.Coordinates.Lon != 83.1 {
		t.Fatal(loc)
	}
	item = apiItem(t, ctx, weather.LoadWeather(ctx, loc.Coordinates))
	if item.Error() {
		t.Fatal(item.E)
	}
	if item.V.(model.Weather).HumidityPct != 80 {
		t.Fatal(item)
	}
	item = apiItem(t, ctx, wiki.FindNearbyPlaces(ctx, loc.Coordinates, 1000, 2))
	if item.Error() {
		t.Fatal(item.E)
	}
	place := item.V.([]model.PlaceRef)[0]
	item = apiItem(t, ctx, wiki.LoadPlaceDescription(ctx, place))
	if item.Error() {
		t.Fatal(item.E)
	}
	description := item.V.(model.PlaceDescription)
	if description.Place.PageID != 42 || description.Place.Title != "Новое название" || description.Summary != "Описание" {
		t.Fatal(description)
	}
}

func TestMalformedAndMissingAPIData(t *testing.T) {
	cases := []struct{ name, body, kind string }{
		{"invalid JSON", "{", "weather"}, {"null weather", "null", "weather"}, {"missing weather", "{}", "weather"},
		{"null geocode", "null", "geocode"}, {"missing coordinates", `{"hits":[{"name":"A"}]}`, "geocode"},
		{"wiki error", `{"error":{"code":"badvalue","info":"bad radius"}}`, "nearby"},
		{"null nearby", "null", "nearby"}, {"missing query", "{}", "nearby"},
		{"missing article", `{"query":{"pages":{"42":{"pageid":42,"missing":""}}}}`, "description"},
		{"wrong article", `{"query":{"pages":{"43":{"pageid":43}}}}`, "description"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, tc.body) }))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			client := asynchttp.New(server.Client())
			wiki := NewWikipediaClient(client, config.WikipediaConfig{APIURL: server.URL, UserAgent: "test"})
			var source rxgo.Observable
			switch tc.kind {
			case "weather":
				source = NewOpenWeatherClient(client, config.OpenWeatherConfig{WeatherURL: server.URL}).LoadWeather(ctx, model.Coordinates{})
			case "geocode":
				source = NewGraphHopperClient(client, config.GraphHopperConfig{GeocodingURL: server.URL}).SearchLocations(ctx, "A")
			case "nearby":
				source = wiki.FindNearbyPlaces(ctx, model.Coordinates{}, 1000, 5)
			default:
				source = wiki.LoadPlaceDescription(ctx, model.PlaceRef{PageID: 42})
			}
			if item := apiItem(t, ctx, source); !item.Error() {
				t.Fatalf("accepted malformed response %+v", item)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestThumbnailDecodingAndRestrictions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	for _, format := range []string{"png", "jpeg"} {
		t.Run(format, func(t *testing.T) {
			var buffer bytes.Buffer
			if format == "png" {
				if err := png.Encode(&buffer, img); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := jpeg.Encode(&buffer, img, nil); err != nil {
					t.Fatal(err)
				}
			}
			var calls atomic.Int32
			transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Host != "upload.wikimedia.org" || r.Header.Get("User-Agent") != "test" {
					t.Error("unexpected thumbnail request")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/octet-stream"}}, Body: io.NopCloser(bytes.NewReader(buffer.Bytes())), Request: r}, nil
			})
			wiki := NewWikipediaClient(asynchttp.New(&http.Client{Transport: transport}), config.WikipediaConfig{UserAgent: "test"})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			item := apiItem(t, ctx, wiki.loadThumbnail(ctx, "https://upload.wikimedia.org/example"))
			if item.Error() {
				t.Fatal(item.E)
			}
			if item.V.(image.Image).Bounds().Dy() != 3 {
				t.Fatal("wrong dimensions")
			}
			for _, url := range []string{"http://upload.wikimedia.org/x", "https://upload.wikimedia.org.evil.test/x", "https://upload.wikimedia.org:443/x", "https://user@upload.wikimedia.org/x", "file:///etc/passwd"} {
				if item := apiItem(t, ctx, wiki.loadThumbnail(ctx, url)); !item.Error() {
					t.Fatalf("accepted %s", url)
				}
			}
			if calls.Load() != 1 {
				t.Fatal("unsafe URL reached transport")
			}
		})
	}
}

func TestThumbnailFailurePreservesArticle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"query":{"pages":{"42":{"pageid":42,"title":"A","extract":"text","thumbnail":{"source":"https://evil.test/a.png"}}}}}`)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wiki := NewWikipediaClient(asynchttp.New(server.Client()), config.WikipediaConfig{APIURL: server.URL})
	item := apiItem(t, ctx, wiki.LoadPlaceDescription(ctx, model.PlaceRef{PageID: 42}))
	if item.Error() {
		t.Fatal(item.E)
	}
	p := item.V.(model.PlaceDescription)
	if p.Summary != "text" || p.ImageError == "" || p.Image != nil {
		t.Fatal(p)
	}
}

func TestPreviewWithoutArticleEmitsTypedNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"query":{"pages":{"-1":{"title":"Missing","missing":""}}}}`)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	wiki := NewWikipediaClient(asynchttp.New(server.Client()), config.WikipediaConfig{APIURL: server.URL})
	item := apiItem(t, ctx, wiki.LoadLocationPreview(ctx, "Missing"))
	if item.Error() {
		t.Fatal(item.E)
	}
	preview, ok := item.V.(*model.LocationPreview)
	if !ok || preview != nil {
		t.Fatalf("unexpected %#v", item.V)
	}
}

func TestRealHTTPGraphStartsEveryDescriptionAndPublishesOnce(t *testing.T) {
	const count = 6
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	weatherStarted, nearbyStarted, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	descriptions := make(chan int64, count)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		if r.URL.Path == "/weather" {
			close(weatherStarted)
			select {
			case <-nearbyStarted:
			case <-r.Context().Done():
				return
			}
			_, _ = io.WriteString(w, `{"main":{"temp":1,"feels_like":0,"humidity":90},"wind":{"speed":2},"weather":[{"description":"ok"}]}`)
			return
		}
		if r.URL.Query().Get("list") == "geosearch" {
			close(nearbyStarted)
			select {
			case <-weatherStarted:
			case <-r.Context().Done():
				return
			}
			refs := make([]map[string]any, count)
			for i := range refs {
				refs[i] = map[string]any{"pageid": i + 1, "title": fmt.Sprint(i + 1), "dist": count - i}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"query": map[string]any{"geosearch": refs}})
			return
		}
		var id int64
		_, _ = fmt.Sscan(r.URL.Query().Get("pageids"), &id)
		descriptions <- id
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = fmt.Fprintf(w, `{"query":{"pages":{"%d":{"pageid":%d,"title":"article","extract":"text"}}}}`, id, id)
	}))
	defer server.Close()
	client := asynchttp.New(server.Client())
	svc := service.New(nil, NewOpenWeatherClient(client, config.OpenWeatherConfig{WeatherURL: server.URL + "/weather"}), NewWikipediaClient(client, config.WikipediaConfig{APIURL: server.URL + "/wiki", UserAgent: "test"}))
	items := svc.LoadLocationInfo(ctx, model.Location{Name: "chosen"}, service.DetailsOptions{RadiusM: 1000, Limit: count}).Observe()
	for i := 0; i < count; i++ {
		select {
		case <-descriptions:
		case <-ctx.Done():
			t.Fatal("HTTP descriptions were serialized")
		}
	}
	select {
	case item := <-items:
		t.Fatalf("premature report %+v", item)
	default:
	}
	close(release)
	select {
	case item := <-items:
		if item.Error() {
			t.Fatal(item.E)
		}
		info := item.V.(model.LocationInfo)
		if len(info.Places) != count || info.WeatherError != "" || info.PlacesError != "" {
			t.Fatal(info)
		}
		ids := make([]int64, count)
		for i, p := range info.Places {
			ids[i] = p.Place.PageID
		}
		if !reflect.DeepEqual(ids, []int64{6, 5, 4, 3, 2, 1}) {
			t.Fatal(ids)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case _, ok := <-items:
		if ok {
			t.Fatal("multiple reports")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestInvalidSearchDoesNotReachHTTP(t *testing.T) {
	client := NewGraphHopperClient(asynchttp.New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("unexpected HTTP request")
		return nil, fmt.Errorf("should not run")
	})}), config.GraphHopperConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	item := apiItem(t, ctx, client.SearchLocations(ctx, strings.Repeat(" ", 3)))
	if !item.Error() {
		t.Fatal("empty search accepted")
	}
}

func TestThumbnailRejectsCorruptDataAndExcessiveDimensions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var small bytes.Buffer
	if err := png.Encode(&small, img); err != nil {
		t.Fatal(err)
	}
	oversized := append([]byte(nil), small.Bytes()...)
	binary.BigEndian.PutUint32(oversized[16:20], 5000)
	binary.BigEndian.PutUint32(oversized[20:24], 5000)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	for _, tc := range []struct {
		name string
		body []byte
		want string
	}{
		{"corrupt", []byte("not an image"), "dimensions"},
		{"too many pixels", oversized, "dimensions exceed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(tc.body)), Request: r}, nil
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			wiki := NewWikipediaClient(asynchttp.New(&http.Client{Transport: transport}), config.WikipediaConfig{})
			item := apiItem(t, ctx, wiki.loadThumbnail(ctx, "https://upload.wikimedia.org/x.png"))
			if !item.Error() || !strings.Contains(item.E.Error(), tc.want) {
				t.Fatalf("unexpected result %+v", item)
			}
		})
	}
}

func TestCancellationOfRealHTTPGraph(t *testing.T) {
	started, stopped := make(chan struct{}, 2), make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
		stopped <- struct{}{}
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := asynchttp.New(server.Client())
	svc := service.New(nil, NewOpenWeatherClient(client, config.OpenWeatherConfig{WeatherURL: server.URL + "/weather"}), NewWikipediaClient(client, config.WikipediaConfig{APIURL: server.URL + "/wiki", UserAgent: "test"}))
	items := svc.LoadLocationInfo(ctx, model.Location{}, service.DetailsOptions{RadiusM: 1000, Limit: 5}).Observe()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("HTTP branch never started")
		}
	}
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("HTTP branch was not cancelled")
		}
	}
	select {
	case item, ok := <-items:
		if ok && !item.Error() {
			t.Fatal("cancelled operation published a successful report")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not terminate")
	}
}
