package motu

type Datastore map[string]interface{}

type Event struct {
	Path  string
	Value interface{}
}
