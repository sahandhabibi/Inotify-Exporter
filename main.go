package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	inotify_max_user_watches_var = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inotify_max_user_watches",
		Help: "Maximum Number of inotify watches",
	})

	inotify_user_watches_running = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inotify_user_watches_running",
		Help: "The Number of Running inotify watches",
	})
	inotify_max_user_instances_var = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inotify_max_user_instances",
		Help: "Maximum Number of inotify instances",
	})
	inotify_user_instances_running = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "inotify_user_instances_running",
		Help: "The Number of running inotify instances",
	})
)

func init() {
	prometheus.MustRegister(inotify_max_user_watches_var)
	prometheus.MustRegister(inotify_user_watches_running)
	prometheus.MustRegister(inotify_max_user_instances_var)
	prometheus.MustRegister(inotify_user_instances_running)
}

func inotify_max_user_watches(metricobject prometheus.Gauge) {
	out, _ := exec.Command("cat", "/proc/sys/fs/inotify/max_user_watches").Output()
	output := string(out[:])
	MetricValue, _ := strconv.ParseFloat(strings.TrimSpace(output), 64)
	metricobject.Set(MetricValue)
}

func inotify_max_user_instances(metricobject prometheus.Gauge) {
	out, _ := exec.Command("cat", "/proc/sys/fs/inotify/max_user_instances").Output()
	output := string(out[:])
	MetricValue, _ := strconv.ParseFloat(strings.TrimSpace(output), 64)
	metricobject.Set(MetricValue)
}

func inotify_user_watches(metricobject prometheus.Gauge) {
	var sum int
	c := string(runcmd("find /proc/*/fd -lname anon_inode:inotify -printf '%hinfo/%f\n' 2>/dev/null | xargs grep -c '^inotify' | sort -n -t: -k2 -r | sed -e 's/.*://'", true))
	scanner := bufio.NewScanner(strings.NewReader(c))
	for scanner.Scan() {
		pid, _ := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		sum += pid
	}
	metricobject.Set(float64(sum))
}

func inotify_user_instances(metricobject prometheus.Gauge) {
	output := string(runcmd("ls -l /proc/*/fd/* 2>/dev/null |grep inotify | wc -l ", true))
	MetricValue, _ := strconv.ParseFloat(strings.TrimSpace(output), 64)
	metricobject.Set(MetricValue)
}

func recordMetrics() {
	go func() {
		for {
			inotify_max_user_watches(inotify_max_user_watches_var)
			inotify_user_watches(inotify_user_watches_running)
			inotify_max_user_instances(inotify_max_user_instances_var)
			inotify_user_instances(inotify_user_instances_running)

			time.Sleep(20 * time.Second)
		}
	}()
}

func runcmd(cmd string, shell bool) []byte {
	if shell {
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			log.Fatal(err)
			panic("some error found")
		}
		return out
	}
	out, err := exec.Command(cmd).Output()
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func main() {
	Port := flag.Int("Port", 2110, "Port Number to listen")
	flag.Parse()
	recordMetrics()
	var port = ":" + strconv.Itoa(*Port)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(port, nil))
}
