package lib

import (
	"fmt"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
)

func (m *Mackerel) CheckService() (bool, error) {
	t, err := m.client.FindServices()
	if err != nil {
		return false, err
	}
	for _, v := range t {
		if v.Name == m.service {
			return true, nil
		}
	}

	return false, nil
}

func (m *Mackerel) FetchHosts() ([]string, error) {
	var hosts []string
	t, err := m.client.FindHosts(&mackerel.FindHostsParam{
		Service: m.service,
		Roles:   []string{m.roles},
	})

	if err != nil {
		return hosts, err
	}

	if m.filter != "" {
		for _, v := range t {
			if strings.Contains(v.Name, m.filter) {
				hosts = append(hosts, v.ID)
			}
		}
	} else {
		for _, v := range t {
			hosts = append(hosts, v.ID)
		}
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("No hosts")
	}

	for i, host := range hosts {
		if opts.Debug {
			logger.Debug(fmt.Sprintf("[DEBUG] host[%d]: %s", i, host))
			time.Sleep(time.Millisecond * 100)
		}
	}

	return hosts, nil
}

func (m *Mackerel) FetchMetrics(hostId, metrics string) (map[time.Time]float64, error) {
	dt := time.Now()
	from_unix := dt.Unix()
	to_unix := dt.Unix()

	res := make(map[time.Time]float64)
	for i := 1; i <= opts.TimeWindow*2; i++ {
		from_unix = dt.Add(-12 * time.Hour * time.Duration(i)).Unix()
		metrics, err := m.client.FetchHostMetricValues(hostId, metrics, from_unix, to_unix)
		if err != nil {
			return res, err
		}

		for _, m := range metrics {
			dtFromUnix := time.Unix(m.Time, 0)
			res[dtFromUnix] = m.Value.(float64)
		}

		to_unix = from_unix
	}

	return res, nil
}

func (m *Mackerel) GetHostName(hostId string) (string, error) {
	name, err := m.client.FindHost(hostId)
	if err != nil {
		return "", err
	}

	return name.Name, nil
}
