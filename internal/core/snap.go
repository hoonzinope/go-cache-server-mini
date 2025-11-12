package core

import (
	"context"
	"go-cache-server-mini/internal/util"
)

type Snap struct {
	// Placeholder for Snapshot related fields and methods
	ctx             context.Context
	SnapDataChannel chan map[string]cacheItem
	SnapDoneChannel chan bool
	SnapFile        *util.FileUtil
	SnapTempFile    *util.FileUtil
	parser          *Parser
}

func NewSnap(ctx context.Context, snapDataChannel chan map[string]cacheItem, snapDoneChannel chan bool, snapPath string) *Snap {
	// check folder path and create Snap file if not exists
	SnapFileUtil := util.NewFileUtil(snapPath, snapPath+"/cache.snap")
	SnapFileTempUtil := util.NewFileUtil(snapPath, snapPath+"/cache.snap.temp")
	return &Snap{
		ctx:             ctx,
		SnapDataChannel: snapDataChannel,
		SnapDoneChannel: snapDoneChannel,
		SnapFile:        SnapFileUtil,
		SnapTempFile:    SnapFileTempUtil,
		parser:          NewParser(),
	}
}

func (s *Snap) Load() (map[string]cacheItem, error) {
	// Placeholder for loading Snap data into cache
	var cacheData map[string]cacheItem = make(map[string]cacheItem)
	lines, err := s.SnapFile.Load()
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		// TODO: Parse line and load into cache
		_, key, item, parseErr := s.parser.ParseStringToCMD(line)
		if parseErr != nil {
			return nil, parseErr
		}
		cacheData[key] = item
	}
	return cacheData, nil
}

func (s *Snap) Save() error {
	// Placeholder for saving Snap data from cache
	for {
		select {
		case data := <-s.SnapDataChannel:
			// Process the snapshot data
			for key, item := range data {
				cmd := s.parser.ConvertCMDToString("SET", key, item)
				s.SnapTempFile.Write(cmd)
			}
			util.SwitchFileUtil(s.SnapTempFile, s.SnapFile) // Switch temp file to main file
			s.SnapDoneChannel <- true
		case <-s.ctx.Done():
			s.Close()
			return nil
		}
	}
}

func (s *Snap) Close() error {
	// Placeholder for closing Snap resources
	if s.SnapDataChannel != nil {
		for {
			if data, ok := <-s.SnapDataChannel; ok {
				// Process remaining snapshot data
				for key, item := range data {
					cmd := s.parser.ConvertCMDToString("SET", key, item)
					s.SnapTempFile.Write(cmd)
				}
			} else {
				break
			}
		}
	}
	s.SnapFile.CloseFile()
	s.SnapTempFile.CloseFile()
	return nil
}
