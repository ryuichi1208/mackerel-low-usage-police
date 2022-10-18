package lib

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var logger *zap.Logger
var opts options

type options struct {
	Origization string `short:"o" long:"org" description:"" required:"false"`
	Service     string `short:"s" long:"service" description:"" required:"true"`
	Roles       string `short:"r" long:"roles" description:"" required:"true"`
	Filter      string `short:"f" long:"filter" description:"" required:"false"`
	TimeWindow  int    `long:"timewindow" description:"Specify time window (unit: day)" default:"1" required:"false"`
	Prefix      string `long:"prefix" description:"" required:"false"`
	Metrics     string `long:"metrics" description:"" default:"cpu.user.percentage" required:"false"`
	Version     bool   `long:"version" description:"print version and exit" required:"false"`
	Debug       bool   `long:"debug" description:"Enable debug mode" required:"false"`
	Verbose     bool   `long:"verbose" description:"Enable debug mode" required:"false"`
}

type Mackerel struct {
	client  mackerel.Client
	org     string
	service string
	roles   string
	filter  string
}

func NewMackerel(token, org, service, roles, filter, metrics string) Mackerel {
	return Mackerel{
		client:  *mackerel.NewClient(token),
		org:     org,
		service: service,
		roles:   roles,
		filter:  filter,
	}
}

func (m Mackerel) GetMetrics() map[string]float64 {
	a, b := m.client.FetchHostMetricValues("2x6jjwhDdjU", "cpu.user.percentage", 1666053720, 1666089720)
	if b != nil {
		return nil
	}

	for _, v := range a {
		fmt.Println(time.Unix(v.Time, 0), v.Value)
	}

	return nil
}

func run() error {
	m := NewMackerel(getMackerelToke(), opts.Origization, opts.Service, opts.Roles, opts.Filter, opts.Metrics)

	service_exists, err := m.CheckService()
	if !service_exists || err != nil {
		logger.Error(fmt.Sprintf("Specified service %s does not exist", opts.Service))
		return err
	}

	var targetMetricsList []string
	switch opts.Metrics {
	case "cpu":
		targetMetricsList = []string{"cpu.user.percentage", "cpu.iowait.percentage", "cpu.system.percentage", "cpu.nice.percentage"}
	case "memory":
		return fmt.Errorf("Not Suppot Metrics")
	}

	hosts, err := m.FetchHosts()

	fmt.Printf("host\t\t\tmax\tmin\tavg\tp50\tp90\n")
	eg := new(errgroup.Group)
	for _, host := range hosts {
		host := host
		eg.Go(func() error {
			name, err := m.GetHostName(host)
			if err != nil {
				return err
			}
			name = strings.Split(name, ".")[0]

			metricsMap := make(map[string]map[time.Time]float64)
			result := make(map[time.Time]float64)
			var values []float64
			var maxTime, minTime time.Time
			var maxValue float64 = 0
			var minValue float64 = math.MaxFloat64

			for _, targetMetrics := range targetMetricsList {
				logger.Debug(fmt.Sprintf("targetMetrics: %s", targetMetrics))
				metricsMap[targetMetrics], err = m.FetchMetrics(host, targetMetrics)
				if err != nil {
					return err
				}
				for time, value := range metricsMap[targetMetrics] {
					if opts.Verbose {
						logger.Debug(fmt.Sprintf("%s: %f", time, value))
					}
					result[time] += value
				}

			}

			var total float64
			for time, sumValue := range result {
				if opts.Verbose {
					logger.Debug(fmt.Sprintf("%s: %f", time, sumValue))
				}

				if maxValue < sumValue {
					maxTime = time
					maxValue = sumValue
				}

				if minValue > sumValue {
					minTime = time
					minValue = sumValue
				}

				total += sumValue
				values = append(values, sumValue)

			}
			if opts.Verbose {
				fmt.Println(host, maxTime, maxValue)
				fmt.Println(host, minTime, minValue)
			}

			avg := total / float64(len(result))
			p50, err := PercentileN(values, 50)
			if err != nil {
				return err
			}
			p90, err := PercentileN(values, 90)
			if err != nil {
				return err
			}
			if avg < 100 {
				fmt.Printf("\x1b[31m")
				fmt.Printf("%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n", name, maxValue, minValue, avg, p50, p90)
				fmt.Printf("\x1b[0m")

			} else {
				fmt.Printf("%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n", name, maxValue, minValue, avg, p50, p90)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func Do() int {
	err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	initLogger()

	err = run()
	if err != nil {
		return 1
	}
	return 0
}
