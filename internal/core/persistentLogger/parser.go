package persistentLogger

import (
	"encoding/json"
	"go-cache-server-mini/internal/core/data"
)

type Parser struct{}

type LineFormat struct {
	Cmd  string
	Key  string
	Item data.CacheItem
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ConvertCMDToString(cmd, key string, item data.CacheItem) string {
	line := LineFormat{
		Cmd:  cmd,
		Key:  key,
		Item: item,
	}
	jsonBytes, _ := json.Marshal(line)
	return string(jsonBytes)
}

func (p *Parser) ParseStringToCMD(line string) (cmd string, key string, item data.CacheItem, err error) {
	var lineFormat LineFormat
	if err := json.Unmarshal([]byte(line), &lineFormat); err != nil {
		return "", "", data.CacheItem{}, err
	}
	cmd = lineFormat.Cmd
	key = lineFormat.Key
	item = lineFormat.Item

	return cmd, key, item, nil
}
