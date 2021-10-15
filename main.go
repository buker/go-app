package main

import (
	"net/http"
	"time"

	"github.com/buker/go-app/docs"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	log "github.com/sirupsen/logrus"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @contact.name TimeGladiator
// @contact.url http://www.timegladiator.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @termsOfService http://swagger.io/terms/

// @BasePath /api/v1

// PingExample godoc
// @Summary Show current date and time
// @Schemes
// @Description Show date and time
// @Tags example
// @Accept json
// @Produce json
// @Success 200 {string} Helloworld
// @Router /example/time [get]
func gettime(g *gin.Context) {
	datetime := time.Now()
	g.JSON(http.StatusOK, datetime)
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

func handleGetRecords(c *gin.Context) {
	var loadedRecords, err = GetAllRecords()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": loadedRecords})
}

func handleGetRecord(c *gin.Context) {
	var record Record
	if err := c.BindUri(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	var loadedRecord, err = GetRecordByID(record.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ID": loadedRecord.ID, "Body": loadedRecord.Body})
}

// @BasePath /api/v1

// CreateRecord godoc
// @Summary Create Record
// @Schemes
// @Description Save record in DB
// @Tags main
// @Accept json
// @Produce json
// @Success 200 {json} main
// @Router /main/record [put]
func handleCreateRecord(c *gin.Context) {
	var record Record
	if err := c.ShouldBindJSON(&record); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	id, err := Create(&record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func handleUpdateRecord(c *gin.Context) {
	var record Record
	if err := c.ShouldBindJSON(&record); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	savedRecord, err := Update(&record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": savedRecord})
}

func main() {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	//////////////////////////////////////////Sentry////////////////////////////////////////////
	app := gin.Default()
	store, _ := redis.NewStore(10, "tcp", "redis:6379", "", []byte("secret"))
	app.Use(sessions.Sessions("mysession", store))
	app.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	app.Use(func(ctx *gin.Context) {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.Scope().SetTag("someRandomTag", "maybeYouNeedIt")
		}
		ctx.Next()
	})
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

	/////////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////Swagger//////////////////////////////////////
	docs.SwaggerInfo.Title = "TimeGladiator API"
	docs.SwaggerInfo.Description = "API of TimeGladiator service"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/v2"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.BasePath = "/api/v1"
	v1 := app.Group("/api/v1")
	{
		eg := v1.Group("/main")
		{
			eg.GET("/helloworld", Helloworld)
		}
		{
			eg.GET("/time", gettime)
		}
		{
			eg.PUT("/record/", handleCreateRecord)
		}
	}

	/////////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////Metrics//////////////////////////////////////
	metricRouter := gin.Default()
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

	/////////////////////////////////////////////////////////////////////////////////////////////////
	///////////////////////Routes//////////////////////////////////////

	app.GET("/", func(ctx *gin.Context) {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetExtra("unwantedQuery", "someQueryDataMaybe")
				hub.CaptureMessage("User provided unwanted query string, but we recovered just fine")
			})
		}
		ctx.Status(http.StatusOK)
	})
	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	app.GET("/incr", func(c *gin.Context) {
		session := sessions.Default(c)
		var count int
		v := session.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count++
		}
		session.Set("count", count)
		session.Save()
		c.JSON(200, gin.H{"count": count})
	})
	log.Info("Starting server on port 8080")
	go func() {
		_ = metricRouter.Run(":8081")
	}()
	_ = app.Run()
}
