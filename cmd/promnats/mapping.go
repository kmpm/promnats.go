package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// fileMappings reads filename, line by line and adds them to portmap if valid
func fileMappings(filename string) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}
	var fil *os.File
	// check filename if path exists
	info, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fil, err = os.Create(filename)
			if err != nil {
				return err
			}

		}
		return err
	} else {
		// cant be a dir
		if info.IsDir() {
			return errors.New("path is a directory")
		}
		// open the file
		fil, err = os.Open(filename)
		if err != nil {
			return err
		}
	}
	defer fil.Close()

	// read line by line
	s := bufio.NewScanner(fil)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		// trim any bad characters
		line := strings.Trim(s.Text(), " \t\r\n")
		// skip # comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}
		// skip empty lines
		if line == "" {
			continue
		}
		// parse and possibly add
		err = addPortmap(line)
		if err != nil {
			return err
		}
	}
	return nil
}
