// Package integration exposes the same source identities shipped in the docs.
package integration

import (
	_ "embed"
	"encoding/json"
)

//go:embed SOURCES.json
var sourceJSON []byte

type Source struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Branch     string `json:"branch"`
}

func Sources() (upstream, donor Source, err error) {
	var data struct {
		Upstream Source `json:"upstream"`
		Donor    Source `json:"donor"`
	}
	err = json.Unmarshal(sourceJSON, &data)
	return data.Upstream, data.Donor, err
}
