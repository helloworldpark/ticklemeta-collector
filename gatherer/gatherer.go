package gatherer

import (
	"fmt"
	"sync"
	"time"

	"github.com/helloworldpark/goticklecollector/collector"
)

// CoinGateway is a wrapper of collector.Coin channel
// It provides only a property to <-chan collector.Coin
type CoinGateway struct {
	gateway chan collector.Coin
}

// Channel is a property to <-chan collector.Coin
func (cg CoinGateway) Channel() <-chan collector.Coin {
	return cg.gateway
}

// Gather collects pipes and export to two pipelines:
// Dataframe Pipeline: defined per vendor, per currency
//                     use it to update data frame
// DB Pipeline: defined for all types of coin
func Gather(pipes []CoinGateway) ([]CoinGateway, CoinGateway) {
	dfBundle := make([]CoinGateway, len(pipes))
	for i := 0; i < len(pipes); i++ {
		bundle := CoinGateway{}
		bundle.gateway = make(chan collector.Coin)
		dfBundle[i] = bundle
	}
	dbBundle := CoinGateway{}
	dbBundle.gateway = make(chan collector.Coin)

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(pipes))

	for idx, pipe := range pipes {
		go func(idx int, in CoinGateway) {
			defer waitGroup.Done()

			for coin := range in.gateway {
				dbBundle.gateway <- coin
				dfBundle[idx].gateway <- coin
			}
		}(idx, pipe)
	}
	go func() {
		waitGroup.Wait()

		fmt.Println("Finished!")
		close(dbBundle.gateway)
		for _, bundle := range dfBundle {
			close(bundle.gateway)
		}
	}()
	return dfBundle, dbBundle
}

// GiveWork gives work to collector.Collector. It runs a goroutine and returns channels per coin type
// Params:
//    c collector.Collector: Type of coin to collect.
//    period time.Duration: Time to sleep in seconds.
// Returns:
//    []CoinGateway: CoinGateway per coin type
func GiveWork(c collector.Collector, period time.Duration) []CoinGateway {
	gateways := make([]CoinGateway, c.Count())
	for i := 0; i < len(gateways); i++ {
		gateway := CoinGateway{}
		gateway.gateway = make(chan collector.Coin)
		gateways[i] = gateway
	}

	closeAllGateways := func() {
		for _, g := range gateways {
			close(g.gateway)
		}
	}

	go func() {
		defer closeAllGateways()
		for true {
			coins := c.Collect()
			fmt.Println(fmt.Sprintf("Collected(#1): %d", len(coins)))
			for idx, coin := range coins {
				gateways[idx].gateway <- coin
			}
			time.Sleep(period)
		}
	}()

	return gateways
}

// MergeGateways merges slices of CoinGateway to one slice of CoinGateway
func MergeGateways(gateways ...[]CoinGateway) []CoinGateway {
	total := 0
	for _, gateway := range gateways {
		total += len(gateway)
	}
	merged := make([]CoinGateway, 0, total)
	for _, gateway := range gateways {
		for _, g := range gateway {
			merged = append(merged, g)
		}
	}
	return merged
}
