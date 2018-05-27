package main

import (
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/helloworldpark/goticklecollector/api"

	"github.com/helloworldpark/goticklecollector/collector"
	"github.com/helloworldpark/goticklecollector/holder"
)

func main() {
	// Parse flags
	credPath := flag.String("credential", "", "Credential for DB access")
	flag.Parse()

	if credPath == nil || *credPath == "" {
		panic("No -credential provided")
	}

	// Read credential
	credential := holder.LoadCredential(*credPath)

	// DB Holder
	dbHolder := holder.NewDBHolder(credential, 100)
	dbHolder.Init()

	// Setup Coin Collectors
	currencies := []string{"eos"}
	collectors := collector.NewCollectors(api.Coinone, currencies)
	holders := make([]holder.Holder, 0)

	dbChannels := make([]<-chan collector.Coin, 0)

	for _, col := range collectors {
		h := holder.New(api.Coinone.Name, col.Currency(), 10)
		worker := collector.GiveWork(col, 20*time.Second)
		dbChannel := h.UpdatingPipe(worker)

		holders = append(holders, h)
		dbChannels = append(dbChannels, dbChannel)
	}

	// Feed to DB
	dbHolder.ConnectDBChannel(dbChannels)

	// Setup API
	router := gin.Default()
	router.GET("/coins/last", func(c *gin.Context) {
		// v := c.Query("vendor")
		cur := c.Query("currency")
		lastSeconds, _ := strconv.ParseInt(c.Query("seconds"), 10, 64)
		coins := make([]collector.Coin, 0)
		for i := 0; i < len(holders); i++ {
			if (&holders[i]).Currency == cur {
				coins = (&holders[i]).ProvideLast(lastSeconds)
			}
		}

		if len(coins) > 0 {
			c.JSON(http.StatusOK, coins)
		} else {
			c.String(http.StatusBadRequest, "No coin data: currency=%s lastseconds=%d", cur, lastSeconds)
		}
	})
	router.Run(":50001")
}
