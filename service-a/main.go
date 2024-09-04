package main

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Cep struct {
	Cep string `json:"cep"`
}

type Output struct {
	City    string             `json:"city"`
	Weather map[string]float64 `json:"weather"`
}

func validateCep(cep string) bool {
	match, _ := regexp.MatchString(`^\d{8}$`, cep)
	return match
}

func cepHandlerGin(ctx *gin.Context) {
	startTime := time.Now()

	tracer := otel.Tracer("service-a")

	_, span := tracer.Start(ctx.Request.Context(), "request_to_service_b")
	defer span.End()

	var input Cep
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	span.SetAttributes(attribute.String("cep", input.Cep))

	if !validateCep(input.Cep) {
		span.SetAttributes(attribute.String("error", "invalid zipcode"))
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid zipcode"})
		return
	}

	sendRequestToServiceBStartTime := time.Now()

	span.SetAttributes(attribute.String("redirect_url", "http://localhost:8081/cep/"+input.Cep))

	ctx.Redirect(http.StatusSeeOther, "http://localhost:8081/cep/"+input.Cep)

	cepServiceDuration := time.Since(sendRequestToServiceBStartTime)

	span.SetAttributes(attribute.String("cep_service_duration", cepServiceDuration.String()))

	totalDuration := time.Since(startTime)

	span.SetAttributes(attribute.String("total_execution_duration", totalDuration.String()))
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Welcome to Service A!")
	})
	r.POST("/cep/", cepHandlerGin)
	r.Run(":8080")
}
