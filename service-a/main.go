package main

import (
	"encoding/json"
	"net/http"
	"regexp"

	"go.opentelemetry.io/otel"
)

func validateCep(cep string) bool {
	match, _ := regexp.MatchString(`^\d{8}$`, cep)
	return match
}

func cepHandler(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(r.Context(), "cepHandler")
	defer span.End()

	var request struct {
		Cep string `json:"cep"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if !validateCep(request.Cep) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	_, childSpan := tracer.Start(ctx, "forwardToServiceB")
	defer childSpan.End()

	http.Redirect(w, r, "http://service-b/cep/"+request.Cep, http.StatusSeeOther)
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	http.HandleFunc("/cep", cepHandler)
	http.ListenAndServe(":8080", nil)
}
