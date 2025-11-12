package core

import (
	"context"
	"go-cache-server-mini/internal/util"
)

type AOF struct {
	ctx               context.Context
	AofDataChannel    chan string
	AofControlChannel chan string
	AofFile           *util.FileUtil
	AofTempFile       *util.FileUtil
	tempFileFlag      bool
	parser            *Parser
}

func NewAOF(ctx context.Context, aofDataChannel chan string, aofControlChannel chan string, aofPath string) *AOF {
	// check folder path and create AOF file if not exists
	AofFileUtil := util.NewFileUtil(aofPath, aofPath+"/cache.aof")
	AofTempFileUtil := util.NewFileUtil(aofPath, aofPath+"/cache.aof.temp")
	return &AOF{
		ctx:               ctx,
		AofDataChannel:    aofDataChannel,
		AofControlChannel: aofControlChannel,
		AofFile:           AofFileUtil,
		AofTempFile:       AofTempFileUtil,
		tempFileFlag:      false,
		parser:            NewParser(),
	}
}

func (a *AOF) Load(data map[string]cacheItem) (map[string]cacheItem, error) {
	// Placeholder for loading AOF data into cache
	lines, err := a.AofFile.Load()
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		// TODO: Parse line and load into cache
		cmd, key, item, parseErr := a.parser.ParseStringToCMD(line)
		if parseErr != nil {
			return nil, parseErr
		}
		switch cmd {
		case "SET":
			data[key] = item
		case "DEL":
			delete(data, key)
		}
	}
	return data, nil
}

func (a *AOF) Append(command string) error {
	// Placeholder for appending a command to the AOF
	for {
		select {
		case control := <-a.AofControlChannel:
			// Handle control messages
			switch control {
			case "PAUSE":
				a.tempFileFlag = true
			case "RESUME":
				util.SwitchFileUtil(a.AofTempFile, a.AofFile) // Switch temp file to main file
				a.tempFileFlag = false
			}
		case cmd := <-a.AofDataChannel:
			switch a.tempFileFlag {
			case true:
				a.AofTempFile.Write(cmd) // Write to temp file
			case false:
				a.AofFile.Write(cmd) // Write to main file
			}
		case <-a.ctx.Done():
			a.Close()
			return nil
		}
	}
}

func (a *AOF) Close() error {
	// Placeholder for closing AOF resources
	// close file handles, flush buffers, etc.
	if a.AofDataChannel != nil {
		for {
			if cmd, ok := <-a.AofDataChannel; ok {
				// Append remaining commands
				if a.tempFileFlag {
					a.AofTempFile.Write(cmd)
				} else {
					a.AofFile.Write(cmd)
				}
			} else {
				break
			}
		}
	}
	a.AofTempFile.CloseFile()
	a.AofFile.CloseFile()
	return nil
}
