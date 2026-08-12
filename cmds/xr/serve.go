package main

import (
	"net/http"
	"os"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	// "github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addServeCmd(parent *cobra.Command) {
	serveCmd := &cobra.Command{
		Use:     "serve DIR",
		Short:   "Run an HTTP file server for a directory",
		Run:     serveFunc,
		GroupID: "Admin",
	}

	serveCmd.Flags().BoolP("cors", "c", false,
		"Send CORS header with '*' value")
	serveCmd.Flags().StringP("address", "a", "0.0.0.0:8080",
		"address:port of listener (0.0.0.0:8080*)")
	serveCmd.Flag("address").DefValue = "" // hide default text

	parent.AddCommand(serveCmd)
}

var address string
var cors bool

func serveFunc(cmd *cobra.Command, args []string) {
	address, _ = cmd.Flags().GetString("address")
	cors, _ = cmd.Flags().GetBool("cors")

	if len(args) != 1 {
		Error("Command requires exactly one arg, the path to a directory")
	}
	dir := strings.TrimRight(args[0], "/")

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			Error(NewXRError("not_found", dir).
				SetDetailf("Directory %q doesn't not exist.", dir))
		}
		Error(err)
	}
	if !info.IsDir() {
		Error("%q must be a directory", dir)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		doit(w, r, dir)
	})

	Verbose("Listening on: %s", address)
	Error(http.ListenAndServe(address, nil))
}

func doit(w http.ResponseWriter, r *http.Request, dir string) {
	file := strings.Trim(r.URL.Path, "/")
	origFile := file

	if r.Method != "GET" {
		Verbose("405 %s /%s", r.Method, file)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.Index(file, "..") >= 0 {
		Verbose("400 %s /%s", r.Method, file)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file = dir + "/" + file
	info, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			Verbose("404 %s /%s", r.Method, file)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		Verbose("400 %s /%s", r.Method, file)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if info.IsDir() {
		file += "/index.html"
	}
	hdr := file + ".hdr"

	buf, err := os.ReadFile(file)
	if err != nil {
		Verbose("400 %s /%s", r.Method, file)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if cors {
		// w.Header().Add("Access-Control-Allow-Headers", "*")
		// w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Origin", "*")
	}

	// Default to nada
	w.Header().Add("Content-Type", "")

	mimeTypes := map[string]string{
		"html": "text/html",
		"js":   "text/javascript",
		"css":  "text/css",
		"svg":  "image/svg+xml",
		"json": "application/json",
		"xreg": "application/json",
		"png":  "image/image/png",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpegl",
		"gif":  "image/gif",
		"ico":  "image/x-icon",
	}

	if i := strings.LastIndex(file, "."); i > 0 {
		ext := file[i+1:]
		if daType := mimeTypes[ext]; daType != "" {
			w.Header().Add("Content-Type", daType)
		}
	}

	// If we have a header(hdr) file, write the HTTP headers before the body
	hdrBuf, _ := os.ReadFile(hdr)
	if len(hdrBuf) > 0 {
		lines := strings.Split(string(hdrBuf), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				w.Header().Add(parts[0], parts[1])
			}
		}
	}

	Verbose("200 %s /%s", r.Method, origFile)
	w.Write(buf)
}
