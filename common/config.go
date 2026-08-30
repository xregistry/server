package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type Config struct {
	Data     map[string]string
	FileName string // e.g. .xr
}

func NewConfig(fn string) *Config {
	PanicIf(fn == "", "NewConfig needs a filename")
	return &Config{
		Data:     map[string]string{},
		FileName: fn,
	}
}

func (c *Config) SetFileName(fn string) *Config {
	PanicIf(fn == "", "SetConfig needs a filename")
	c.FileName = fn
	return c
}

func (c *Config) GetFilename(fn string) string {
	return c.FileName
}

func (c *Config) Get(name string) string {
	if c.Data == nil {
		return ""
	}
	return c.Data[name]
}

func (c *Config) GetAsInt(name string) int {
	if c.Data == nil {
		return 0
	}
	val := c.Data[name]
	if val == "" {
		return 0
	}
	i, _ := strconv.Atoi(val)
	return i
}

func (c *Config) Set(name, value string) *Config {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	PanicIf(name == "", "Can't call Config.Set with empty name")

	if value == "" {
		if c.Data != nil {
			delete(c.Data, name)
			if len(c.Data) == 0 {
				c.Data = nil
			}
		}
	} else {
		if c.Data == nil {
			c.Data = map[string]string{}
		}
		c.Data[name] = value
	}
	return c
}

func (c *Config) Clear() *Config {
	c.Data = nil
	return c
}

func (c *Config) Load(fn string) error {
	if fn == "" {
		if _, err := os.Stat("./" + c.FileName); err == nil {
			fn = "./" + c.FileName
		} else {
			path, _ := os.UserHomeDir()
			if path != "" {
				path = path + "/" + c.FileName
				if _, err := os.Stat(path); err == nil {
					fn = path
				} else {
					// No config file, just return
					c.FileName = ""
					return nil
				}
			}
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
			headers[key] = value
		}
	}
	return headers
}
