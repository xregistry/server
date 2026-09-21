package common

import (
	"fmt"
	"maps"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type Config struct {
	// We use an "any" rather than a "string" because while the config file
	// itself will always be read-in as strings, if people want to use this
	// struct as a way to share data across the app then they'll need to be
	// able to store more than just strings, e.g. pointers to structs
	Data map[string]any

	// FileName will be the true path of what was loaded, or "" if the file
	// is not found
	FileName string // e.g. .xr
}

func NewConfig(fn string) *Config {
	return &Config{
		Data:     map[string]any{},
		FileName: fn,
	}
}

func (c *Config) Clone() *Config {
	return &Config{
		Data:     maps.Clone(c.Data),
		FileName: c.FileName,
	}
}

func (c *Config) SetFileName(fn string) *Config {
	c.FileName = fn
	return c
}

func (c *Config) GetFilename(fn string) string {
	return c.FileName
}

func (c *Config) Get(name string) any {
	if c.Data == nil {
		return ""
	}
	return c.Data[name]
}

func (c *Config) GetAsString(name string) string {
	if c.Data == nil {
		return ""
	}

	val := c.Data[name]
	if IsNil(val) {
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}

	if i, ok := val.(int); ok {
		return fmt.Sprintf("%d", i)
	}

	panic(fmt.Sprintf("%q needs to be a string, not %T", name, val))
}

func (c *Config) GetAsInt(name string) int {
	if c.Data == nil {
		return 0
	}

	val := c.Data[name]
	if IsNil(val) {
		return 0
	}

	if i, ok := val.(int); ok {
		return i
	}

	if s, ok := val.(string); ok {
		i, _ := strconv.Atoi(s)
		return i
	}

	panic(fmt.Sprintf("%q must be an int, not %T", name, val))
}

// GetAsBool returns name's value parsed as a bool (via strconv.ParseBool,
// so "1"/"t"/"T"/"TRUE"/"true"/"True" etc. are all accepted), or false if
// name is unset or its value isn't a valid bool.
func (c *Config) GetAsBool(name string) bool {
	if c.Data == nil {
		return false
	}
	val := c.Data[name]
	if IsNil(val) {
		return false
	}

	// Try bool first
	if b, ok := val.(bool); ok {
		return b
	}

	// try converting from string
	if s, ok := val.(string); ok {
		b, _ := strconv.ParseBool(s)
		return b
	}

	panic(fmt.Sprintf("%q must be a bool, not %T", name, val))
}

func (c *Config) Set(name string, value any) *Config {
	name = strings.TrimSpace(name)
	// value = strings.TrimSpace(value)

	PanicIf(name == "", "Can't call Config.Set with empty name")

	if IsNil(value) {
		if c.Data != nil {
			delete(c.Data, name)
			if len(c.Data) == 0 {
				c.Data = nil
			}
		}
	} else {
		if c.Data == nil {
			c.Data = map[string]any{}
		}
		c.Data[name] = value
	}
	return c
}

func (c *Config) Clear() *Config {
	c.Data = nil
	c.FileName = ""
	return c
}

func (c *Config) Load(fn string) error {
	originFn := fn

	if fn == "" {
		fn = c.FileName
	}

	fn = strings.TrimSpace(fn)

	if fn != "" && !path.IsAbs(fn) {
		// Not absolute so search for it
		if _, err := os.Stat("./" + fn); err == nil {
			fn = "./" + fn
		} else {
			homePath, _ := os.UserHomeDir()
			if homePath != "" {
				homePath = homePath + "/" + fn
				if _, err := os.Stat(homePath); err == nil {
					fn = homePath
				}
			}
		}
	}

	// In the end, if the config file is missing just return, not an error.
	// Maybe one day we should pass in arg and let the caller decide
	if originFn == "" {
		if _, err := os.Stat(fn); err != nil {
			c.FileName = "" // let them know nothing was loaded
			return nil
		}
	}

	buf, err := os.ReadFile(fn)
	if err != nil {
		return fmt.Errorf("Error loading config file (%s): %s", fn, err)
	}

	// Format:
	// # comment
	// prop[.prop]: value

	lines := strings.Split(string(buf), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		name = strings.TrimSpace(name)

		if name == "" {
			return fmt.Errorf("Error in config file (%s): missing name on: %s",
				fn, line)
		}

		value = strings.TrimSpace(value)
		c.Set(name, value)
	}

	c.FileName = fn
	return nil
}

func (c *Config) SetFromEnv(propName, envName string) *Config {
	val := os.Getenv(envName)
	if val != "" {
		c.Set(propName, val)
	}
	return c
}

func (c *Config) SetFromCmd(prop string, cmd *cobra.Command, flag string) *Config {
	if cmd.Flags().Changed(flag) {
		if val, _ := cmd.Flags().GetString(flag); val != "" {
			c.Set(prop, val)
		}
	}
	return c
}

func (c *Config) SetFromCmdInt(prop string, cmd *cobra.Command, flag string) *Config {
	if cmd.Flags().Changed(flag) {
		val, _ := cmd.Flags().GetInt(flag)
		c.Set(prop, fmt.Sprintf("%d", val))
	}
	return c
}

// Utils - some are client vs server specific

func (c *Config) GetHeaders() map[string]string {
	headers := map[string]string(nil)

	for key, value := range c.Data {
		if !strings.HasPrefix(key, "header.") {
			continue
		}
		key = strings.TrimSpace(key[7:])
		if key != "" {
			if headers == nil {
				headers = map[string]string{}
			}
			if s, ok := value.(string); ok {
				headers[key] = s
			} else {
				panic(fmt.Sprintf("%q must be a string, not %T", key, value))
			}
		}
	}
	return headers
}
