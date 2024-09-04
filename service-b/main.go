package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var baseURL = "https://viacep.com.br/ws/"

type Cep struct {
	Cep string `json:"cep"`
}

type Output struct {
	City    string             `json:"city"`
	Weather map[string]float64 `json:"weather"`
}

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

func weatherHandlerGin(ctx *gin.Context) {
	startTime := time.Now()

	tracer := otel.Tracer("service-b")
	_, span := tracer.Start(ctx.Request.Context(), "processing_in_service_b")
	defer span.End()

	cep := ctx.Param("cep")
	if cep == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "CEP is required"})
		return
	}

	input := Cep{
		Cep: cep,
	}

	span.SetAttributes(attribute.String("cep", input.Cep))

	getLocationByCEPStartTime := time.Now()

	city, statusCode, err := getLocationByCEP(input.Cep)
	if err != nil && statusCode == http.StatusUnprocessableEntity {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	} else if statusCode == http.StatusNotFound {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	cepServiceDuration := time.Since(getLocationByCEPStartTime)
	span.SetAttributes(attribute.String("get_location_by_cep_duration", cepServiceDuration.String()))

	getWeatherByCityStartTime := time.Now()

	tempC, err := getWeatherByCity(city)
	if err != nil && statusCode == http.StatusUnprocessableEntity {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	} else if statusCode == http.StatusNotFound {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	tempF, tempK := convertTemperature(tempC)

	weather := map[string]float64{
		"temp_C": tempC,
		"temp_F": tempF,
		"temp_K": tempK,
	}

	output := Output{
		City:    city,
		Weather: weather,
	}

	getWeatherByCityDuration := time.Since(getWeatherByCityStartTime)
	span.SetAttributes(attribute.String("get_weather_by_city_duration", getWeatherByCityDuration.String()))

	endTime := time.Since(startTime)
	span.SetAttributes(attribute.String("total_duration_in_service_b", endTime.String()))

	ctx.JSON(http.StatusOK, output)
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	r := gin.Default()

	r.GET("/cep/:cep", weatherHandlerGin)
	r.Run(":8081")
}
