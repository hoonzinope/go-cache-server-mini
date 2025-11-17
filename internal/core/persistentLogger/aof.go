package persistentLogger

import (
	"go-cache-server-mini/internal/core/data"
	"go-cache-server-mini/internal/util"
	"time"
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
	batchCmdBuffer    []string
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
		batchCmdBuffer:    make([]string, 0, 1000), // buffer for batch commands
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
	lines, err := fileUtil.Load()
	if err != nil {
		return data, err
	}
	for _, line := range lines {
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
	batchTicker := time.NewTicker(100 * time.Millisecond)
	defer func() {
		batchTicker.Stop()
		a.close()
		close(a.done)
	}()
	for {
		select {
		case control, controlOk := <-a.AofControlChannel:
			if !controlOk {
				if len(a.batchCmdBuffer) > 0 {
					if err := a.flush(); err != nil {
						return err
					}
				}
				return nil
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
				if len(a.batchCmdBuffer) > 0 {
					if err := a.flush(); err != nil {
						return err
					}
				}
				return nil
			}
			a.batchCmdBuffer = append(a.batchCmdBuffer, cmd)
			if len(a.batchCmdBuffer) >= 1000 {
				if err := a.flush(); err != nil {
					return err
				}
			}
		case <-batchTicker.C:
			if len(a.batchCmdBuffer) > 0 {
				if err := a.flush(); err != nil {
					return err
				}
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

func (a *AOF) flush() error {
	var err error
	var targetFile *util.FileUtil
	if a.tempFileFlag {
		targetFile = a.AofTempFile
	} else {
		targetFile = a.AofFile
	}
	for _, cmd := range a.batchCmdBuffer {
		err = targetFile.Write(cmd) // Write to target file
		if err != nil {
			return err
		}
	}
	a.batchCmdBuffer = a.batchCmdBuffer[:0] // Reset buffer
	return nil
}
