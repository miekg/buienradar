package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.science.ru.nl/log"
	"go.science.ru.nl/promfmt"
)

var (
	flagWrite = flag.Bool("w", true, "write to /var/lib/prometheus/node-exporter/f2bne.prom")
)

const promfile = "/var/lib/prometheus/node-exporter/buienradar.prom"

type Buienradar struct {
	Name     string
	Humidity float64
	Pressure float64
	Rain     float64
	Temp     float64
}

var (
	buienradarHumidity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "buienradar_humidity_percentage",
		Help: "The current humidity percentage",
	}, []string{"name"})
	buienradarPressure = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "buienradar_pressure_hpa",
		Help: "The current air pressure in hPa",
	}, []string{"name"})
	buienradarRain = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "buienradar_rain_mmph",
		Help: "The current rain in mm per hour",
	}, []string{"name"})
	buienradarTemp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "buienradar_temp_celcius",
		Help: "The current temperatuur in celcius",
	}, []string{"name"})
	buienradarTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "buienradar_last_run_time_seconds",
		Help: "Epoch timestamp of the last run.",
	})
)

func main() {
	flag.Parse()
	doc, err := xmlquery.LoadURL("http://data.buienradar.nl/1.0/feed/xml")
	if err != nil {
		log.Warningf("Error fetching XML %s:", err)
	}

	brs := []Buienradar{}
	for _, station := range xmlquery.Find(doc, "//weerstations/weerstation") {
		humidity, _ := strconv.ParseFloat(xmlquery.Find(station, "luchtvochtigheid/text()")[0].InnerText(), 64)
		pressure, _ := strconv.ParseFloat(xmlquery.Find(station, "luchtdruk/text()")[0].InnerText(), 64)
		rain, _ := strconv.ParseFloat(xmlquery.Find(station, "regenMMPU/text()")[0].InnerText(), 64)
		temp, _ := strconv.ParseFloat(xmlquery.Find(station, "temperatuurGC/text()")[0].InnerText(), 64)

		br := Buienradar{
			Name:     strings.ToLower(strings.Replace(strings.Replace(xmlquery.Find(station, "stationnaam/text()")[0].InnerText(), "Meetstation ", "", 1), " ", "-", -1)),
			Humidity: float64(humidity),
			Pressure: float64(pressure),
			Rain:     float64(rain),
			Temp:     float64(temp),
		}
		brs = append(brs, br)
	}

	for _, br := range brs {
		buienradarHumidity.WithLabelValues(br.Name).Set(br.Humidity)
		buienradarPressure.WithLabelValues(br.Name).Set(br.Pressure)
		buienradarRain.WithLabelValues(br.Name).Set(br.Rain)
		buienradarTemp.WithLabelValues(br.Name).Set(br.Temp)
	}
	buienradarTimestamp.Set(float64(time.Now().Unix()))

	if !*flagWrite {
		promfmt.Fprint(os.Stdout, promfmt.NewPrefixFilter("buienradar_"))
		return
	}
	if err := promfmt.WriteFile(promfile, promfmt.NewPrefixFilter("buienradar_")); err != nil {
		log.Fatalf("Failed to write to prom file: %s", err)
	}
}
