package util

import (
	"bufio"
	"errors"
	"os"
)

type FileUtil struct {
	folderPath string
	filePath   string
	file       *os.File
}

func NewFileUtil(folderPath, filePath string) *FileUtil {

	fileUtil := &FileUtil{
		folderPath: folderPath,
		filePath:   filePath,
	}
	return fileUtil
}

func SwitchFileUtil(tempFileUtil, newFileUtil *FileUtil) error {
	// close old file handles
	tempFileUtil.CloseFile()
	newFileUtil.CloseFile()
	// rename temp file to new file
	err := os.Rename(tempFileUtil.filePath, newFileUtil.filePath)
	return err
}

// Write appends a line to the file (file handle not closed)
func (f *FileUtil) Write(line string) error {
	f.createIfNotExists()
	// open file in append mode & always keep it open until CloseFile is called
	if f.file == nil {
		file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		f.file = file
	}
	_, err := f.file.WriteString(line + "\n")
	if err != nil {
		return err
	}
	return nil
}

func (f *FileUtil) Truncate() error {
	f.createIfNotExists()
	// close file if already opened
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}
	// truncate file
	err := os.Truncate(f.filePath, 0)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileUtil) Load() (lines []string, err error) {
	// read all lines from file
	file, err := os.Open(f.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil // return empty slice if file does not exist
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func (f *FileUtil) createIfNotExists() error {
	// check folder path
	_, err := os.Stat(f.folderPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(f.folderPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// check file path
	_, fileErr := os.Stat(f.filePath)
	if os.IsNotExist(fileErr) {
		file, err := os.Create(f.filePath)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func (f *FileUtil) CloseFile() error {
	if f.file != nil {
		err := f.file.Close()
		if err != nil {
			return err
		}
		f.file = nil
	}
	return nil
}
