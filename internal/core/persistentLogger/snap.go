package persistentLogger

import (
	"go-cache-server-mini/internal/core/data"
	"go-cache-server-mini/internal/util"
	"log"
)

type Snap struct {
	// Snapshot related fields and methods
	SnapDataChannel chan map[string]data.CacheItem
	SnapDoneChannel chan bool
	SnapFile        *util.FileUtil
	SnapTempFile    *util.FileUtil
	parser          *Parser
	snapPath        string
	done            chan struct{}
}

func NewSnap(snapDataChannel chan map[string]data.CacheItem, snapDoneChannel chan bool, snapPath string) *Snap {
	// check folder path and create Snap file if not exists
	SnapFileUtil := util.NewFileUtil(snapPath, snapPath+"/cache.snap")
	SnapTempFileUtil := util.NewFileUtil(snapPath, snapPath+"/cache.snap.temp")
	snap := &Snap{
		SnapDataChannel: snapDataChannel,
		SnapDoneChannel: snapDoneChannel,
		SnapFile:        SnapFileUtil,
		SnapTempFile:    SnapTempFileUtil,
		parser:          NewParser(),
		snapPath:        snapPath,
		done:            make(chan struct{}),
	}
	go snap.Save()
	return snap
}

func (s *Snap) Load(data map[string]data.CacheItem) (map[string]data.CacheItem, error) {
	// loading Snap data into cache
	data, err := s.loadFromFile(s.SnapFile, data)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (s *Snap) loadFromFile(fileUtil *util.FileUtil, data map[string]data.CacheItem) (map[string]data.CacheItem, error) {
	// if no data loaded from temp file, read main snap file
	lines, err := fileUtil.Load()
	if err != nil {
		return data, err
	}
	for _, line := range lines {
		_, key, item, parseErr := s.parser.ParseStringToCMD(line)
		if parseErr != nil {
			return nil, parseErr
		}
		data[key] = item
	}
	return data, nil
}

func (s *Snap) Save() error {
	defer s.close()
	// Placeholder for saving Snap data from cache
	for data := range s.SnapDataChannel {
		// Process the snapshot data
		if len(data) == 0 {
			// If no data, just truncate the snap file
			if err := s.SnapFile.Truncate(); err != nil {
				log.Printf("Error truncating snap file: %v", err)
			}
		} else {
			// Write data to temp snap file
			if err := s.SnapTempFile.Truncate(); err != nil {
				log.Printf("Error truncating temp snap file: %v", err)
				s.SnapDoneChannel <- false
				continue
			}
			for key, item := range data {
				cmd, err := s.parser.ConvertCMDToString("SET", key, item)
				if err != nil {
					log.Printf("Error converting CMD to string for snap: %v", err)
					continue
				}
				if err = s.SnapTempFile.Write(cmd); err != nil {
					log.Printf("Error writing to temp snap file: %v", err)
					s.SnapDoneChannel <- false
					continue
				}
			}
			if err := util.SwitchFileUtil(s.SnapTempFile, s.SnapFile); err != nil { // Switch temp file to main file & delete temp file
				log.Printf("Error switching snap files: %v", err)
				s.SnapDoneChannel <- false
				continue
			}
		}
		s.SnapDoneChannel <- true
	}
	return nil
}

func (s *Snap) close() error {
	// closing Snap resources
	close(s.SnapDoneChannel)
	s.SnapFile.CloseFile()
	s.SnapTempFile.CloseFile()
	close(s.done)
	return nil
}

func (s *Snap) Wait() {
	<-s.done
}
