package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"go.uber.org/mock/gomock"
)

// Mock implementations for testing using Uber's gomock
type MockWeatherReporter struct {
	ctrl     *gomock.Controller
	recorder *MockWeatherReporterMockRecorder
}

type MockWeatherReporterMockRecorder struct {
	mock *MockWeatherReporter
}

func NewMockWeatherReporter(ctrl *gomock.Controller) *MockWeatherReporter {
	mock := &MockWeatherReporter{ctrl: ctrl}
	mock.recorder = &MockWeatherReporterMockRecorder{mock}
	return mock
}

func (m *MockWeatherReporter) EXPECT() *MockWeatherReporterMockRecorder {
	return m.recorder
}

func (m *MockWeatherReporter) GenerateReport(ctx context.Context, localization string) (*application.GeneralWeatherInfo, error) {
	ret := m.ctrl.Call(m, "GenerateReport", ctx, localization)
	ret0, _ := ret[0].(*application.GeneralWeatherInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockWeatherReporterMockRecorder) GenerateReport(ctx, localization interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateReport", reflect.TypeOf((*MockWeatherReporter)(nil).GenerateReport), ctx, localization)
}

type MockVideoStreamReporter struct {
	ctrl     *gomock.Controller
	recorder *MockVideoStreamReporterMockRecorder
}

type MockVideoStreamReporterMockRecorder struct {
	mock *MockVideoStreamReporter
}

func NewMockVideoStreamReporter(ctrl *gomock.Controller) *MockVideoStreamReporter {
	mock := &MockVideoStreamReporter{ctrl: ctrl}
	mock.recorder = &MockVideoStreamReporterMockRecorder{mock}
	return mock
}

func (m *MockVideoStreamReporter) EXPECT() *MockVideoStreamReporterMockRecorder {
	return m.recorder
}

func (m *MockVideoStreamReporter) GenerateReport(ctx context.Context, localization string) (*application.VideosStream, error) {
	ret := m.ctrl.Call(m, "GenerateReport", ctx, localization)
	ret0, _ := ret[0].(*application.VideosStream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockVideoStreamReporterMockRecorder) GenerateReport(ctx, localization interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateReport", reflect.TypeOf((*MockVideoStreamReporter)(nil).GenerateReport), ctx, localization)
}

var testStore *sqlite.Store
var db *sql.DB // Separate test connection for seeding and inspecting the fixture database.

// TestMain gives the package one SQLite database. Tests that fetch fresh data each use their
// own city, so a background cache save from one test never turns another test's miss into a hit.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "traveltab-test")
	if err != nil {
		log.Fatal(err)
	}

	testStore, err = sqlite.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		log.Fatal(err)
	}
	db, err = sql.Open("sqlite3", filepath.Join(dir, "test.db"))
	if err != nil {
		log.Fatal(err)
	}
	// Production commands run from the repository root, where views and public live.
	if err := os.Chdir("../../.."); err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	_ = db.Close()
	_ = testStore.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// Test helper functions
func createTestServer(ctrl *gomock.Controller) (*Server, *MockWeatherReporter, *MockVideoStreamReporter) {
	mockWeather := NewMockWeatherReporter(ctrl)
	mockVideos := NewMockVideoStreamReporter(ctrl)

	server := NewAppServer(mockWeather, mockVideos, testStore)
	return server, mockWeather, mockVideos
}

func createTestGeneralWeatherInfo() *application.GeneralWeatherInfo {
	return &application.GeneralWeatherInfo{
		Weather: application.Weather{
			Temperature: 25.0,
			FeelsLike:   26.0,
			Wind:        10.0,
			Humidity:    60.0,
			Condition:   "sunny",
		},
		Waves: application.Waves{
			Height: 1.5,
		},
		City:     "Lisbon",
		Country:  "portugal",
		Lon:      -9.1393,
		Lat:      38.7223,
		EmbedURL: "https://embed.waze.com/iframe?zoom=10&lat=38.7223&lon=-9.1393",
	}
}

func createTestGeneralWeatherInfoFor(city string) *application.GeneralWeatherInfo {
	generalInfo := createTestGeneralWeatherInfo()
	generalInfo.City = city
	return generalInfo
}

func createTestVideosStream() *application.VideosStream {
	return &application.VideosStream{
		{Title: "Test Video 1", VideoID: "abc123"},
		{Title: "Test Video 2", VideoID: "def456"},
	}
}

func createTestHotels() *application.Hotels {
	return &application.Hotels{
		{
			HotelName:    "Test Hotel",
			HotelURL:     "https://example.com/hotel",
			HotelPrice:   "$100",
			HotelRating:  4.5,
			HotelAddress: "123 Test St",
			HotelMapURL:  "https://maps.google.com",
			ContactPhone: "+1234567890",
			PriceRange:   "$100-$200",
			HotelPhotos:  []string{"photo1.jpg", "photo2.jpg"},
			HotelReviews: []application.HotelReview{
				{AuthorName: "John Doe", Text: "Great hotel!", Rating: 5.0},
			},
		},
	}
}

func videosQuery(city string) string {
	return "Turistic places in " + city + ", portugal"
}

// expectFreshData sets up the video reporter for one cache miss on city.
func expectFreshData(mockVideos *MockVideoStreamReporter, city string) {
	mockVideos.EXPECT().GenerateReport(gomock.Any(), videosQuery(city)).Return(createTestVideosStream(), nil)
}

// doRequest sends req through the server's real routes and returns the status and body.
func doRequest(t *testing.T, server *Server, req *http.Request) (int, string) {
	t.Helper()

	resp, err := server.app.Test(req, 5000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

// waitForCache waits for the background save started by retireveFreshInformation.
func waitForCache(t *testing.T, city string) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := testStore.GetCityData(city)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "city %s was never cached", city)
}

// forgetCity removes city from the cache so the test starts with a cache miss, even when
// the tests run more than once with -count.
func forgetCity(t *testing.T, city string) {
	t.Helper()

	_, err := db.Exec(`DELETE FROM city_data WHERE city = ?`, strings.ToLower(city))
	require.NoError(t, err)
}

// callRetrieveFreshInformation runs retireveFreshInformation outside of a real request.
func callRetrieveFreshInformation(server *Server, generalInfo *application.GeneralWeatherInfo) (TemplateData, error) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	return server.retireveFreshInformation(ctx, generalInfo, generalInfo.City)
}

// Tests for NewAppServer
func TestNewAppServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWeather := NewMockWeatherReporter(ctrl)
	mockVideos := NewMockVideoStreamReporter(ctrl)

	server := NewAppServer(mockWeather, mockVideos, testStore)

	assert.NotNil(t, server)
	assert.NotNil(t, server.app)
	assert.Equal(t, mockWeather, server.weatherReporters)
	assert.Equal(t, mockVideos, server.videoStreamReporters)
}

// Tests for Listen
func TestListen(t *testing.T) {
	server := &Server{
		app: fiber.New(fiber.Config{DisableStartupMessage: true}),
	}

	address := make(chan string, 1)
	server.app.Hooks().OnListen(func(data fiber.ListenData) error {
		address <- net.JoinHostPort(data.Host, data.Port)
		return nil
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- server.Listen("127.0.0.1:0")
	}()

	var addr string
	select {
	case addr = <-address:
	case err := <-listenErr:
		t.Fatalf("Listen returned before serving: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server never started listening")
	}

	// Getting a response proves the server is serving before it is shut down.
	client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}, Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + "/")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	require.NoError(t, server.app.Shutdown())

	select {
	case err := <-listenErr:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Listen did not return after Shutdown")
	}
}

// Tests for listGeneralInfo
func TestListGeneralInfo_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Lisbon")

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Lisbon, Portugal").Return(createTestGeneralWeatherInfo(), nil)
	expectFreshData(mockVideos, "Lisbon")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "<html")
	assert.Contains(t, body, "Test Video 1")
	assert.Contains(t, body, "Test Video 1")
	waitForCache(t, "Lisbon")
}

func TestListGeneralInfo_WithCityParameter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Porto")

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Porto").Return(createTestGeneralWeatherInfoFor("Porto"), nil)
	expectFreshData(mockVideos, "Porto")

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Porto", nil))

	assert.Equal(t, fiber.StatusOK, status)
	waitForCache(t, "Porto")
}

func TestListGeneralInfo_HTMXRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Sintra")

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Sintra").Return(createTestGeneralWeatherInfoFor("Sintra"), nil)
	expectFreshData(mockVideos, "Sintra")

	// The search form posts to /process-form/ through HTMX.
	req := httptest.NewRequest("GET", "/process-form/?city_name=Sintra", nil)
	req.Header.Set("HX-Request", "true")
	status, body := doRequest(t, server, req)

	assert.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, body, "<html", "HTMX requests should only get the content fragment")
	assert.Contains(t, body, "Test Video 1")
	waitForCache(t, "Sintra")
}

func TestListGeneralInfo_WeatherReporterError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Lisbon, Portugal").Return(nil, errors.New("weather API error"))

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, fiber.StatusInternalServerError, status)
}

// Tests for checkDatabase
func TestCheckDatabase_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// No video expectations: a cache hit must not call those reporters.
	server, mockWeather, _ := createTestServer(ctrl)

	cached := map[string]any{
		"GeneralInfo": createTestGeneralWeatherInfoFor("Madeira"),
		"Videos":      application.VideosStream{{Title: "Cached Video", VideoID: "cached123"}},
		"Hotels":      createTestHotels(),
	}
	require.NoError(t, testStore.SaveCityData("Madeira", cached))

	data, err := server.checkDatabase("Madeira")
	require.NoError(t, err)
	require.Len(t, data.Videos, 1)
	assert.Equal(t, "Cached Video", data.Videos[0].Title)

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Madeira").Return(createTestGeneralWeatherInfoFor("Madeira"), nil)
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Madeira", nil))

	assert.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Cached Video")
}

func TestCheckDatabase_CacheMiss(t *testing.T) {
	server := &Server{storage: testStore}

	_, err := server.checkDatabase("Atlantis")
	assert.ErrorIs(t, err, sql.ErrNoRows)

	_, err = server.checkDatabase("")
	assert.Error(t, err)
}

// Tests for retireveFreshInformation
func TestRetireveFreshInformation_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Braga")

	generalInfo := createTestGeneralWeatherInfoFor("Braga")
	expectFreshData(mockVideos, "Braga")

	data, err := callRetrieveFreshInformation(server, generalInfo)

	require.NoError(t, err)
	assert.Equal(t, *generalInfo, data.GeneralInfo)
	assert.Equal(t, *createTestVideosStream(), data.Videos)
	waitForCache(t, "Braga")
}

func TestRetireveFreshInformation_VideosError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Aveiro")

	generalInfo := createTestGeneralWeatherInfoFor("Aveiro")
	mockVideos.EXPECT().GenerateReport(gomock.Any(), videosQuery("Aveiro")).Return(nil, errors.New("videos API error"))

	data, err := callRetrieveFreshInformation(server, generalInfo)

	// Missing videos fall back to an empty list without failing the page.
	require.NoError(t, err)
	assert.Empty(t, data.Videos)
	waitForCache(t, "Aveiro")
}

// Integration test for the complete flow: the first visit fetches and caches, the second is served from the cache.
func TestListGeneralInfo_CompleteFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Coimbra")

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Coimbra").Return(createTestGeneralWeatherInfoFor("Coimbra"), nil).Times(2)
	expectFreshData(mockVideos, "Coimbra")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Coimbra", nil))
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Test Video 1")
	waitForCache(t, "Coimbra")

	status, body = doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Coimbra", nil))
	assert.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Test Video 1")
}

// Test edge cases
func TestListGeneralInfo_EmptyCityName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Cascais")

	// The reporter answers with another city so this test's cache entry stays separate from
	// TestListGeneralInfo_Success.
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Lisbon, Portugal").Return(createTestGeneralWeatherInfoFor("Cascais"), nil)
	expectFreshData(mockVideos, "Cascais")

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=", nil))

	assert.Equal(t, fiber.StatusOK, status)
	waitForCache(t, "Cascais")
}

// Test for concurrent requests
func TestListGeneralInfo_ConcurrentRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Faro")

	// Fiber fills its render key list lazily on an app's first render, which races when the
	// first renders are concurrent (still true in Fiber v2.52.15). Rendering a cached page
	// first keeps this test about the app's own concurrency.
	require.NoError(t, testStore.SaveCityData("Tavira", map[string]any{"GeneralInfo": createTestGeneralWeatherInfoFor("Tavira")}))
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Tavira").Return(createTestGeneralWeatherInfoFor("Tavira"), nil)
	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Tavira", nil))
	require.Equal(t, fiber.StatusOK, status)

	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Faro").Return(createTestGeneralWeatherInfoFor("Faro"), nil).Times(3)

	// Requests that arrive after the first save are served from the cache.
	mockVideos.EXPECT().GenerateReport(gomock.Any(), videosQuery("Faro")).Return(createTestVideosStream(), nil).MinTimes(1).MaxTimes(3)

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp, err := server.app.Test(httptest.NewRequest("GET", "/?city_name=Faro", nil), 5000)
			if assert.NoError(t, err) {
				assert.Equal(t, fiber.StatusOK, resp.StatusCode)
				_ = resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	waitForCache(t, "Faro")
}

func TestServer_ItineraryRouteIsGone(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)
	status, _ := doRequest(t, server, httptest.NewRequest("POST", "/generate-itinerary", nil))
	assert.Equal(t, http.StatusNotFound, status)
}

func TestListGeneralInfo_PageHasNoHotelsCard(t *testing.T) {
	ctrl := gomock.NewController(t)
	server, weather, _ := createTestServer(ctrl)
	require.NoError(t, testStore.SaveCityData("CleanupGuard", map[string]any{"GeneralInfo": createTestGeneralWeatherInfoFor("CleanupGuard"), "Hotels": createTestHotels()}))
	weather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(createTestGeneralWeatherInfoFor("CleanupGuard"), nil).Times(2)
	for _, htmx := range []bool{false, true} {
		req := httptest.NewRequest("GET", "/", nil)
		if htmx {
			req.Header.Set("HX-Request", "true")
		}
		status, body := doRequest(t, server, req)
		require.Equal(t, http.StatusOK, status)
		assert.NotContains(t, body, "Test Hotel")
		assert.NotContains(t, body, "hotel-card")
	}
}

// Guard: old cache JSON remains readable after removing the obsolete field.
func TestCheckDatabase_LoadsOldRowsThatStillHaveHotels(t *testing.T) {
	require.NoError(t, testStore.SaveCityData("OldRow", map[string]any{"GeneralInfo": createTestGeneralWeatherInfoFor("OldRow"), "Videos": createTestVideosStream(), "Hotels": []map[string]string{{"HotelName": "Old hotel"}}}))
	data, err := (&Server{storage: testStore}).checkDatabase("OldRow")
	require.NoError(t, err)
	assert.Equal(t, "OldRow", data.GeneralInfo.City)
	require.Len(t, data.Videos, 2)
	assert.Equal(t, "Test Video 1", data.Videos[0].Title)
}
