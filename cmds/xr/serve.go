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

	serveCmd.Flags().StringP("address", "a", "0.0.0.0:8080",
		"address:port of listener")

	parent.AddCommand(serveCmd)
}

func serveFunc(cmd *cobra.Command, args []string) {
	address, _ := cmd.Flags().GetString("address")

	if len(args) != 1 {
		Error("Command requires exactly one arg, the path to a directory")
	}
	dir := strings.TrimRight(args[0], "/")

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			Error(NewXRError("not_found", dir, dir).
				SetDetailf("Directory %q doesn't not exist", dir))
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
		Verbose("405 %s %s", r.Method, file)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if strings.Index(file, "..") >= 0 {
		Verbose("400 %s %s", r.Method, file)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file = dir + "/" + file
	info, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			Verbose("404 %s %s", r.Method, file)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		Verbose("400 %s %s", r.Method, file)
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
		Verbose("400 %s %s", r.Method, file)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
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

	Verbose("200 %s %s", r.Method, origFile)
	w.Write(buf)
}
