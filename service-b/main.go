package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var baseURL = "https://viacep.com.br/ws/"

func createHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr, Timeout: 10 * time.Second}
}

func isValidCEP(cep string) bool {
	cepPattern := `^\d{5}-\d{3}$|^\d{8}$`
	re := regexp.MustCompile(cepPattern)
	return re.MatchString(cep)
}

func getLocationByCEP(cep string) (string, int, error) {
	if !isValidCEP(cep) {
		return "", http.StatusUnprocessableEntity, fmt.Errorf("invalid zipcode")
	}

	client := createHTTPClient()
	resp, err := client.Get(baseURL + cep + "/json/")
	if err != nil {
		return "", resp.StatusCode, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("can not find zipcode")
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", http.StatusUnprocessableEntity, err
	}

	if data["erro"] == "true" {
		return "", http.StatusNotFound, fmt.Errorf("can not find zipcode")
	}

	city, ok := data["localidade"]
	if !ok {
		return "", http.StatusNotFound, fmt.Errorf("can not find zipcode")
	}

	return city, 200, nil
}

func getWeatherByCity(city string) (float64, error) {
	client := createHTTPClient()
	apiKey := "87022f0c0e0d4335a1d182957242207"
	encodedCity := url.QueryEscape(city)
	url := "https://api.weatherapi.com/v1/current.json?key=" + apiKey + "&q=" + encodedCity
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("can not find zipcode")
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	current, ok := data["current"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("temperature data not found")
	}

	tempC, ok := current["temp_c"].(float64)
	if !ok {
		return 0, fmt.Errorf("temperature data not found")
	}

	return tempC, nil
}

func convertTemperature(tempC float64) (float64, float64) {
	tempF := tempC*1.8 + 32

	tempK := tempC + 273.15
	tempKStr := fmt.Sprintf("%.2f", tempK)
	tempKFloat, err := strconv.ParseFloat(tempKStr, 64)
	if err != nil {
		fmt.Println("Error converting temperature to float64:", err)
	}

	return tempF, tempKFloat
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(r.Context(), "weatherHandler")
	defer span.End()

	cep := strings.TrimPrefix(r.URL.Path, "/weather/")

	getLocationByCEPStartTime := time.Now()
	_, childSpan := tracer.Start(ctx, "get_location_by_cep")
	defer childSpan.End()

	city, statusCode, err := getLocationByCEP(cep)
	if err != nil && statusCode == http.StatusUnprocessableEntity {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	} else if statusCode == http.StatusNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cepServiceDuration := time.Since(getLocationByCEPStartTime)
	childSpan.SetAttributes(attribute.Float64("get_location_by_cep_duration_ms", float64(cepServiceDuration.Milliseconds())))

	getWeatherByCityStartTime := time.Now()
	_, childSpan = tracer.Start(ctx, "get_weather_by_city")
	defer childSpan.End()

	tempC, err := getWeatherByCity(city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	tempF, tempK := convertTemperature(tempC)

	response := map[string]float64{
		"temp_C": tempC,
		"temp_F": tempF,
		"temp_K": tempK,
	}

	getWeatherByCityDuration := time.Since(getWeatherByCityStartTime)
	childSpan.SetAttributes(attribute.Float64("get_weather_by_city_duration_ms", float64(getWeatherByCityDuration.Milliseconds())))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	http.HandleFunc("/cep/", weatherHandler)
	http.ListenAndServe(":8081", nil)
}
