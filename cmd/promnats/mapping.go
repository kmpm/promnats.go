package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

type PortMaps map[int]string

// parsePortmap takes a <port>:<ID> string input and parses it
func parsePortmap(s string) (port int, id string, err error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		err = fmt.Errorf("each mapping must contain 2 parts: %v", parts)
		return
	}
	id = parts[1]

	port, err = strconv.Atoi(parts[0])
	if err != nil {
		err = fmt.Errorf("value '%s' is not parsable as port number: %W", parts[0], err)
		return
	}

	if port > 65535 || port < 0 {
		err = fmt.Errorf("%d is not a valid port number", port)
	}

	return
}

// fileMappings reads filename, line by line and returns valid ones
func fileMappings(filename string) (out PortMaps, err error) {
	out = make(PortMaps)
	if filename == "" {
		err = fmt.Errorf("empty filename")
		return
	}
	var fil *os.File
	var info fs.FileInfo
	// check filename if path exists
	info, err = os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fil, err = os.Create(filename)
			if err != nil {
				return
			}
			fil.Close()
		}
		return
	} else {
		// cant be a dir
		if info.IsDir() {
			err = errors.New("path is a directory")
			return
		}
		// open the file
		fil, err = os.Open(filename)
		if err != nil {
			return
		}
	}
	defer fil.Close()

	// read line by line
	s := bufio.NewScanner(fil)
	s.Split(bufio.ScanLines)
	var port int
	var id string
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
		port, id, err = parsePortmap(line)
		if err != nil {
			return
		}
		out[port] = id
	}
	return
}
