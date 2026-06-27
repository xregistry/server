package main

import (
	// "fmt"
	"strings"

	"github.com/spf13/cobra"
)

type Operation struct {
	Action string
	Value  string
}

// Extract the operations/flags from the command line and preserve the order
func GetArgOrder(cmd *cobra.Command, osArgs []string) []Operation {
	var ops []Operation

	// Skip cmd name
	for i := 1; i < len(osArgs); i++ {
		arg := osArgs[i]

		action := ""
		if strings.HasPrefix(arg, "--") {
			action, _, _ = strings.Cut(arg[2:], "=")
		}
		if action == "" {
			continue
		}

		// Handle both: --flag=value && --flag value
		if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
			arg = arg[eqIdx+1:]
		} else {
			i++
			if i >= len(osArgs) {
				continue // should never happen, Cobra should catch it
			}
			arg = osArgs[i]
		}

		ops = append(ops, Operation{
			Action: action,
			Value:  arg,
		})
	}

	return ops
}
