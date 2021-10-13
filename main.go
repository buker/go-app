package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/buker/go-app/docs"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	log "github.com/sirupsen/logrus"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @termsOfService http://swagger.io/terms/

func gettime(c *gin.Context) {
	datetime := fmt.Sprintln(time.Now())
	c.Writer.Write([]byte(datetime))
	log.Info("Time requested")
}

// @BasePath /api/v1

// PingExample godoc
// @Summary ping example
// @Schemes
// @Description do ping
// @Tags example
// @Accept json
// @Produce json
// @Success 200 {string} Helloworld
// @Router /example/helloworld [get]
func Helloworld(g *gin.Context) {
	g.JSON(http.StatusOK, "helloworld")
}
func main() {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://a67153b6c1214429846bd148ec2e5be5@o380765.ingest.sentry.io/6004421",
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if hint.Context != nil {
				if req, ok := hint.Context.Value(sentry.RequestContextKey).(*http.Request); ok {
					// You have access to the original Request here
					log.Info("Sentry request: %v", req)
				}
			}

			return event
		},
	}); err != nil {
		log.Fatalf("Sentry initialization failed: %v\n", err)
	}
	docs.SwaggerInfo.Title = "Swagger Example API"
	docs.SwaggerInfo.Description = "This is a sample server Petstore server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "petstore.swagger.io"
	docs.SwaggerInfo.BasePath = "/v2"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	app := gin.Default()
	metricRouter := gin.Default()
	docs.SwaggerInfo.BasePath = "/api/v1"
	v1 := app.Group("/api/v1")
	{
		eg := v1.Group("/example")
		{
			eg.GET("/helloworld", Helloworld)
		}
	}
	// get global Monitor object
	metrics := ginmetrics.GetMonitor()
	// +optional set metric path, default /debug/metrics
	metrics.SetMetricPath("/metrics")
	// +optional set slow time, default 5s
	metrics.SetSlowTime(10)
	// +optional set request duration, default {0.1, 0.3, 1.2, 5, 10}
	// used to p95, p99
	metrics.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})
	// set middleware for gin
	metrics.UseWithoutExposingEndpoint(app)
	metrics.Expose(metricRouter)
	app.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	app.Use(func(ctx *gin.Context) {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.Scope().SetTag("someRandomTag", "maybeYouNeedIt")
		}
		ctx.Next()
	})

	app.GET("/", func(ctx *gin.Context) {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("unwantedQuery", "someQueryDataMaybe")
				hub.CaptureMessage("User provided unwanted query string, but we recovered just fine")
			})
		}
		ctx.Status(http.StatusOK)
	})
	app.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	app.GET("/foo", func(ctx *gin.Context) {
		// sentrygin handler will catch it just fine. Also, because we attached "someRandomTag"
		// in the middleware before, it will be sent through as well
		panic("y tho")
	})

	app.GET("/product/:id", func(ctx *gin.Context) {
		ctx.JSON(200, map[string]string{
			"productId": ctx.Param("id"),
		})
	})
	app.GET("/time", gettime)
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	log.Info("Starting server on port 8080")
	go func() {
		_ = metricRouter.Run(":8081")
	}()
	_ = app.Run()
}
