package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var depth = 0
var ConfigFile = EnvString("XR_CONFORM_CONFIG", "")
var ShowLogs = EnvBool("XR_SHOWLOGS", false)

func conformFunc(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		Error("No arguments allowed for this command")
	}

	reg, xErr := xrlib.GetRegistry(GetServer())
	Error(xErr)

	if ConfigFile != "" {
		Error(reg.LoadConfigFromFile(ConfigFile))
	}

	td := NewTD(GetServer())
	td.SetRegistry(reg)

	FailFast = false
	td.Run(TestRegistry)

	// td.Dump("")
	if depth <= 0 { // == 0 || depth == -1 {
		// Can't actually do zero, so zero = -1 (all)
		depth = 9999999
	} else {
		// depth = depth + 1
	}
	td.Print(os.Stdout, "", ShowLogs, depth-1)

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
	conformCmd.Flags().BoolVarP(&ShowLogs, "logs", "l", ShowLogs,
		"Show logs even on success")
	conformCmd.Flags().IntVarP(&depth, "depth", "d", depth, "Console depth")
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
	AARRAY      // []
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
	AARRAY       // []
	NUM         // 0-9

	return tokens, nil
}
*/
