package common

import (
	"testing"
)

var PrettyJSONTests = []struct {
	in  string
	exp string
}{
	{in: ``, exp: "Err: EOF"},
	{in: `{}`, exp: "{}"},
	{in: "{   \n   }", exp: "{}"},
	{in: `{"foo":"bar"}`, exp: `{
  "foo": "bar"
}`},
	// make sure order doesn't change
	{in: `{"zoo":"zop","aoo":"aop"}`, exp: `{
  "zoo": "zop",
  "aoo": "aop"
}`},
	{in: `{"foo": {"zoo":"zop","aoo":"aop"}}`, exp: `{
  "foo": {
    "zoo": "zop",
    "aoo": "aop"
  }
}`},

	{in: `{"foo": [ {"zoo":"zop","aoo":"aop"},{"a":"b","c":"d","b":2}]}`,
		exp: `{
  "foo": [
    {
      "zoo": "zop",
      "aoo": "aop"
    },
    {
      "a": "b",
      "c": "d",
      "b": 2
    }
  ]
}`},

	{in: `{"foo": {}}`, exp: `{
  "foo": {}
}`},
	{in: `{"foo": {  }}`, exp: `{
  "foo": {}
}`},
	{in: `{"foo": {    ` + "\n" + `}}`, exp: `{
  "foo": {}
}`},

	{in: `[]`, exp: "[]"},
	{in: "[   \n   ]", exp: "[]"},
	{in: `[ "z", "a" ]`, exp: `[
  "z",
  "a"
]`},
	{in: `[ { "z":1, "a":2 } ]`, exp: `[
  {
    "z": 1,
    "a": 2
  }
]`},
}

func TestPrettyPrintJSON(t *testing.T) {
	// PrettyPrintJSON([]byte, prefix, indent) -> []byte, err
	// This will implicitly test ParseJSONToObject so I don't think we need
	// separate test for that func

	for _, test := range PrettyJSONTests {
		res, err := PrettyPrintJSON([]byte(test.in), "", "  ")
		str := string(res)
		if err != nil {
			str = "Err: " + err.Error()
		}

		if str != test.exp {
			t.Fatalf("Pretty Test:\n%s\nExp:\n%s\nGot:\n%s",
				test.in, test.exp, str)
		}
	}
}
