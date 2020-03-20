package haproxy

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MaxBosse/hamon/log"
)

type GlobalMap map[string]LoadbalancerMap
type LoadbalancerMap map[string]GroupMap
type GroupMap map[string]map[string]string

// Loadbalancer ...
type Loadbalancer struct {
	Name    string
	Urls    []string
	Options map[string]interface{}
}

type lbChan struct {
	Name string
	Body string
}

var (
	timeout = time.Duration(5 * time.Second)
	client  = http.Client{
		Timeout: timeout,
	}
)

// SetTimeout allows setting the Timeout in seconds for all http requests
func SetTimeout(timeout int) {
	timeoutDuration := time.Duration(time.Duration(timeout) * time.Second)
	client.Timeout = timeoutDuration
}

// Load requests the CSV data for all loadbalancers and merges the data into one GlobalMap
func Load(Loadbalancers map[string]Loadbalancer) (*GlobalMap, error) {
	if len(Loadbalancers) == 0 {
		return nil, errors.New("No loadbalancers to load")
	}

	LB := make(GlobalMap)
	c := make(chan lbChan)
	counter := 0

	for _, loadbalancer := range Loadbalancers {
		LB[loadbalancer.Name] = make(LoadbalancerMap)

		for _, url := range loadbalancer.Urls {
			counter++
			go getCsv(url, loadbalancer.Name, c)
		}

	}

	var lbc lbChan
	for {
		select {
		case lbc = <-c:
			r := csv.NewReader(strings.NewReader(lbc.Body))
			r.TrimLeadingSpace = true
			log.Noteln("start unmarshaling ", lbc.Name)
			for {
				err := unmarshal(r, LB[lbc.Name])
				if err == io.EOF {
					break
				}
				if err != nil {

					return nil, err
				}
			}
			log.Noteln("end unmarshaling ", lbc.Name)
			counter--
			if counter == 0 {
				return &LB, nil
			}
		}
	}
}

func getCsv(url string, name string, c chan lbChan) {
	defer func() {
		if r := recover(); r != nil {
			var tempa lbChan
			log.Errorln("recovered", r)
			c <- tempa
		}
	}()
	var tempa lbChan
	tempa.Name = name
	log.Noteln("Requesting", url)

	resp, err := client.Get(url)
	if err != nil {
		log.Panicln(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicln(err)
	}
	tempa.Body = string(body)

	log.Noteln("Done request", url)
	c <- tempa
}

func unmarshal(reader *csv.Reader, thisLB LoadbalancerMap) error {
	// Avoid allocs in the for loop
	var (
		y         int
		i         int
		keyName   string
		oldValue  string
		ok        bool
		ival      int64
		oval      int64
		err       error
		olderr    error
		record    []string
		tempStats map[string]string
	)

	positionMap := make(map[int]string)
	statsToLoad := make(map[string]bool)
	statsToLoad["scur"] = true
	statsToLoad["rate"] = true
	statsToLoad["tracked"] = true
	statsToLoad["status"] = true
	statsToLoad["lastchg"] = true
	statsToLoad["pxname"] = true
	statsToLoad["svname"] = true

	// Read a line
	firstLine := true
	for {
		tempStats = make(map[string]string)

		record, err = reader.Read()
		if err != nil {
			return err
		}

		if firstLine {
			for y = 0; y < len(record); y++ {
				if y == 0 {
					positionMap[y] = record[y][2:]
					continue
				}
				positionMap[y] = record[y]
			}

			firstLine = false

			continue
		}

		for i = 0; i < len(record); i++ {
			keyName, _ = positionMap[i]
			if _, ok = statsToLoad[keyName]; ok {
				if record[i] == "" {
					record[i] = "0"
				}

				if keyName == "pxname" || keyName == "svname" || keyName == "lastchg" {
					tempStats[keyName] = record[i]
					continue
				}

				oldValue = thisLB[tempStats["pxname"]][tempStats["svname"]][keyName]

				if oldValue != record[i] && oldValue != "" {
					ival, err = strconv.ParseInt(record[i], 10, 64)
					oval, olderr = strconv.ParseInt(oldValue, 10, 64)
					// Merge as string if one record could not be converted to an integer
					if err != nil || olderr != nil {
						var buffer bytes.Buffer
						buffer.WriteString(oldValue)
						buffer.WriteString(",")
						buffer.WriteString(record[i])
						log.Debugf("%d: Combining strings to %s", i, buffer.String())
						tempStats[keyName] = buffer.String()
						continue
					}

					tempStats[keyName] = strconv.FormatInt(ival+oval, 10)
					continue
				}

				tempStats[keyName] = record[i]
			}
		}

		if len(thisLB[tempStats["pxname"]]) == 0 {
			thisLB[tempStats["pxname"]] = make(GroupMap)
		}
		thisLB[tempStats["pxname"]][tempStats["svname"]] = tempStats

		log.Debugf("%+v", thisLB[tempStats["pxname"]][tempStats["svname"]])
	}
}
