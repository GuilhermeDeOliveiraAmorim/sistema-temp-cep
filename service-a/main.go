package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func validateCep(cep string) bool {
	match, _ := regexp.MatchString(`^\d{8}$`, cep)
	return match
}

func cepHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(r.Context(), "cepHandler")
	defer span.End()

	startTime := time.Now()

	var request struct {
		Cep string `json:"cep"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		span.SetAttributes(attribute.String("error", "Invalid input"))
		return
	}

	fmt.Println(request.Cep)
	fmt.Println(!validateCep(request.Cep))

	span.SetAttributes(attribute.String("cep", request.Cep))

	if !validateCep(request.Cep) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		span.SetAttributes(attribute.String("error", "invalid zipcode"))
		return
	}

	cepServiceStartTime := time.Now()
	fmt.Println(cepServiceStartTime)
	_, childSpan := tracer.Start(ctx, "forwardToServiceB")
	defer childSpan.End()

	childSpan.SetAttributes(attribute.String("cep", request.Cep))
	childSpan.SetAttributes(attribute.String("redirect_url", "http://sistema-b/cep/"+request.Cep))

	http.Redirect(w, r, "http://sistema-b/cep/"+request.Cep, http.StatusSeeOther)

	cepServiceDuration := time.Since(cepServiceStartTime)
	childSpan.SetAttributes(attribute.Float64("cep_service_duration_ms", float64(cepServiceDuration.Milliseconds())))

	totalDuration := time.Since(startTime)
	span.SetAttributes(attribute.Float64("total_execution_duration_ms", float64(totalDuration.Milliseconds())))
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	http.HandleFunc("/cep", cepHandler)
	http.ListenAndServe(":8080", nil)
}
