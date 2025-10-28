package common

import (
	"bytes"
	// "fmt"
	"strings"
	"testing"
	"text/tabwriter"
)

var TabWriterTests = []struct {
	in       string
	exp      string
	rightExp string
	flags    uint
	indent   string
}{
	{in: "",
		exp:      "",
		rightExp: "",
	},

	{in: "\n",
		exp:      "\n",
		rightExp: "\n",
	},
	{in: "\t\n",
		exp:      "\n",
		rightExp: "\n",
	},
	{in: "\t\t\n",
		exp:      "\n",
		rightExp: "\n",
	},
	{in: "x\t\n",
		exp:      "x\n",
		rightExp: ".x\n",
	},
	{in: "123\t4567\t\n",
		exp:      "123.4567\n",
		rightExp: ".123.4567\n",
	},
	{in: "123\t4567\t\n" +
		"12\t45\t\n",
		exp:      "123.4567\n12..45\n",
		rightExp: ".123.4567\n..12...45\n",
	},
	{in: "123\t4567\t\n" +
		"12\t45\t",
		exp: "123.4567\n" +
			"12..45",
		rightExp: ".123.4567\n" +
			"..1245",
	},
	{
		in: "11\t\n" +
			"123\t456\t\n" +
			"1\t\n" +
			"2\t3\t4\t\n",
		exp: "11\n" +
			"123.456\n" +
			"1\n" +
			"2...3...4\n",
		rightExp: "..11\n" +
			".123.456\n" +
			"...1\n" +
			"...2...3.4\n",
	},
	{in: "123\t4567\t\n" +
		"1234\t\t123\t\n" +
		"12\t45\t\n",
		exp: "123..4567\n" +
			"1234......123\n" +
			"12...45\n",
		rightExp: "..123.4567\n" +
			".1234......123\n" +
			"...12...45\n",
	},
	{in: "1234\t4321\t\n" +
		"\t32\t\n" +
		"23\t\n" +
		"1\t\n" +
		"\t33\t\n" +
		"1\t",
		exp: "1234.4321\n" +
			".....32\n" +
			"23\n" +
			"1\n" +
			".....33\n" +
			"1",
		rightExp: ".1234.4321\n" +
			"........32\n" +
			"...23\n" +
			"....1\n" +
			"........33\n" +
			"1",
	},
	{in: "1234\t4321\t\n" +
		"1234567890123\n" +
		"33\t44\t\n" +
		"66\t123456789012\n" +
		"\t55\t\n",
		exp: "1234.4321\n" +
			"1234567890123\n" +
			"33...44\n" +
			"66...123456789012\n" +
			".....55\n",
		rightExp: ".1234.4321\n" +
			"1234567890123\n" +
			"...33...44\n" +
			"...66.123456789012\n" +
			"........55\n",
	},
	{in: "c1\tc2\tc3\t\n" +
		"c1\tc2\tc333\t\n" +
		"123456789012\n" +
		"1\n" +
		"1\t23456789012\n" +
		"\n" +
		"\t\n" +
		"t\t\n" +
		"d2345\t1234\t\n" +
		"d2345\t1234\tccccccc\t\n",
		exp: "c1....c2...c3\n" +
			"c1....c2...c333\n" +
			"123456789012\n" +
			"1\n" +
			"1.....23456789012\n" +
			"\n" +
			"\n" +
			"t\n" +
			"d2345.1234\n" +
			"d2345.1234.ccccccc\n",
		rightExp: "....c1...c2......c3\n" +
			"....c1...c2....c333\n" +
			"123456789012\n" +
			"1\n" +
			".....1.23456789012\n" +
			"\n" +
			"\n" +
			".....t\n" +
			".d2345.1234\n" +
			".d2345.1234.ccccccc\n",
	},
}

func TestTabWriter(t *testing.T) {
	for _, test := range TabWriterTests {
		buf := &bytes.Buffer{}

		w := NewTabWriter(buf, []byte(test.indent), 0, 8, 1, '.', test.flags)
		w.Write([]byte(test.in))
		w.Flush()

		niceIn := strings.ReplaceAll(strings.ReplaceAll(test.in, "\t", "\\t"),
			"\n", "\\n\n")

		if buf.String() != test.exp {
			goBuf := &bytes.Buffer{}
			goW := tabwriter.NewWriter(goBuf, 0, 8, 1, '.', test.flags)
			goW.Write([]byte(test.in))
			goW.Flush()

			t.Errorf("INPUT - FORMATTED:\n%s\n", niceIn)
			t.Errorf("Right: false\n")
			t.Errorf("GO RETURNS - RAW:\n%#v\n", goBuf.String())
			t.Errorf("GO RETURNS:\n%s\n", goBuf.String())
			t.Errorf("Test - RAW:\nExp:\n%#v\nGot:\n%#v",
				test.exp, buf.String())
			t.Fatalf("Test:\nExp:\n%s\nGot:\n%s",
				test.exp, buf.String())
		}

		// RIGHT ALIGN

		buf.Reset()
		// fmt.Printf("----- NEW -----\n")
		w = NewTabWriter(buf, []byte(test.indent), 0, 8, 1, '.',
			test.flags|AlignRight)
		w.Write([]byte(test.in))
		w.Flush()

		if buf.String() != test.rightExp {
			goBuf := &bytes.Buffer{}
			// fmt.Printf("------ GO -----\n")
			goW := tabwriter.NewWriter(goBuf, 0, 8, 1, '.',
				test.flags|tabwriter.AlignRight)
			goW.Write([]byte(test.in))
			goW.Flush()

			t.Errorf("INPUT - FORMATTED:\n%s\n", niceIn)
			t.Errorf("Right: true\n")
			t.Errorf("GO RETURNS - RAW:\n%#v\n", goBuf.String())
			t.Errorf("GO RETURNS:\n%s\n", goBuf.String())
			t.Errorf("Test - RAW:\nExp:\n%#v\nGot:\n%#v",
				test.rightExp, buf.String())
			t.Fatalf("Test:\nExp:\n%s\nGot:\n%s",
				test.rightExp, buf.String())
		}
	}
}
