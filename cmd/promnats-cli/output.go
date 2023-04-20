package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"os"
	"path"
)

const templateIndex = "templates/index.html"
const templateLayout = "templates/layout.html"

var (
	//go:embed templates/layout.html
	baseLayoutFS embed.FS

	//go:embed templates/index.html
	indexTemplateFS embed.FS

	indexTemplate *template.Template
)

func init() {
	baseLayout := template.Must(template.New("layout").ParseFS(baseLayoutFS, templateLayout))
	indexTemplate = template.Must(baseLayout.ParseFS(indexTemplateFS, templateIndex))
}

type outputData struct {
	*pathinfo
	Parent string
	Links  []string
}

func writeOutputHTML(filename string, here *outputData) error {
	var buf bytes.Buffer
	err := indexTemplate.ExecuteTemplate(&buf, "layout", here)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, buf.Bytes(), os.ModePerm)
	return err
}

func writeOutput(active *pathinfo) error {
	data, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		// log.Fatalf("could not marshal keepers %v", err)
		return err
	}
	err = os.WriteFile(path.Join(opts.Dest, "active.json"), data, os.ModePerm)
	if err != nil {
		// log.Fatalf("could not write metrics.json: %v", err)
		return err
	}
	if opts.WithHTML {
		err = writeOutputHTML(path.Join(opts.Dest, "index.html"), &outputData{Parent: "", pathinfo: active, Links: []string{"./active.json"}})
		if err != nil {
			return err
		}
		err = writeChildHTML(opts.Dest, active)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeChildHTML(basepath string, parent *pathinfo) error {
	for n, c := range parent.Children {
		thispath := path.Join(basepath, n)
		if c.IsDir {
			out := &outputData{Parent: parent.Path, pathinfo: c}
			err := writeOutputHTML(path.Join(thispath, "index.html"), out)
			if err != nil {
				return err
			}
			if len(c.Children) > 0 {
				err = writeChildHTML(thispath, c)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
