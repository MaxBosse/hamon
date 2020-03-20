package main

import (
	"flag"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/MaxBosse/hamon/haproxy"
	"github.com/MaxBosse/hamon/log"
)

var (
	// BuildTime of the build provided by the build command
	BuildTime = "Not provided"
	// GitHash of build provided by the build command
	GitHash = "Not provided"
	// GitBranch of the build provided by the build command
	GitBranch = "Not provided"

	// Closures that order the Change structure.
	name = func(c1, c2 *server) bool {
		return c1.name < c2.name
	}
	group = func(c1, c2 *server) bool {
		return c1.group < c2.group
	}
)

const (
	// Name of the Application
	Name = "HaMon"
	// Version of the Application
	Version = "0.0.4"

	normGraph = "├──"
	endGraph  = "└──"
)

func loop(config Config) {
	for {
		//log.Print("Starting loading...")
		LB, err := haproxy.Load(config.Loadbalancers)
		if err != nil {
			panic(err)
		}
		//log.Print("Loading done.")

		stats := process(LB, config)

		sort.Sort(stats.loadbalancers)

		fmt.Printf("\033[H\033[2J")
		fmt.Println(Name, Version, "-", GitHash, GitBranch, BuildTime)
		fmt.Println("Global Sessions:", stats.totalSession, "\t\tGlobal SessionsRate:", stats.totalSessionRate)
		fmt.Println("")
		for _, loadbalancer := range stats.loadbalancers {
			fmt.Println(rightPad2Len(loadbalancer.name, " ", 10), "Sessions:", rightPad2Len(strconv.FormatInt(loadbalancer.totalSession, 10), " ", 5), "SessionRate:", loadbalancer.totalSessionRate)

			OrderedBy(group, name).Sort(loadbalancer.servers)
			i := 1
			for _, Server := range loadbalancer.servers {
				graph := normGraph
				if i == len(loadbalancer.servers) {
					graph = endGraph
				}

				fmt.Println(graph, rightPad2Len(Server.group, " ", 20), rightPad2Len(Server.name, " ", 10), Server.status)
				i++
			}

			fmt.Println("")
		}
		fmt.Println("")
		t := time.Now()
		fmt.Printf("Last update: %s\n", t.Format("2006-01-02 15:04:05"))

		time.Sleep(time.Second * 5)
	}
}

func main() {
	var (
		configPath = flag.String("config", "config.yml", "Path to yml configuration file")
		logLevel   = flag.String("loglevel", "error", "LogLevel [error|warning|note|debug]")
		config     = Config{
			HideNoCheck: false,
			Timeout:     5,
		}
	)

	flag.Parse()

	log.SetLevel(*logLevel)

	log.Noteln("Starting up...")

	config.Load(*configPath)
	haproxy.SetTimeout(config.Timeout)

	loop(config)
}
