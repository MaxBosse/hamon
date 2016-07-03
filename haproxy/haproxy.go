package haproxy

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/MaxBosse/hamon/log"
)

type GlobalMap map[string]map[string]map[string]Stats
type LoadbalancerMap map[string]map[string]Stats
type GroupMap map[string]Stats

// Loadbalancer ...
type Loadbalancer struct {
	Name string
	Urls []string
}

// Stats ...
type Stats struct {
	Pxname        string `strict:"true"` // we use it for merging
	Svname        string `strict:"true"` // we use it for merging
	Qcur          string
	Qmax          string
	Scur          int
	Smax          int
	Slim          string
	Stot          int
	Bin           int
	Bout          int
	Dreq          string
	Dresp         int
	Ereq          string
	Econ          string
	Eresp         string
	Wretr         string
	Wredis        string
	Status        string
	Weight        string
	Act           string
	Bck           string
	Chkfail       string
	Chkdown       string
	Lastchg       string `strict:"true"` // seconds since last change should not be summed up
	Downtime      string
	Qlimit        string
	Pid           int
	Iid           int
	Sid           int
	Throttle      string
	Lbtot         string
	Tracked       string
	Type          int
	Rate          int
	RateLim       string
	RateMax       int
	CheckStatus   string
	CheckCode     string
	CheckDuration string
	Hrsp1xx       int
	Hrsp2xx       int
	Hrsp3xx       int
	Hrsp4xx       int
	Hrsp5xx       int
	HrspOther     int
	Hanafail      string
	ReqRate       string
	ReqRateMax    string
	ReqTot        string
	CliAbrt       string
	SrvAbrt       string
	CompIn        string
	CompOut       string
	CompByp       string
	CompRsp       string
	Lastsess      string
	LastChk       string
	LastAgt       string
	Qtime         string
	Ctime         string
	Rtime         string
	Ttime         string
	Dummy         string
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
			r.Comment = '#'
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
	tempStats := Stats{}

	// Read a line
	record, err := reader.Read()
	if err != nil {
		return err
	}

	s := reflect.ValueOf(&tempStats).Elem()
	sT := reflect.TypeOf(tempStats)

	// Make sure it fits into our stats struct
	if s.NumField() != len(record) {
		return &fieldMismatch{s.NumField(), len(record)}
	}

	// Start iterating over each field
	for i := 0; i < s.NumField(); i++ {

		fT := sT.Field(i)
		f := s.Field(i)
		switch f.Type().String() {
		case "string":
			// Skip merging for strict fields
			if fT.Tag.Get("strict") == "true" {
				f.SetString(record[i])
				continue
			}

			if _, ok := thisLB[tempStats.Pxname][tempStats.Svname]; !ok {
				f.SetString(record[i])
				continue
			} else {
				oldValue := reflect.ValueOf(thisLB[tempStats.Pxname][tempStats.Svname]).Field(i).String()

				if oldValue != record[i] {
					ival, err := strconv.ParseInt(record[i], 10, 0)
					oval, olderr := strconv.ParseInt(oldValue, 10, 0)
					// Merge as string if one record could not be converted to an integer
					if err != nil || olderr != nil {
						var buffer bytes.Buffer
						buffer.WriteString(oldValue)
						buffer.WriteString(",")
						buffer.WriteString(record[i])
						log.Debugf("%d: Combining strings to %s", i, buffer.String())
						f.SetString(buffer.String())
						continue
					}

					log.Debugf("%d: Adding strings to %d", i, oval)
					f.SetString(strconv.FormatInt(ival, 10))
					continue
				} else {
					f.SetString(record[i])
					continue
				}
			}
		case "int":
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return err
			}

			// Skip merging for strict fields
			if fT.Tag.Get("strict") == "true" {
				f.SetInt(ival)
				continue
			}

			// Add to already existing number if server already exists in this lb+group
			if _, ok := thisLB[tempStats.Pxname][tempStats.Svname]; ok {
				ival += reflect.ValueOf(thisLB[tempStats.Pxname][tempStats.Svname]).Field(i).Int()
			}
			f.SetInt(ival)
			continue
		default:
			return &unsupportedType{f.Type().String()}
		}
	}

	if len(thisLB[tempStats.Pxname]) == 0 {
		thisLB[tempStats.Pxname] = make(GroupMap)
	}
	thisLB[tempStats.Pxname][tempStats.Svname] = tempStats

	return nil
}

type fieldMismatch struct {
	expected, found int
}

func (e *fieldMismatch) Error() string {
	return "CSV line fields mismatch. Expected " + strconv.Itoa(e.expected) + " found " + strconv.Itoa(e.found)
}

type unsupportedType struct {
	Type string
}

func (e *unsupportedType) Error() string {
	return "Unsupported type: " + e.Type
}
