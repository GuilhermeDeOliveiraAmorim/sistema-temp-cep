package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	http.HandleFunc("/cep", handleCEP)
	log.Println("Serviço A está rodando na porta 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleCEP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("servico-a")
	ctx, span := tracer.Start(ctx, "handleCEP")
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

	if len(request.CEP) != 8 || !isNumeric(request.CEP) {
		http.Error(w, `{"mensagem": "invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	fmt.Println(request.CEP)

	client := createHTTPClient()
	req, err := http.NewRequest("POST", "http://localhost:8081/localizacao", bytes.NewBuffer([]byte(fmt.Sprintf(`{"cep": "%s"}`, request.CEP))))
	if err != nil {
		log.Printf("Error contacting CEP API: %v", err)
		http.Error(w, `{"mensagem": "can not get cep data"}`, http.StatusInternalServerError)
		return
	}

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error contacting Weather API: %v", err)
		http.Error(w, `{"mensagem": "can not get weather data"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Weather API returned status %d: %s", resp.StatusCode, string(body))
		http.Error(w, `{"mensagem": "can not get weather data"}`, http.StatusInternalServerError)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response from Serviço B", http.StatusInternalServerError)
		return
	}

	var tempResponse TempResponse
	if err := json.Unmarshal(body, &tempResponse); err != nil {
		http.Error(w, "Error parsing response from Serviço B", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tempResponse)
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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
			semconv.ServiceNameKey.String("ServicoA"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}
