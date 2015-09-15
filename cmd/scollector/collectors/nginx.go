package collectors

import (
	"bufio"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bosun.org/cmd/scollector/conf"
	"bosun.org/metadata"
	"bosun.org/opentsdb"
	"bosun.org/slog"
)

type Data struct {
	Connections uint64
	Accepts     uint64
	Handled     uint64
	Requests    uint64
	Reading     uint64
	Writing     uint64
	Waiting     uint64
}

const (
	nginxInfoUrl = "http://localhost:8080/nginx_status"
)

func init() {
	registerInit(func(c *conf.Conf) {
		var url string
		if c.Nginx.Url == "" {
			url = nginxInfoUrl
		} else {
			url = c.Nginx.Url
		}
		slog.Infof("Nginx Url: %s", url)
		collectors = append(
			collectors,
			&IntervalCollector{
				F: func() (opentsdb.MultiDataPoint, error) {
					return c_nginx_status(url)
				},
				Enable:   enableURL(url, "Nginx Status"),
				Interval: time.Second * 30,
			})
	})
}

func c_nginx_status(url string) (opentsdb.MultiDataPoint, error) {
	var md opentsdb.MultiDataPoint
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	d := new(Data)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		// Example response:
		// Active connections: 2
		// server accepts handled requests
		//  7127 7127 8628
		// Reading: 0 Writing: 1 Waiting: 1

		// From collectd
		if len(fields) == 3 {
			if strings.Compare(fields[0], "Active") == 0 &&
				strings.Compare(fields[1], "connections:") == 0 {
				d.Connections, _ = strconv.ParseUint(fields[2], 10, 64)
			} else {
				d.Accepts, _ = strconv.ParseUint(fields[0], 10, 64)
				d.Handled, _ = strconv.ParseUint(fields[1], 10, 64)
				d.Requests, _ = strconv.ParseUint(fields[2], 10, 64)

			}
		} else if len(fields) == 6 {
			if strings.Compare(fields[0], "Reading:") == 0 &&
				strings.Compare(fields[2], "Writing:") == 0 &&
				strings.Compare(fields[4], "Waiting:") == 0 {
				d.Reading, _ = strconv.ParseUint(fields[1], 10, 64)
				d.Writing, _ = strconv.ParseUint(fields[3], 10, 64)
				d.Waiting, _ = strconv.ParseUint(fields[5], 10, 64)
			}
		}
	}

	Add(&md, "nginx.connections", d.Connections, nil, metadata.Gauge, metadata.Count, "")
	Add(&md, "nginx.accepts", d.Accepts, nil, metadata.Counter, metadata.Count, "")
	Add(&md, "nginx.handled", d.Handled, nil, metadata.Counter, metadata.Count, "")
	Add(&md, "nginx.requests", d.Requests, nil, metadata.Counter, metadata.Count, "")
	Add(&md, "nginx.reading", d.Reading, nil, metadata.Gauge, metadata.Count, "")
	Add(&md, "nginx.writing", d.Writing, nil, metadata.Gauge, metadata.Count, "")
	Add(&md, "nginx.waiting", d.Waiting, nil, metadata.Gauge, metadata.Count, "")

	return md, nil
}
