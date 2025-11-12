package core

import (
	"fmt"
	"strconv"
	"time"
)

type Parser struct {
	timeFormat string
}

func NewParser() *Parser {
	return &Parser{
		timeFormat: "2006-01-02 15:04:05",
	}
}

func (p *Parser) ConvertCMDToString(cmd, key string, item cacheItem) string {
	value := string(item.value)
	expiration := item.expiration.Format(p.timeFormat)
	persistent := strconv.FormatBool(item.persistent)
	return cmd + " " + key + " " + value + " " + expiration + " " + persistent
}

func (p *Parser) ParseStringToCMD(line string) (cmd string, key string, item cacheItem, err error) {
	// Placeholder for parsing a command string into components
	var valueStr, expirationStr, persistentStr string

	_, err = fmt.Sscanf(line, "%s %s %v %s %s", &cmd, &key, &valueStr, &expirationStr, &persistentStr)
	if err != nil {
		return "", "", cacheItem{}, err
	}

	item.value = []byte(valueStr)

	expiration, parseErr := time.Parse(p.timeFormat, expirationStr)
	if parseErr != nil {
		return "", "", cacheItem{}, parseErr
	}
	item.expiration = expiration

	persistent, err := strconv.ParseBool(persistentStr)
	if err != nil {
		return "", "", cacheItem{}, err
	}
	item.persistent = persistent

	return cmd, key, item, nil
}
