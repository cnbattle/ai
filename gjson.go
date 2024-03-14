package ai

import (
	"github.com/tidwall/gjson"
)

// JsonParse parses the json and returns a result.
func JsonParse(json string) gjson.Result {
	return gjson.Parse(json)

}

// JsonGet searches json for the specified path.
func JsonGet(json, path string) gjson.Result {
	return gjson.Get(json, path)
}

// ParseJson parses the json and returns a result.
func ParseJson(json string) gjson.Result {
	return gjson.Parse(json)

}

// GetJson searches json for the specified path.
func GetJson(json, path string) gjson.Result {
	return gjson.Get(json, path)
}
