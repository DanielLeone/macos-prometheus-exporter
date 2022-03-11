package main

import (
	"bufio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func wifiGauge(name string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: "",
	}, []string{"bssid", "ssid"})
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})

	agrCtlRSSIRegex = regexp.MustCompile(`\s+agrCtlRSSI: (.*)\s+`)
	agrCtlRSSI      = wifiGauge("wifi_agrCtlRSSI")

	agrExtRSSIRegex = regexp.MustCompile(`\s+agrExtRSSI: (.*)\s+`)
	agrExtRSSI      = wifiGauge("wifi_agrExtRSSI")

	agrCtlNoiseRegex = regexp.MustCompile(`\s+agrCtlNoise: (.*)\s+`)
	agrCtlNoise      = wifiGauge("wifi_agrCtlNoise")

	agrExtNoiseRegex = regexp.MustCompile(`\s+agrExtNoise: (.*)\s+`)
	agrExtNoise      = wifiGauge("wifi_agrExtNoise")

	lastTxRateRegex = regexp.MustCompile(`\s+lastTxRate: (.*)\s+`)
	lastTxRate      = wifiGauge("wifi_lastTxRate")

	maxRateRegex = regexp.MustCompile(`\s+maxRate: (.*)\s+`)
	maxRate      = wifiGauge("wifi_maxRate")

	mcsRegex = regexp.MustCompile(`\s+MCS: (.*)\s+`)
	mcs      = wifiGauge("wifi_mcs")

	channelRegex = regexp.MustCompile(`\s+channel: (.*)\s+`)
	channel      = wifiGauge("wifi_channel")

	powermetricsCpuThermalLevel = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_cpu_thermal_level",
		Help: "",
	})

	powermetricsGpuThermalLevel = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_gpu_thermal_level",
		Help: "",
	})

	powermetricsIoThermalLevel = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_io_thermal_level",
		Help: "",
	})

	powermetricsFanRpm = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_fan_rpm",
		Help: "",
	})

	powermetricsCpuDieTemperature = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_cpu_die_temperature",
		Help: "",
	})

	powermetricsGpuDieTemperature = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powermetrics_gpu_die_temperature",
		Help: "",
	})
)

// $ /System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport -I
//      agrCtlRSSI: -62
//      agrExtRSSI: 0
//     agrCtlNoise: -96
//     agrExtNoise: 0
//           state: running
//         op mode: station
//      lastTxRate: 270
//         maxRate: 600
// lastAssocStatus: 0
//     802.11 auth: open
//       link auth: wpa2-psk
//           BSSID: 44:d9:e7:f8:b3:0
//            SSID: Goosenet
//             MCS: 4
//         channel: 149,1

// $ sudo powermetrics --samplers smc -i1 -n1
// Machine model: MacBookPro15,1
// SMC version: Unknown
// EFI version: 1554.20.0
// OS version: 20G95
// Boot arguments: chunklist-security-epoch=0 -chunklist-no-rev2-dev
// Boot time: Sun Feb 27 07:49:33 2022
//
//
//
// *** Sampled system activity (Thu Mar  3 13:41:31 2022 +0800) (8.32ms elapsed) ***
//
//
//
// **** SMC sensors ****
//
// CPU Thermal level: 77
// GPU Thermal level: 27
// IO Thermal level: 27
// Fan: 3906.89 rpm
// CPU die temperature: 74.80 C
// GPU die temperature: 70.00 C
// CPU Plimit: 0.00
// GPU Plimit (Int): 0.00
// GPU2 Plimit (Ext1): 0.00
// Number of prochots: 0

func recordMetrics() {
	for {
		opsProcessed.Inc()

		out, err := exec.Command("powermetrics", "--samplers", "smc", "-i1", "-n1").Output()
		if err != nil {
			log.Fatal(string(err.(*exec.ExitError).Stderr))
		}

		powermetricsCpuThermalLevelRegex := regexp.MustCompile(`^CPU Thermal level: (\d+)$`)
		powermetricsCpuThermalLevelLineCheck := "CPU Thermal level:"

		powermetricsGpuThermalLevelRegex := regexp.MustCompile(`^GPU Thermal level: (\d+)$`)
		powermetricsGpuThermalLevelLineCheck := "GPU Thermal level:"

		powermetricsIoThermalLevelRegex := regexp.MustCompile(`^IO Thermal level: (\d+)$`)
		powermetricsIoThermalLevelLineCheck := "IO Thermal level:"

		powermetricsFanRpmRegex := regexp.MustCompile(`^Fan: (\d+.\d+) rpm$`)
		powermetricsFanRpmLineCheck := "Fan:"

		powermetricsCpuDieTemperatureRegex := regexp.MustCompile(`^CPU die temperature: (\d+.\d+) C.*`)
		powermetricsCpuDieTemperatureLineCheck := "CPU die temperature:"

		powermetricsGpuDieTemperatureRegex := regexp.MustCompile(`^GPU die temperature: (\d+.\d+) C.*`)
		powermetricsGpuDieTemperatureLineCheck := "GPU die temperature:"

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			text := scanner.Text()
			testExtractParseAssign(text, powermetricsCpuThermalLevelLineCheck, powermetricsCpuThermalLevelRegex, powermetricsCpuThermalLevel)
			testExtractParseAssign(text, powermetricsGpuThermalLevelLineCheck, powermetricsGpuThermalLevelRegex, powermetricsGpuThermalLevel)
			testExtractParseAssign(text, powermetricsIoThermalLevelLineCheck, powermetricsIoThermalLevelRegex, powermetricsIoThermalLevel)
			testExtractParseAssign(text, powermetricsFanRpmLineCheck, powermetricsFanRpmRegex, powermetricsFanRpm)
			testExtractParseAssign(text, powermetricsCpuDieTemperatureLineCheck, powermetricsCpuDieTemperatureRegex, powermetricsCpuDieTemperature)
			testExtractParseAssign(text, powermetricsGpuDieTemperatureLineCheck, powermetricsGpuDieTemperatureRegex, powermetricsGpuDieTemperature)
		}

		out, err = exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I").Output()
		if err != nil {
			log.Fatal(string(err.(*exec.ExitError).Stderr))
		}

		//      agrCtlRSSI: -57
		//     agrExtRSSI: 0
		//    agrCtlNoise: -94
		//    agrExtNoise: 0
		//          state: running
		//        op mode: station
		//     lastTxRate: 526
		//        maxRate: 144
		//lastAssocStatus: 0
		//    802.11 auth: open
		//      link auth: wpa2-psk
		//          BSSID: ee:55:b8:5:62:e5
		//           SSID: FTPSolutions_Development
		//            MCS: 6
		//        channel: 36,80

		bssidRegex := regexp.MustCompile(`\s+BSSID: (\w+:\w+:\w+:\w+:\w+:\w+)\s+`)
		ssidRegex := regexp.MustCompile(`\s+SSID: (.*)\s+`)

		bssid := extractRegexMatch(out, bssidRegex)
		ssid := extractRegexMatch(out, ssidRegex)

		testExtractParseAssign(string(out), "agrCtlRSSI", agrCtlRSSIRegex, agrCtlRSSI.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "agrCtlRSSI", agrCtlRSSIRegex, agrCtlRSSI.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "agrExtRSSI", agrExtRSSIRegex, agrExtRSSI.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "agrCtlNoise", agrCtlNoiseRegex, agrCtlNoise.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "agrExtNoise", agrExtNoiseRegex, agrExtNoise.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "lastTxRate", lastTxRateRegex, lastTxRate.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "maxRate", maxRateRegex, maxRate.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "MCS", mcsRegex, mcs.With(map[string]string{"bssid": bssid, "ssid": ssid}))
		testExtractParseAssign(string(out), "channel", channelRegex, channel.With(map[string]string{"bssid": bssid, "ssid": ssid}))

		time.Sleep(2 * time.Second)
	}
}

func extractRegexMatch(out []byte, exp *regexp.Regexp) string {
	matches := exp.FindStringSubmatch(string(out))
	//log.Println(text, lineCheckString, matches)
	if len(matches) != 2 {
		log.Fatal("who the fock", matches)
	}
	value := matches[1]
	return value
}

func testExtractParseAssign(text string, lineCheckString string, extractRegex *regexp.Regexp, gauge prometheus.Gauge) {
	if strings.Contains(text, lineCheckString) {
		matches := extractRegex.FindStringSubmatch(text)
		//log.Println(text, lineCheckString, matches)
		if len(matches) != 2 {
			log.Fatal("who the fock", matches)
		}
		if s, err := strconv.ParseFloat(matches[1], 64); err == nil {
			gauge.Set(s)
		}
	}
}

func main() {
	log.Println("Starting Exporter")
	go recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":9003", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Finishing Exporter")
}
