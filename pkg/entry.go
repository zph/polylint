package polylint

import (
	"embed"
	"encoding/json"
)

//go:embed VERSION.json
var f embed.FS

type Metadata struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

var LibMetadata Metadata

func init() {

	var LibMetadata Metadata

	data, _ := f.ReadFile("VERSION.json")
	if err := json.Unmarshal(data, &LibMetadata); err != nil {
		panic(err)
	}
}
