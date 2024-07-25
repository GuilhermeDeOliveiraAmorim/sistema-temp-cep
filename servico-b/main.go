package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func createHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr, Timeout: 10 * time.Second}
}

type CEPRequest struct {
	CEP string `json:"cep"`
}

type Location struct {
	Localidade string `json:"localidade"`
}

type Weather struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type TempResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shut down tracer: %v", err)
		}
	}()

	http.HandleFunc("/localizacao", handleLocation)
	log.Println("Serviço B está rodando na porta 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handleLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("servico-b")
	ctx, span := tracer.Start(ctx, "handleLocation")
	defer span.End()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CEPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(request.CEP) != 8 {
		http.Error(w, `{"mensagem": "invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	location, err := getLocation(ctx, request.CEP)
	if err != nil {
		http.Error(w, `{"mensagem": "can not find zipcode"}`, http.StatusNotFound)
		return
	}

	weather, err := getWeather(ctx, location.Localidade)
	if err != nil {
		log.Printf("Error contacting Weather API: %v", err)
		http.Error(w, `{"mensagem": "can not get weather data"}`, http.StatusInternalServerError)
		return
	}

	tempResponse := TempResponse{
		City:  location.Localidade,
		TempC: weather.Current.TempC,
		TempF: weather.Current.TempC*1.8 + 32,
		TempK: weather.Current.TempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tempResponse)
}

func getLocation(ctx context.Context, cep string) (Location, error) {
	client := createHTTPClient()
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep), nil)
	req = req.WithContext(ctx) // Passa o contexto para a requisição
	resp, err := client.Do(req)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	var location Location
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		return Location{}, err
	}

	return location, nil
}

func getWeather(ctx context.Context, city string) (Weather, error) {
	client := createHTTPClient()
	apiKey := "87022f0c0e0d4335a1d182957242207"
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", apiKey, city), nil)
	req = req.WithContext(ctx) // Passa o contexto para a requisição
	resp, err := client.Do(req)
	if err != nil {
		return Weather{}, err
	}
	defer resp.Body.Close()

	var weather Weather
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return Weather{}, err
	}

	return weather, nil
}

func initTracer() (*trace.TracerProvider, error) {
	endpoint := "http://localhost:9411/api/v2/spans"
	exporter, err := zipkin.New(endpoint)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("ServicoB"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}
