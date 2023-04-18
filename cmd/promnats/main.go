package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kmpm/promnats.go"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
)

type Opts struct {
	Timeout time.Duration
	Trace   bool
	Debug   bool
	Dest    string
}

var (
	opts Opts = Opts{
		Timeout: 10 * time.Second,
	}
)

func main() {
	contextFlag := flag.String("context", "", "context")
	serverFlag := flag.String("server", "", "server like "+nats.DefaultURL)
	flag.BoolVar(&opts.Trace, "trace", false, "show trace")
	flag.BoolVar(&opts.Debug, "debug", false, "show debug")
	flag.StringVar(&opts.Dest, "dest", "./metrics", "folder to write output to")

	flag.Parse()

	if opts.Debug {
		log.Printf("  opt.Timeout: %v\n", opts.Timeout)
		log.Printf("  Context: %s", *contextFlag)
		log.Printf("  Args: %v", flag.Args())
	}

	if flag.NArg() != 1 {
		log.Fatalln("you must provide 1 argument as base subject")

	}
	var nc *nats.Conn
	var err error

	if *serverFlag == "" {
		nc, err = natscontext.Connect(*contextFlag)
		if err != nil {
			log.Fatalf("could not connect to context: %v", err)
		}
	} else {
		nc, err = nats.Connect(*serverFlag)
		log.Fatalf("could not connect to server: %v", err)
	}

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
		if len(parts) < 3 {
			log.Printf("id must have at least 3 parts: %s", id)
			continue
		}

		p := path.Join(opts.Dest, parts[0])

		err = os.MkdirAll(p, os.ModePerm)
		if err != nil {
			fmt.Printf("ERR cannot create dir %s: %v", p, err)
			continue
		}

		filename := path.Join(p, parts[1]+".txt")
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
	active, err := cleanup(opts.Dest, time.Hour)
	if err != nil {
		log.Fatalf("error in cleanup: %v", err)
	}

	meta := map[string][]string{}

	for _, x := range active {
		parts := strings.Split(x, string(os.PathSeparator))
		if v, ok := meta[parts[1]]; ok {
			meta[parts[1]] = append(v, parts[1])
		} else {
			meta[parts[1]] = []string{parts[1]}
		}
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		log.Fatalf("could not marshal keepers %v", err)
	}
	err = os.WriteFile(path.Join(opts.Dest, "metrics.json"), data, os.ModePerm)
	if err != nil {
		log.Fatalf("could not write metrics.json: %v", err)
	}

}

func cleanup(p string, dur time.Duration) (active []string, err error) {
	// Walk the directory tree
	err = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
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
				active = append(active, path)
				return nil
			}

			// Delete the file
			err := os.Remove(path)
			if err != nil {
				return err
			}
			log.Println("Deleted:", path)
		}
		return nil
	})
	return
}

func doReqAsync(ctx context.Context, req any, subj string, waitFor int, nc *nats.Conn, cb func(*nats.Msg)) error {
	jreq := []byte("{}")
	var err error

	if req != nil {
		jreq, err = json.MarshalIndent(req, "", "  ")
		if err != nil {
			return err
		}
	}

	if opts.Trace {
		log.Printf(">>> %s: %s\n", subj, string(jreq))
	}

	var (
		mu  sync.Mutex
		ctr = 0
	)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var finisher *time.Timer
	if waitFor == 0 {
		finisher = time.NewTimer(300 * time.Millisecond)
		go func() {
			select {
			case <-finisher.C:
				cancel()
			case <-ctx.Done():
				return
			}
		}()
	}

	errs := make(chan error)
	sub, err := nc.Subscribe(nc.NewRespInbox(), func(m *nats.Msg) {
		mu.Lock()
		defer mu.Unlock()

		if opts.Trace {
			if m.Header != nil {
				log.Printf("<<< Header: %+v", m.Header)
			}
		}

		if finisher != nil {
			finisher.Reset(300 * time.Millisecond)
		}

		if m.Header.Get("Status") == "503" {
			errs <- nats.ErrNoResponders
			return
		}

		cb(m)
		ctr++

		if waitFor > 0 && ctr == waitFor {
			cancel()
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	if waitFor > 0 {
		sub.AutoUnsubscribe(waitFor)
	}

	msg := nats.NewMsg(subj)
	msg.Data = jreq
	msg.Reply = sub.Subject
	msg.Header.Add("Accept", "text/html")

	err = nc.PublishMsg(msg)
	if err != nil {
		return err
	}

	select {
	case err = <-errs:
		if err == nats.ErrNoResponders && strings.HasPrefix(subj, "$SYS") {
			return fmt.Errorf("server request failed, ensure the account used has system privileges and appropriate permissions")
		}

		return err
	case <-ctx.Done():
	}

	if opts.Trace {
		log.Printf("=== Received %d responses", ctr)
	}

	return nil
}

func doReq(ctx context.Context, req any, subj string, waitFor int, nc *nats.Conn) ([]*nats.Msg, error) {
	res := []*nats.Msg{}
	mu := sync.Mutex{}

	err := doReqAsync(ctx, req, subj, waitFor, nc, func(m *nats.Msg) {
		mu.Lock()
		res = append(res, m)
		mu.Unlock()
	})

	return res, err
}
