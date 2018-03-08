package main

import (
	"flag"
	"github.com/google/gops/agent"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tokopedia/gosample/hello"
	"github.com/tokopedia/gosample/user"
	"github.com/tokopedia/logging/tracer"
	"gopkg.in/tokopedia/grace.v1"
	"gopkg.in/tokopedia/logging.v1"
)

func main() {

	flag.Parse()
	logging.LogInit()

	debug := logging.Debug.Println

	debug("app started") // message will not appear unless run with -debug switch

	if err := agent.Listen(&agent.Options{}); err != nil {
		log.Fatal(err)
	}

	hwm := hello.NewHelloWorldModule()

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/hello", hwm.SayHelloWorld)
	http.HandleFunc("/api/users", user.HandlerGetUserList)
	http.HandleFunc("/users", user.HandlerUserHtmlPage)
	http.HandleFunc("/pingcounter", user.HandlerPingCounter)
	go logging.StatsLog()

	tracer.Init(&tracer.Config{Port: 8700, Enabled: true})

	log.Fatal(grace.Serve(":9002", nil))
}
