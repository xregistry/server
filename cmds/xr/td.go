package main

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

var PASS = 1
var FAIL = 2
var WARN = 3
var SKIP = 4
var LOG = 5 // Only shown on failure or when they ask to see all logs
var MSG = 6 // Like LOG but will always be printed

var StatusText = []string{"", "PASS", "FAIL", "WARN", "SKIP", "LOG", "MSG"}

var ConsoleDepth = -1
var LogfileDepth = -1
var FailFast = false
var IgnoreWarn = true
var TestsRun = map[string]*TD{}

type TestFn func(td *TD)

func (fn TestFn) Name() string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	before, name, _ := strings.Cut(name, ".")
	if name == "" {
		name = before
	}
	return name
}

type LogEntry struct {
	Date    time.Time
	Type    int // pass, fail, warning, skip, else log or TD
	Text    string
	Subtest *TD
}

// TestData
type TD struct {
	TestName string
	Parent   *TD `json:"-"`
	Logs     []*LogEntry

	Status int // PASS, FAIL, ...
	Props  map[string]any

	NumPass int // These will include the status of _this_ TD and its children
	NumFail int
	NumWarn int
	NumSkip int
}

func NewTD(name string, parent ...*TD) *TD {
	p := (*TD)(nil)
	if len(parent) > 0 {
		if len(parent) > 1 {
			panic("too many parents")
		}
		p = parent[0]
	}

	newTD := &TD{
		TestName: name,
		Parent:   p,
		Logs:     []*LogEntry{},

		Status:  PASS,
		Props:   map[string]any{},
		NumPass: 1,
	}

	if p != nil {
		newLE := &LogEntry{
			Date:    time.Now(),
			Type:    0,
			Text:    "",
			Subtest: newTD,
		}
		p.Logs = append(p.Logs, newLE)
		p.AddStatus(PASS)

		newTD.Props = p.Props
	}

	return newTD
}

func (td *TD) ExitCode() int {
	if td.Status == PASS || td.Status == SKIP ||
		(IgnoreWarn && td.Status == WARN) {
		return 0
	}
	if td.Status == 0 {
		return 255
	}
	return td.Status
}

func (td *TD) Dump(indent string) {
	fmt.Printf("%sTestName: %s\n", indent, td.TestName)
	if len(td.Logs) > 0 {
		fmt.Printf("%s  Logs:\n", indent)
		for _, le := range td.Logs {
			if le.Subtest != nil {
				fmt.Printf("%s    %s", indent, le.Date.Format(time.RFC3339))
				le.Subtest.Dump(indent + "    ")
			} else {
				fmt.Printf("%s    %s (%s) %q\n", indent,
					le.Date.Format(time.RFC3339), StatusText[le.Type],
					le.Text)
			}
		}
	}
}

func (td *TD) Print(out io.Writer, indent string, showLogs bool, depth int) {
	if depth != 0 || td.Status == FAIL {
		td.write(out, indent, showLogs, depth)
		fmt.Printf("\n"+indent+"Pass: %d   Fail: %d   Warn: %d   Skip: %d\n",
			td.NumPass, td.NumFail, td.NumWarn, td.NumSkip)
	}
}

var tdDebug = false

func Debug(out io.Writer, fmtStr string, args ...any) {
	if tdDebug {
		str := fmt.Sprintf(fmtStr, args...)
		if str[len(str)-1] != '\n' {
			str += "\n"
		}
		out.Write([]byte(str))
	}
}

func (td *TD) write(out io.Writer, indent string, showLogs bool, depth int) {
	if depth != 0 || td.Status == FAIL {
		td.writeHeader(out, indent, showLogs, depth)
		nextDepth := depth
		if depth > 0 {
			nextDepth = nextDepth - 1
		}
		td.writeBody(out, indent, showLogs, nextDepth)
	}
}

func (td *TD) writeHeader(out io.Writer, indent string, showLogs bool, depth int) {
	if depth != 0 || td.Status == FAIL {
		str := indent + StatusText[td.Status] + ": "

		if tdDebug {
			td.TestName += fmt.Sprintf(" (hd: %d", depth)
		}
		out.Write([]byte(str + td.TestName + "\n"))

		// Debug(out,"%s%s %d/%d/%d/%d",
		// str, td.TestName, td.NumPass, td.NumFail, td.NumWarn, td.NumSkip)
	}
}

func (td *TD) writeBody(out io.Writer, indent string, showLogs bool, depth int) {
	if depth == 0 && td.Status != FAIL {
		return
	}
	saveIndent := indent
	endSaveIndent := strings.ReplaceAll(saveIndent, "├─", "│ ")

	// Calc the last logEntry - skipping the LOG messages at the end
	lastLog := len(td.Logs) - 1
	Debug(out, "%sstat:%s sl:%v", indent, StatusText[td.Status], showLogs)
	// if td.Status != FAIL && showLogs == false {
	if showLogs == false {
		for ; lastLog > 0; lastLog-- {
			log := td.Logs[lastLog]
			if log.Type == LOG {
				Debug(out, "%s  dropping  le:%s", indent, log.Text)
				continue
			}

			if (log.Type == FAIL || (log.Subtest != nil && log.Subtest.Status == FAIL)) || depth > 0 {
				break
			}
			Debug(out, "%s  dropping  le:%s", indent, log.Text)
		}
	}
	Debug(out, "%sLL:%d len:%d depth:%d", indent, lastLog, len(td.Logs), depth)

	for i := 0; i <= lastLog; i++ {
		le := td.Logs[i]
		Debug(out, "%sprocessing %s l#%d depth:%d", indent, le.Text, i, depth)
		str := ""
		// date := le.Date.Format(time.RFC3339)
		if i == lastLog {
			indent = endSaveIndent + "└─ "
		} else if i < lastLog {
			indent = endSaveIndent + "├─ "
		}

		if tdDebug {
			le.Text += fmt.Sprintf(" (bd: %d ll:%d i:%d", depth, lastLog, i)
		}
		if le.Type > 0 && le.Type < LOG {
			str = PrettyPrint(indent, StatusText[le.Type]+": ", le.Text)
		} else if le.Subtest == nil { // log or msg
			// Show logs it's a MSG, they asked for all logs, or TD=FAIL
			if le.Type == MSG || showLogs || td.Status == FAIL {
				str = PrettyPrint(indent, "", le.Text)
			}
		} else { // subtest
			nextDepth := depth
			if depth > 0 {
				nextDepth = nextDepth - 1
			}
			if i == lastLog {
				le.Subtest.writeHeader(out, endSaveIndent+"└─ ", showLogs, depth)
				le.Subtest.writeBody(out, saveIndent+"   ", showLogs, nextDepth)
			} else /* if i < lastLog */ {
				le.Subtest.write(out, indent, showLogs, nextDepth)
			}
		}
		out.Write([]byte(str))
	}
}

func (td *TD) AddStatus(status int) {
	if status == 0 || status >= LOG {
		return
	}
	/*
		fmt.Printf("  %q before status: %s", td.TestName, StatusText[td.Status])
		fmt.Printf(" ( %d / %d / %d / %d\n", td.NumPass, td.NumFail,
		td.NumWarn, td.NumSkip)
	*/
	switch td.Status {
	case 0, PASS:
		td.Status = status
	case WARN:
		switch status {
		case FAIL:
			td.Status = status
		}
	case SKIP:
		switch status {
		case FAIL:
			td.Status = status
		case WARN:
			td.Status = status
		}
	}

	// Recalc totals, up the chain
	for p := td; p != nil; p = p.Parent {
		// fmt.Printf("    Recalc'ing: %s\n", p.TestName)
		sums := [4]int{0, 0, 0, 0}
		sums[p.Status-1] = 1
		for _, le := range p.Logs {
			if le.Subtest != nil {
				sums[0] += le.Subtest.NumPass
				sums[1] += le.Subtest.NumFail
				sums[2] += le.Subtest.NumWarn
				sums[3] += le.Subtest.NumSkip
			} else if le.Type > 0 && le.Type < LOG {
				sums[le.Type-1]++
			}
		}
		p.NumPass = sums[0]
		p.NumFail = sums[1]
		p.NumWarn = sums[2]
		p.NumSkip = sums[3]
		// fmt.Printf("    After Recalc: %s - %v\n", p.TestName, sums)
	}

	// ShowStack()
	/*
		fmt.Printf("  %q after status: %s", td.TestName, StatusText[td.Status])
		fmt.Printf(" ( %d / %d / %d / %d\n", td.NumPass, td.NumFail,
		td.NumWarn, td.NumSkip)
	*/

	if td.Parent != nil {
		td.Parent.AddStatus(td.Status)
	}
}

// PASS|FAIL|WARN|SKIP, testNameText, substitute args for testName
func (td *TD) Report(status int, args ...any) {
	// fmt.Printf("Report: %q %s : %v\n", td.TestName, StatusText[status], args)

	if len(args) > 0 {
		td.Logs = append(td.Logs, &LogEntry{
			Date:    time.Now(),
			Type:    status,
			Text:    fmt.Sprintf(args[0].(string), args[1:]...),
			Subtest: nil,
		})
	} /* else {
		td.Logs = append(td.Logs, &LogEntry{
			Date:    time.Now(),
			Type:    status,
			Text:    "",
			Subtest: nil,
		})
	} */
	td.AddStatus(status)
	if FailFast && status == FAIL {
		td.Stop()
	}
}

func (td *TD) Pass(args ...any)    { td.Report(PASS, args...) }
func (td *TD) Fail(args ...any)    { td.Report(FAIL, args...) }
func (td *TD) FailNow(args ...any) { td.Report(FAIL, args...); td.Stop() }
func (td *TD) Warn(args ...any)    { td.Report(WARN, args...) }
func (td *TD) Skip(args ...any)    { td.Report(SKIP, args...) }
func (td *TD) Log(args ...any)     { td.Report(LOG, args...) }
func (td *TD) Msg(args ...any)     { td.Report(MSG, args...) }
func (td *TD) Stop()               { panic("stop") }

func (td *TD) DependsOn(fn TestFn) {
	if prevTD, ok := TestsRun[fn.Name()]; ok {
		if prevTD.Status == FAIL {
			td.FailNow("Dependency %q (cached), exiting", fn.Name())
		} else {
			td.Log("Dependency %q passed (cached)", fn.Name())
		}
	} else {
		newTD := td.Run(fn)

		if newTD.Status == FAIL {
			td.Msg("Dependency %q failed, exiting", fn.Name())
			td.Stop()
			return
		} else {
			// td.Pass("Dependency %q", fn.Name())
		}
	}
}

func (td *TD) Run(fn TestFn) *TD {
	before, name, _ := strings.Cut(fn.Name(), ".")
	if name == "" {
		name = before
	}
	newTD := NewTD(name, td)

	// Save in the cache
	TestsRun[fn.Name()] = newTD

	// Run it and catch any panic()
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Do nothing
				// Just allow the panic() caller to exit immediately
				if r != "stop" {
					panic(r)
				}
			}
		}()
		fn(newTD)
	}()

	return newTD
}

func (td *TD) Must(expr bool, args ...any) {
	if !expr {
		td.Fail(args...)
		return
	}
	td.Pass(args...)
}

func (td *TD) MustEqual(exp any, got any, args ...any) {
	if !reflect.DeepEqual(exp, got) {
		td.Log("Exp(%T): %s", exp, xrlib.ToJSON(exp))
		td.Log("Got(%T): %s", got, xrlib.ToJSON(got))
		td.Fail(args...)
		return
	}
	td.Pass(args...)
}

func (td *TD) MustNotEqual(exp any, got any, args ...any) {
	if reflect.DeepEqual(exp, got) {
		td.Log("Exp(%T): %s", exp, xrlib.ToJSON(exp))
		td.Log("Got(%T): %s", got, xrlib.ToJSON(got))
		td.Fail(args...)
		return
	}
	td.Pass(args...)
}

func (td *TD) GetProp(obj map[string]any, prop string) (any, bool, error) {
	pp, err := registry.PropPathFromUI(prop)
	if err != nil {
		td.FailNow("Error in test prep: %s(%s)", prop, err)
	}
	return registry.ObjectGetProp(obj, pp)
}

func (td *TD) NoError(err error, args ...any) {
	if err == nil {
		return
	}
	td.Fail(args...)
}

func (td *TD) NoErrorStop(err error, args ...any) {
	if err == nil {
		return
	}
	td.FailNow(args...)
}

func (td *TD) PropMustEqual(obj map[string]any, prop string, exp any) {
	pp, err := registry.PropPathFromUI(prop)
	if err != nil {
		td.FailNow("Error in test prep: %s(%s)", prop, err)
	}
	res, _, err := registry.ObjectGetProp(obj, pp)
	td.NoError(err, "Error getting prop(%s): %s", pp.UI(), err)
	daInt, err := registry.AnyToUInt(res)
	if err == nil {
		res = daInt
	}
	if !reflect.DeepEqual(exp, res) {
		td.Log("Exp(%T): %s", exp, xrlib.ToJSON(exp))
		td.Log("Got(%T): %s", res, xrlib.ToJSON(res))
		td.Fail("Wrong value for attribute %q", prop)
		return
	}
	td.Pass("%q = %q", prop, exp)
}

func (td *TD) PropMustNotEqual(obj map[string]any, prop string, exp any) {
	pp, err := registry.PropPathFromUI(prop)
	if err != nil {
		td.FailNow("Error in test prep: %s(%s)", prop, err)
	}
	res, _, err := registry.ObjectGetProp(obj, pp)
	td.NoError(err, "Error getting prop(%s): %s", pp.UI(), err)
	if registry.IsNil(res) {
		td.Fail("Attribute %q must not be null", prop)
		return
	}

	// expStr := MaxString(exp, 20)
	resStr := MaxString(res, 20)
	if reflect.DeepEqual(exp, res) {
		td.Log("Exp(%T): %s", exp, xrlib.ToJSON(exp))
		td.Log("Got(%T): %s", res, xrlib.ToJSON(res))
		td.Fail("Wrong value for attribute %q (%s)", prop, resStr)
		return
	}
	td.Pass("%q != %q (%s)", prop, exp, resStr)
}

func MaxString(val any, maxLen int) string {
	str := fmt.Sprintf("%v", val)
	if len(str) > maxLen {
		str = str[:(maxLen-3)] + "..."
	}
	return str
}

func (td *TD) PropMustExist(obj map[string]any, prop string) {
	pp, err := registry.PropPathFromUI(prop)
	if err != nil {
		td.FailNow("Error in test prep: %s(%s)", prop, err)
	}
	res, _, err := registry.ObjectGetProp(obj, pp)
	td.NoError(err, "Error getting prop(%s): %s", pp.UI(), err)
	if registry.IsNil(res) {
		td.Fail("Attribute %q must not be null", prop)
	}
	td.Pass("%q must exist", prop)
}

func (td *TD) PropMustNotExist(obj map[string]any, prop string) {
	pp, err := registry.PropPathFromUI(prop)
	if err != nil {
		td.FailNow("Error in test prep: %s(%s)", prop, err)
	}
	res, _, err := registry.ObjectGetProp(obj, pp)
	td.NoError(err, "Error getting prop(%s): %s", pp.UI(), err)
	if !registry.IsNil(res) {
		td.Fail("Attribute %q must be null", prop)
	}
	td.Pass("%q must not exist", prop)
}

func (td *TD) Should(expr bool, args ...any) {
	if !expr {
		td.Warn(args...)
	}
	td.Pass(args...)
}

func (td *TD) ShouldEqual(exp any, got any, args ...any) {
	if !reflect.DeepEqual(exp, got) {
		td.Log("Exp: %s", xrlib.ToJSON(exp))
		td.Log("Got: %s", xrlib.ToJSON(got))
		td.Warn(args...)
	}
	td.Pass(args...)
}

func PrettyPrint(indent string, prefix string, text string) string {
	rIndent := []rune(indent)                     // rune-Indent
	rPrefix := []rune(prefix)                     // rune-Prefix
	rRest := []rune(strings.TrimRight(text, " ")) // rune-Rest-of-text

	cIndent := append([]rune{}, rIndent...) // clean-Indent
	cIndent = append(cIndent, []rune(strings.Repeat(" ", len(rPrefix)))...)
	width := 79 - len(cIndent)

	// Just for first line, then use cIndent
	rIndent = append(rIndent, rPrefix...)

	str := []rune{}
	first := true

	for len(rRest) > 0 || first {
		first = false
		chopAt := 0
		lastSpace := -1
		skip := false

		for ; chopAt < len(rRest); chopAt++ {
			if rRest[chopAt] == '\n' {
				str = append(str, rIndent...)
				tmp := strings.TrimRight(string(rRest[:chopAt]), " ") + "\n"
				str = append(str, []rune(tmp)...)
				rRest = rRest[chopAt+1:]
				if len(rRest) == 0 {
					// Make sure we go thru it one more time to add blank line
					first = true
				}
				skip = true
				break
			}
			if rRest[chopAt] == ' ' {
				lastSpace = chopAt
			}

			// Normally we'd check this before the \n and ' ' but if the next
			// char (the one we want to skip on at the end of the line) is a
			// space then go ahead and break on that instead
			if chopAt+1 > width {
				break
			}
		}
		if !skip {
			if chopAt+1 > width && lastSpace >= 0 {
				chopAt = lastSpace
			}
			str = append(str, rIndent...)
			tmp := strings.TrimRight(string(rRest[:chopAt]), " ")
			str = append(str, []rune(tmp)...)
			str = append(str, rune('\n'))
			rRest = []rune(strings.TrimLeft(string(rRest[chopAt:]), " "))
		}
		rIndent = cIndent

		tmp := strings.ReplaceAll(string(rIndent), "└─", "  ")
		rIndent = []rune(strings.ReplaceAll(tmp, "├─", "│ "))
	}

	return string(str)
}
