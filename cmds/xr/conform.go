package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
)

func TestAll(td *TD) {
	td.DependsOn(TestSniffTest)
	td.Run(TestLoadModel)
	td.Run(TestRoot)
}

func TestAll2(td *TD) {
	td.Run(TestGroups)
}

func TestGroups(td *TD) {
}

var depthCount = 0
var ConfigFile = EnvString("XR_CONFORM_CONFIG", "")
var ShowLogs = EnvBool("XR_SHOWLOGS", false)

func EnvBool(name string, def bool) bool {
	val := os.Getenv(name)
	if val != "" {
		def = strings.EqualFold(val, "true")
	}
	return def
}

func EnvString(name string, def string) string {
	val := os.Getenv(name)
	if val != "" {
		def = val
	}
	return def
}

func conformFunc(cmd *cobra.Command, args []string) {
	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	if ConfigFile != "" {
		Error(reg.LoadConfigFromFile(ConfigFile))
	}

	td := NewTD(Server)
	td.Props["xreg"] = reg

	FailFast = false
	td.Run(TestAll)
	td.Run(TestAll2)

	// td.Dump("")
	if depthCount <= 0 { // == 0 || depthCount == -1 {
		// Can't actually do zero, so zero = -1 (all)
		depthCount = 9999999
	} else {
		depthCount = depthCount + 1
	}
	td.Print(os.Stdout, "", ShowLogs, depthCount)

	if td.ExitCode() != 0 {
		os.Exit(td.ExitCode())
	}
}

func addConformCmd(parent *cobra.Command) {
	conformCmd := &cobra.Command{
		Use:     "conform",
		Short:   "xRegistry Conformance Tester",
		Run:     conformFunc,
		GroupID: "Admin",
	}
	conformCmd.Flags().StringVarP(&ConfigFile, "config", "c", ConfigFile,
		"Location of config file")
	conformCmd.Flags().BoolVarP(&ShowLogs, "logs", "l", ShowLogs,
		"Show logs on success")
	conformCmd.Flags().CountVarP(&depthCount, "depth", "d", "Console depth")
	conformCmd.Flags().BoolVarP(&tdDebug, "tdDebug", "t", tdDebug, "td debug")

	parent.AddCommand(conformCmd)
}

/*
type JSON map[string]any

func GetStack() []string {
	stack := []string{}

	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		if line == 0 {
			break
		}
		stack = append(stack,
			fmt.Sprintf("%s - %s:%d",
				path.Base(runtime.FuncForPC(pc).Name()), path.Base(file), line))
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
	return stack
}

func ShowStack() {
	stack := GetStack()
	fmt.Println("----- Stack")
	for _, line := range stack {
		fmt.Println(line)
	}
}

func ToJSON(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error Marshaling: %s", err))
	}
	return string(buf)
}
*/

type XRegistry struct {
	// Config values:
	//   server.url: VALUE
	//   header.NAME: VALUE
	Config map[string]string
}

func NewXRegistry() (*XRegistry, error) {
	xreg := &XRegistry{}
	return xreg, xreg.LoadConfig("")
}

func NewXRegistryWithConfigPath(path string) (*XRegistry, error) {
	xreg := &XRegistry{}
	return xreg, xreg.LoadConfig(path)
}

func (xr *XRegistry) GetServerURL() string {
	return xr.GetConfig("server.url")
}

func (xr *XRegistry) LoadConfig(path string) error {
	err := xr.LoadConfigFromFile(path)
	if err != nil {
		return err
	}
	return nil
}

// File syntax:
// prop: value
// # comment
func (xr *XRegistry) LoadConfigFromFile(filename string) error {
	if filename == "" {
		filename = "xr.config"
	}
	buf, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return xr.LoadConfigFromBuffer(string(buf))
}

// Buffer syntax:
// prop: value
// # comment
func (xr *XRegistry) LoadConfigFromBuffer(buffer string) error {
	lines := strings.Split(buffer, "/n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		if name == "" {
			return fmt.Errorf("Error in config data - no name: %q", line)
		}
		xr.SetConfig(name, value)
	}
	return nil
}

func (xr *XRegistry) GetConfig(name string) string {
	if xr.Config == nil {
		return ""
	}
	return xr.Config[name]
}

func (xr *XRegistry) SetConfig(name string, value string) error {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	if name == "" {
		return fmt.Errorf("Config name can't be blank")
	}
	if value == "" {
		delete(xr.Config, name)
	} else {
		if xr.Config == nil {
			xr.Config = map[string]string{}
		}
		xr.Config[name] = value
	}
	return nil
}

type HTTPResponse struct {
	Error      error
	StatusCode int
	Headers    http.Header
	Body       []byte
	JSON       map[string]any
}

func (xr *XRegistry) Get(path string) *HTTPResponse {
	return xr.CurlWithHeaders("GET", path, nil, "")
}

func (xr *XRegistry) Put(path string, body string) *HTTPResponse {
	return xr.CurlWithHeaders("PUT", path, nil, body)
}

func (xr *XRegistry) Post(path string, body string) *HTTPResponse {
	return xr.CurlWithHeaders("POST", path, nil, body)
}

func (xr *XRegistry) Patch(path string, body string) *HTTPResponse {
	return xr.CurlWithHeaders("PATCH", path, nil, body)
}

func (xr *XRegistry) Curl(verb string, path string, body string) *HTTPResponse {
	return xr.CurlWithHeaders(verb, path, nil, body)
}

// HTTPResponse
// golang error if things failed at the tranport level
func (xr *XRegistry) CurlWithHeaders(verb string, path string, headers map[string]string, body string) *HTTPResponse {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	bodyReader := io.Reader(nil)
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequest(verb, xr.GetServerURL()+"/"+path, bodyReader)
	if err != nil {
		return &HTTPResponse{Error: err}
	}

	for key, value := range xr.Config {
		key = strings.TrimSpace(key[7:])
		if !strings.HasSuffix(key, "header.") {
			continue
		}
		key = strings.TrimSpace(key[7:])
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		req.Header.Add(key, value) // ok even if value is ""
	}

	for key, value := range headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		req.Header.Add(key, value)
	}

	doRes, err := client.Do(req)
	if err != nil || doRes == nil {
		return &HTTPResponse{Error: err}
	}
	defer doRes.Body.Close()

	res := &HTTPResponse{
		StatusCode: doRes.StatusCode,
		Headers:    doRes.Header.Clone(),
	}
	res.Body, err = io.ReadAll(doRes.Body)
	if err != nil {
		return &HTTPResponse{Error: err}
	}

	if len(res.Body) > 0 {
		// Ignore any error, just assume it's not JSON and keep going
		json.Unmarshal(res.Body, &res.JSON)
	}

	return res
}

/*
func (j *JSON) JPath(path string) any {
	tokens, err := TokenizeJPath(path)
	if err != nil {
		panic(fmt.Sprintf("Bad jpath: %q, %s", path, err))
	}
	if len(tokens) == 0 {
		return nil
	}
	return nil
}
*/

const (
	NAME        = iota + 1
	ROOT        // $
	THIS        // @
	CHILD       // .
	DESCENDANTS // ..
	WILDCARD    // *
	ARRAY       // []
	NUM         // 0-9
)

type Token struct {
	kind  int
	value string
}

/*
func TokenizeJPath(path string) ([]*Token, error) {
	word := ""
	tokens := []Token(nil)

	CalcGroup := func(ch byte) int {
		switch ch {
		case '.':
			return 0
		case '@':
			return 1
		case '$':
			return 2
		case '[':
			return 3
		case ']':
			return 4
		case '*':
			return 5
		case '\'':
			return 6
		}
		if ch >= '0' && ch <= '9' {
			return 7
		}
		if (ch == '_') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			return 8
		}
		panic("what?" + string(ch))
	}

	// actionS + nextState
	// 1:bldWord, 2:endWord, 3:startQuote, 4:endQuote, 5:endRoot, 6:endThis
	// 7:endChild, 8:endDesc, 9: endWild, A:endArray, B: end
	//    .   @   $   [   ]   *  '   09  _az
	stateTable := [][]int{
		{}, // Just so we don't use 0
		{0, 0},
	}

	state := 1
	for i := 0; i < len(path); i++ {
		ch := path[i]
		actions := stateTable[state][CalcGroup(ch)]
		state = actions % 10

		for actions = actions / 10; actions != 0; actions = actions / 10 {
			switch actions % 10 {
			case 1:
				word += string(ch)
			case 2:
				tokens = append(tokens, &Token{NAME, word})
				word = ""
			case 3: //
			case 4:
				word += string(ch)
			case 5:
				tokens = append(tokens, &Token{NAME, word})
				word = ""
			case 6:
				word += string(ch)
			case 7:
				tokens = append(tokens, &Token{NUM, word})
				word = ""
			}
		}
	}
	DESCENDANTS // ..
	WILDCARD    // *
	ARRAY       // []
	NUM         // 0-9

	return tokens, nil
}
*/
