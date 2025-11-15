package persistentLogger

import (
	"go-cache-server-mini/internal/core/data"
	"go-cache-server-mini/internal/util"
)

type AOF struct {
	AofDataChannel    chan string
	AofControlChannel chan string
	AofFile           *util.FileUtil
	AofTempFile       *util.FileUtil
	tempFileFlag      bool
	parser            *Parser
	AofPath           string
	done              chan struct{}
}

func NewAOF(aofDataChannel chan string, aofControlChannel chan string, aofPath string) *AOF {
	// check folder path and create AOF file if not exists
	AofFileUtil := util.NewFileUtil(aofPath, aofPath+"/cache.aof")
	AofTempFileUtil := util.NewFileUtil(aofPath, aofPath+"/cache.aof.temp")
	aof := &AOF{
		AofDataChannel:    aofDataChannel,
		AofControlChannel: aofControlChannel,
		AofFile:           AofFileUtil,
		AofTempFile:       AofTempFileUtil,
		tempFileFlag:      false,
		parser:            NewParser(),
		AofPath:           aofPath,
		done:              make(chan struct{}),
	}
	go aof.Save()
	return aof
}

func (a *AOF) Load(data map[string]data.CacheItem) (map[string]data.CacheItem, error) {
	// loading AOF data into cache
	data, err := a.loadFromFile(a.AofFile, data)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (a *AOF) loadFromFile(fileUtil *util.FileUtil, data map[string]data.CacheItem) (map[string]data.CacheItem, error) {
	// if aof temp file exists, read first
	lines, err := fileUtil.Load()
	if err != nil {
		return data, err
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

func (a *AOF) Save() error {
	defer func() {
		a.close()
		close(a.done)
	}()
	for {
		select {
		case control, controlOk := <-a.AofControlChannel:
			if !controlOk {
				continue
			}
			// Handle control messages
			switch control {
			case "PAUSE":
				a.tempFileFlag = true
				a.AofTempFile.Truncate() // create/truncate temp file
			case "RESUME":
				util.SwitchFileUtil(a.AofTempFile, a.AofFile) // Switch temp file to main file & delete temp file
				a.tempFileFlag = false
			}
		case cmd, cmdOk := <-a.AofDataChannel:
			if !cmdOk {
				return nil
			}
			switch a.tempFileFlag {
			case true:
				a.AofTempFile.Write(cmd) // Write to temp file
			case false:
				a.AofFile.Write(cmd) // Write to main file
			}
		}
	}
}

func (a *AOF) close() error {
	// close file handles, flush buffers, etc.
	a.AofTempFile.CloseFile()
	a.AofFile.CloseFile()
	return nil
}

func (a *AOF) Wait() {
	<-a.done
}
