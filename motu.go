package motu

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type motu struct {
	Addr      string
	Datastore Datastore
	client    *http.Client
}

func NewMotu(addr string) *motu {
	m := new(motu)
	m.Addr = addr
	m.Datastore = make(Datastore, 500)
	return m
}

func (m *motu) StartListener(ch chan *Event) {
	l := new(Listener)
	l.Motu = m
	l.Ch = ch
	l.Start()
}

type singleFloatValue struct {
	Value float64 `json:"value"`
}

type singleIntValue struct {
	Value int64 `json:"value"`
}

func (m *motu) post(path string, data []byte) error {
	if m.client == nil {
		tr := &http.Transport{
			MaxIdleConns:    1,
			IdleConnTimeout: 30 * time.Second,
		}
		m.client = &http.Client{Transport: tr}
	}

	payload := url.Values{"json": {string(data)}}
	req, err := http.NewRequest("PATCH", "http://"+m.Addr+"/datastore/"+path, strings.NewReader(payload.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.client.Do(req)
	if err != nil {
		// Timeout o.i.d.
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("got HTTP %d response", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(body) != 0 {
		return fmt.Errorf("Expected empty response: %v", body)
	}
	return nil
}

func (m *motu) setSingleFloatValue(path string, value float64) error {
	floatValue := new(singleFloatValue)
	floatValue.Value = value

	data, err := json.Marshal(floatValue)
	if err != nil {
		return fmt.Errorf("failed to create JSON: %v", err)
	}

	return m.post(path, data)
}

func (m *motu) setSingleIntValue(path string, value int64) error {
	intValue := new(singleIntValue)
	intValue.Value = value

	data, err := json.Marshal(intValue)
	if err != nil {
		return fmt.Errorf("failed to create JSON: %v", err)
	}

	return m.post(path, data)
}

func (m *motu) setSingleBoolValue(path string, value bool) error {
	if value {
		return m.setSingleIntValue(path, 1)
	} else {
		return m.setSingleIntValue(path, 0)
	}
}

func (m *motu) GetFaderPosition(id string) (float64, error) {
	value, ok := m.Datastore[id+"/fader"].(float64)
	if !ok {
		return 0, fmt.Errorf("Cannot convert %v to float", m.Datastore[id+"/fader"])
	}
	return value, nil
}

func (m *motu) SetFaderPosition(id string, value float64) error {
	return m.setSingleFloatValue(id+"/fader", value)
}

func (m *motu) GetFaderMute(id string) (bool, error) {
	value, ok := m.Datastore[id+"/mute"].(float64)
	if !ok {
		return false, fmt.Errorf("Cannot convert %v to int", m.Datastore[id+"/mute"])
	}
	return value == 1, nil
}

func (m *motu) SetFaderMute(id string, value bool) error {
	return m.setSingleBoolValue(id+"/mute", value)
}

func (m *motu) ToggleFaderMute(id string, value bool) error {
	value, err := m.GetFaderMute(id)
	if err != nil {
		return err
	}
	return m.setSingleBoolValue(id+"/mute", !value)
}
