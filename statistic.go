package main

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MaxBosse/hamon/haproxy"
	"github.com/MaxBosse/hamon/log"
)

type loadbalancers []loadbalancer

func (slice loadbalancers) Len() int {
	return len(slice)
}

func (slice loadbalancers) Less(i, j int) bool {
	return slice[i].name < slice[j].name
}

func (slice loadbalancers) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type server struct {
	name   string
	group  string
	status string
}

type sortServer []server

// From SortMultiKeys example of https://golang.org/pkg/sort/
type lessFunc func(p1, p2 *server) bool

type multiSorter struct {
	servers []server
	less    []lessFunc
}

func (ms *multiSorter) Sort(servers []server) {
	ms.servers = servers
	sort.Sort(ms)
}

func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

func (ms *multiSorter) Len() int {
	return len(ms.servers)
}

func (ms *multiSorter) Swap(i, j int) {
	ms.servers[i], ms.servers[j] = ms.servers[j], ms.servers[i]
}

func (ms *multiSorter) Less(i, j int) bool {
	p, q := &ms.servers[i], &ms.servers[j]
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	return ms.less[k](p, q)
}

type loadbalancer struct {
	name             string
	totalSession     int
	totalSessionRate int
	status           string
	servers          sortServer
}

type statistics struct {
	totalSession     int
	totalSessionRate int
	loadbalancers    loadbalancers
}

// End

func leftPad(s string, padStr string, pLen int) string {
	return strings.Repeat(padStr, pLen) + s
}

func rightPad(s string, padStr string, pLen int) string {
	return s + strings.Repeat(padStr, pLen)
}

func rightPad2Len(s string, padStr string, overallLen int) string {
	var padCountInt int
	padCountInt = 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = s + strings.Repeat(padStr, padCountInt)
	return retStr[:overallLen]
}
func leftPad2Len(s string, padStr string, overallLen int) string {
	var padCountInt int
	padCountInt = 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = strings.Repeat(padStr, padCountInt) + s
	return retStr[(len(retStr) - overallLen):]
}

func process(LB *haproxy.GlobalMap, config Config) statistics {
	var (
		lbs   loadbalancers
		stats statistics
	)

	for LbName, Loadbalancer := range *LB {
		lbstats := new(loadbalancer)
		lbstats.name = LbName
		for GroupName, Group := range Loadbalancer {
			for ServerName, Server := range Group {
				if ServerName == "FRONTEND" {
					stats.totalSession += Server.Scur
					lbstats.totalSession += Server.Scur
					log.Notef("Adding sessions %d to lb %s from group %s and server %s", Server.Scur, LbName, GroupName, ServerName)

					stats.totalSessionRate += Server.Rate
					lbstats.totalSessionRate += Server.Rate
					log.Notef("Adding sessionsrate %d to lb %s from group %s and server %s", Server.Rate, LbName, GroupName, ServerName)
				} else if ServerName != "BACKEND" {
					if Server.Tracked == "" {
						switch Server.Status {
						case "UP":
							if Server.Scur > 50 {
								var buffer bytes.Buffer
								buffer.WriteString(Server.Status)
								buffer.WriteString(" has high current sessions: ")
								buffer.WriteString(strconv.Itoa(Server.Scur))

								var tmpS server
								tmpS.name = ServerName
								tmpS.group = GroupName
								tmpS.status = buffer.String()
								lbstats.servers = append(lbstats.servers, tmpS)
							}
						case "0":
							log.Warningf("ERROR CASE 0 ON %s %s %s", ServerName, GroupName, LbName)
						case "no check":
							if !config.HideNoCheck {
								var tmpS server
								tmpS.name = ServerName
								tmpS.group = GroupName
								tmpS.status = "Server has no check defined!"
								lbstats.servers = append(lbstats.servers, tmpS)
							}
						default:
							var buffer bytes.Buffer
							buffer.WriteString(Server.Status)
							buffer.WriteString(" for ")
							ival, err := strconv.ParseInt(Server.Lastchg, 10, 0)
							if err != nil {
								buffer.WriteString(Server.Lastchg)
								buffer.WriteString(" seconds")
							} else {
								buffer.WriteString((time.Duration(ival) * time.Second).String())
							}

							var tmpS server
							tmpS.name = ServerName
							tmpS.group = GroupName
							tmpS.status = buffer.String()
							lbstats.servers = append(lbstats.servers, tmpS)
						}
					}
				}
			}

		}
		lbs = append(lbs, *lbstats)
	}
	stats.loadbalancers = lbs
	return stats
}
