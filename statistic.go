package main

import (
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
	totalSession     int64
	totalSessionRate int64
	status           string
	servers          sortServer
}

type statistics struct {
	totalSession     int64
	totalSessionRate int64
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
		lbs                 loadbalancers
		stats               statistics
		str                 strings.Builder
		tmpS                server
		ival                int64
		err                 error
		bIgnoreHighSessions bool
		scur                int64
		rate                int64
	)

	for LbName, Loadbalancer := range *LB {
		lbstats := new(loadbalancer)
		lbstats.name = LbName
		for GroupName, Group := range Loadbalancer {
			for ServerName, Server := range Group {
				scur, _ = strconv.ParseInt(Server["scur"], 10, 64)
				rate, _ = strconv.ParseInt(Server["rate"], 10, 64)

				if ServerName == "FRONTEND" {
					stats.totalSession += scur
					lbstats.totalSession += scur
					log.Notef("Adding sessions %d to lb %s from group %s and server %s", scur, LbName, GroupName, ServerName)

					stats.totalSessionRate += rate
					lbstats.totalSessionRate += rate
					log.Notef("Adding sessionsrate %d to lb %s from group %s and server %s", Server["rate"], LbName, GroupName, ServerName)
				} else if ServerName != "BACKEND" {
					if Server["tracked"] == "0" {
						switch Server["status"] {
						case "UP":
							if bIgnoreHighSessions, _ = config.Loadbalancers[LbName].Options["ignoreHighSessions"].(bool); !bIgnoreHighSessions {
								if scur > 150 {
									str.Reset()
									str.WriteString(Server["status"])
									str.WriteString(" has high current sessions: ")
									str.WriteString(Server["scur"])

									tmpS.name = ServerName
									tmpS.group = GroupName
									tmpS.status = str.String()
									lbstats.servers = append(lbstats.servers, tmpS)
								}
							}
						case "0":
							log.Warningf("ERROR CASE 0 ON %s %s %s", ServerName, GroupName, LbName)
						case "no check":
							if !config.HideNoCheck {
								tmpS.name = ServerName
								tmpS.group = GroupName
								tmpS.status = "Server has no check defined!"
								lbstats.servers = append(lbstats.servers, tmpS)
							}
						default:
							str.Reset()
							str.WriteString(Server["status"])
							str.WriteString(" for ")
							ival, err = strconv.ParseInt(Server["lastchg"], 10, 64)
							if err != nil {
								str.WriteString(Server["lastchg"])
								str.WriteString(" seconds")
							} else {
								str.WriteString((time.Duration(ival) * time.Second).String())
							}

							tmpS.name = ServerName
							tmpS.group = GroupName
							tmpS.status = str.String()
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
