package registry

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// HTML, # of expands
func RegHTMLify(buf []byte, proxyHost, targetHost string) ([]byte, int) {
	// Escape < and >
	buf = []byte(strings.ReplaceAll(string(buf), "<", "&lt;"))
	buf = []byte(strings.ReplaceAll(string(buf), ">", "&gt;"))

	str := fmt.Sprintf(`"(https?://[^?"\n]*)(\??)([^"\n]*)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href='$1?ui&$3'>$1$2$3</a>"`)
	buf = re.ReplaceAll(buf, []byte(repl))

	if targetHost != "" {
		matchHost := targetHost

		// Allow for http or https in the output
		if strings.HasPrefix(matchHost, "http") {
			_, matchHost, _ = strings.Cut(matchHost, "://")
		}
		matchHost = `https?://` + matchHost

		str := fmt.Sprintf(`<a href='%s([^?']*)\?ui`, matchHost)
		re := regexp.MustCompile(str)
		repl := fmt.Sprintf(`<a href='%s/proxy?host=%s&path=$1`,
			proxyHost, targetHost)
		buf = re.ReplaceAll(buf, []byte(repl))
	}

	res := new(bytes.Buffer)

	// Now add the toggle (expand) stuff for the JSON nested entities

	// remove trailing \n so we don't have an extra line for the next stuff
	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}

	count := 0
	for _, line := range strings.Split(string(buf), "\n") {
		spaces := "" // leading spaces
		numSpaces := 0
		first := rune(0) // first non-space char
		last := rune(0)  // last non-space char
		decDepth := false

		for _, ch := range line {
			if first == 0 { // doing spaces
				if ch == ' ' {
					numSpaces++
					spaces += " "
					continue
				}
			}
			if ch != ' ' {
				if first == 0 {
					first = rune(ch)
				}
				last = rune(ch)
			}
		}
		line = line[numSpaces:] // Remove leading spaces

		decDepth = (first == '}' || first == ']')
		incDepth := (last == '{' || last == '[')

		// btn is the special first column of the output, non-selectable
		btn := "<span class=spc> </span>" // default: non-selectable space

		if incDepth {
			// Build the 'expand' toggle char
			count++
			exp := fmt.Sprintf("<span class=exp id='s%d' "+
				"onclick='toggleExp(this)'>"+HTML_EXP+"</span>", count)

			if numSpaces == 0 {
				// OLD - if we want toggle at the root too
				// Use the special first column for it
				// btn = exp
				// end OLD

				// Don't show toggle at root.
				// To ensure count starts at 1 for the real first toggle
				// just decrement it here
				count--
			} else {
				// Replace the last space with the toggle.
				// Add a nearly-hidden space so when people copy the text it
				// won't be missing a space due to the toggle
				spaces = spaces[:numSpaces-1] + exp +
					"<span class=hide > </span>"
			}
		}

		res.WriteString(btn)    // special first column
		res.WriteString(spaces) // spaces + 'exp' if needed
		if decDepth {
			// End the block before the tailing "}" or "]"
			res.WriteString("</span>") // block
		}
		res.WriteString(line)
		if incDepth {
			// write the "..." and then the <span> for the toggle text
			res.WriteString(fmt.Sprintf("<span style='display:none;cursor:pointer' "+
				"id='s%ddots' onclick='toggleExp(s%d)'>...</span>", count, count))
			res.WriteString(fmt.Sprintf("<span id='s%dblock'>", count))
		}

		res.WriteString("\n")
	}

	return res.Bytes(), count
}
