package motu

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Listener struct {
	Motu   *motu
	Ch     chan *Event
	client *http.Client
	etag   string
}

func (l *Listener) Start() {
	tr := &http.Transport{
		MaxIdleConns:    1,
		IdleConnTimeout: 30 * time.Second,
	}
	l.client = &http.Client{Transport: tr}

	initialized := false

	for {
		datastore, err := l.fetchDatastore()
		if err != nil {
			log.Printf("Fetching datastore failed: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for k, v := range *datastore {
			l.Motu.Datastore[k] = v
			if initialized {
				event := new(Event)
				event.Path = k
				event.Value = v
				l.Ch <- event
			}
		}
		initialized = true
	}
}

func (l *Listener) fetchDatastore() (*Datastore, error) {
	for {
		req, err := http.NewRequest("GET", "http://"+l.Motu.Addr+"/datastore", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create http request: %v", err)
		}
		if l.etag != "" {
			req.Header.Set("If-None-Match", l.etag)
		}
		resp, err := l.client.Do(req)
		if err != nil {
			// Timeout o.i.d.
			return nil, fmt.Errorf("http request failed: %v", err)
		}
		defer resp.Body.Close()

		if l.etag != "" && resp.StatusCode == 304 {
			// No change
			// MOTU API will have waited 10 seconds before returning HTTP 304, so
			// we can immediately continue
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading HTTP response failed: %v", err)
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("got HTTP %d response", resp.StatusCode)
		}

		if !json.Valid(body) {
			return nil, fmt.Errorf("got invalid JSON response: %v", body)
		}

		datastore := new(Datastore)
		if err := json.Unmarshal(body, datastore); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %v", body)
		}

		l.etag = resp.Header.Get("Etag")

		log.Printf("Got datastore, etag is now: %v\n", l.etag)
		log.Printf("Datastore = %v\n", datastore)

		return datastore, nil
	}
}
