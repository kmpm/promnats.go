package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kmpm/promnats.go"
	"github.com/nats-io/nats.go"
)

const (
	min_parts = 1
)

type pathinfo struct {
	Path     string               `json:"path,omitempty"`
	IsFile   bool                 `json:"is_file"`
	IsDir    bool                 `json:"is_dir"`
	Modified time.Time            `json:"modified,omitempty"`
	Children map[string]*pathinfo `json:"children,omitempty"`
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(opts.Dest)
	if err != nil {
		panic(err)
	}
	return abs
}

func work(nc *nats.Conn) error {
	destAbs := mustAbs(opts.Dest)
	absParts := strings.Split(destAbs, string(os.PathSeparator))

	msgs, err := doReq(context.TODO(), nil, flag.Arg(0), 0, nc)
	if err != nil {
		log.Fatalf("error doReq() = %v", err)
	}

	written := map[string][]string{}

	for i, msg := range msgs {
		id := strings.Trim(msg.Header.Get(promnats.HeaderPnID), ". ")
		if id == "" {
			log.Printf("no %s header", promnats.HeaderPnID)
			continue
		}
		parts := strings.Split(id, ".")
		if len(parts) < min_parts {
			log.Printf("id must have at least %d part(s): %s", min_parts, id)
			continue
		}

		// if last of absulute destination is same as first of p drop it
		for absParts[len(absParts)-1] == parts[0] || len(parts) != 0 {
			parts = parts[1:]
		}

		// build a path out of the id parts
		p := path.Join(parts[:len(parts)-1]...)
		p = path.Join(opts.Dest, p)

		err = os.MkdirAll(p, os.ModePerm)
		if err != nil {
			fmt.Printf("ERR cannot create dir %s: %v", p, err)
			continue
		}

		filename := path.Join(p, parts[len(parts)-1]+".txt")
		err = os.WriteFile(filename, msg.Data, 0644)
		if err != nil {
			log.Printf("ERR cannot write file %s: %v", filename, err)
			continue
		}
		if v, ok := written[parts[0]]; ok {
			written[parts[0]] = append(v, filename)
		} else {
			written[parts[0]] = []string{filename}
		}
		if opts.Debug {
			log.Printf("file[%d]: %s", i, filename)
		}
	}

	if opts.Debug {
		log.Printf("written %+v", written)
	}
	// for k, v := range written {

	// 	p := path.Join(opts.Dest, k)
	// 	htmlfile := path.Join(p, "index.html")

	// 	os.WriteFile(htmlfile, []byte(strings.Join(v, "\r\n")), os.ModePerm)
	// }
	active := pathinfo{Path: "metrics", Children: map[string]*pathinfo{}}
	err = cleanup(opts.Dest, opts.MaxAge, &active)
	if err != nil {
		log.Fatalf("error in cleanup: %v", err)
	}
	err = writeOutput(&active)
	if err != nil {
		log.Fatalf("error writing output: %v", err)
	}
	return nil
}

func cleanup(p string, dur time.Duration, active *pathinfo) error {

	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	abs += string(os.PathSeparator)
	// Walk the directory tree
	err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			err = removeEmptyPath(path)
			if err != nil {
				log.Printf("could not remove dir %s: %v", path, err)
			}
			return nil
		}

		// Check if the file extension is .txt
		if strings.ToLower(filepath.Ext(path)) == ".txt" {

			// // Check if the file should be excluded
			// if contains(exclude, filepath.Base(path)) {
			// 	fmt.Println("Skipping:", path)
			// 	return nil
			// }

			// Check if the file was modified within the duration given
			modTime := info.ModTime()
			if time.Since(modTime) < dur {
				if opts.Debug {
					log.Println("Skipping:", path)
				}
				// active = append(active, pathinfo{path, modTime, make([]pathinfo, 0)})
				folderpath, err := filepath.Abs(filepath.Dir(path))
				if err != nil {
					return err
				}
				folderpath = strings.ReplaceAll(folderpath, abs, "")
				folders := strings.Split(folderpath, string(os.PathSeparator))

				cur := active
				for _, f := range folders {
					if _, ok := cur.Children[f]; !ok {
						cur.Children[f] = &pathinfo{Path: f, Children: map[string]*pathinfo{}, IsDir: true}
					}
					cur = cur.Children[f]
				}
				base := filepath.Base(path)
				cur.Children[base] = &pathinfo{Path: base, Modified: modTime, IsFile: true}
				return nil
			}

			// Delete the file
			err := os.Remove(path)
			if err != nil {
				return err
			}
			log.Println("Deleted:", path)
			//delete folder if empty
			err = removeEmptyPath(filepath.Dir(path))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func removeEmptyPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("provided path is not a directory")
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		err := os.Remove(path)
		if err != nil {
			return err
		}
		log.Println("Deleted:", path)
	} else {
		return nil
	}

	return nil
}
