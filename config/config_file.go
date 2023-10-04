package config

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"time"
)

type FileSettings struct {
	Configuration `yaml:",inline"`
	// older settings will be aggregated into the generic map, should be read via cli.Context
	Settings map[string]interface{} `yaml:",inline"`
}

func (c *FileSettings) Source() string {
	return c.sourceFile
}

func (c *FileSettings) Int(name string) (int, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(int); ok {
			return v, nil
		}
		return 0, fmt.Errorf("expected int found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) Int64(name string) (int64, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(int64); ok {
			return v, nil
		}
		return 0, fmt.Errorf("expected int64 found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) Uint(name string) (uint, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(uint); ok {
			return v, nil
		}
		return 0, fmt.Errorf("expected uint found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) Uint64(name string) (uint64, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(uint64); ok {
			return v, nil
		}
		return 0, fmt.Errorf("expected uint64 found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) Duration(name string) (time.Duration, error) {
	if raw, ok := c.Settings[name]; ok {
		switch v := raw.(type) {
		case time.Duration:
			return v, nil
		case string:
			return time.ParseDuration(v)
		}
		return 0, fmt.Errorf("expected duration found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) Float64(name string) (float64, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(float64); ok {
			return v, nil
		}
		return 0, fmt.Errorf("expected float found %T for %s", raw, name)
	}
	return 0, nil
}

func (c *FileSettings) String(name string) (string, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(string); ok {
			return v, nil
		}
		return "", fmt.Errorf("expected string found %T for %s", raw, name)
	}
	return "", nil
}

func (c *FileSettings) StringSlice(name string) ([]string, error) {
	if raw, ok := c.Settings[name]; ok {
		if slice, ok := raw.([]interface{}); ok {
			strSlice := make([]string, len(slice))
			for i, v := range slice {
				str, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("expected string, found %T for %v", i, v)
				}
				strSlice[i] = str
			}
			return strSlice, nil
		}
		return nil, fmt.Errorf("expected string slice found %T for %s", raw, name)
	}
	return nil, nil
}

func (c *FileSettings) IntSlice(name string) ([]int, error) {
	if raw, ok := c.Settings[name]; ok {
		if slice, ok := raw.([]interface{}); ok {
			intSlice := make([]int, len(slice))
			for i, v := range slice {
				str, ok := v.(int)
				if !ok {
					return nil, fmt.Errorf("expected int, found %T for %v ", v, v)
				}
				intSlice[i] = str
			}
			return intSlice, nil
		}
		if v, ok := raw.([]int); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected int slice found %T for %s", raw, name)
	}
	return nil, nil
}

func (c *FileSettings) Int64Slice(name string) ([]int64, error) {
	if raw, ok := c.Settings[name]; ok {
		if slice, ok := raw.([]interface{}); ok {
			intSlice := make([]int64, len(slice))
			for i, v := range slice {
				str, ok := v.(int64)
				if !ok {
					return nil, fmt.Errorf("expected int64, found %T for %v ", v, v)
				}
				intSlice[i] = str
			}
			return intSlice, nil
		}
		if v, ok := raw.([]int64); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected int64 slice found %T for %s", raw, name)
	}
	return nil, nil
}

func (c *FileSettings) Float64Slice(name string) ([]float64, error) {
	if raw, ok := c.Settings[name]; ok {
		if slice, ok := raw.([]interface{}); ok {
			intSlice := make([]float64, len(slice))
			for i, v := range slice {
				str, ok := v.(float64)
				if !ok {
					return nil, fmt.Errorf("expected float64, found %T for %v ", v, v)
				}
				intSlice[i] = str
			}
			return intSlice, nil
		}
		if v, ok := raw.([]float64); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected float64 slice found %T for %s", raw, name)
	}
	return nil, nil
}

func (c *FileSettings) Generic(name string) (cli.Generic, error) {
	return nil, errors.New("option type Generic not supported")
}

func (c *FileSettings) Bool(name string) (bool, error) {
	if raw, ok := c.Settings[name]; ok {
		if v, ok := raw.(bool); ok {
			return v, nil
		}
		return false, fmt.Errorf("expected boolean found %T for %s", raw, name)
	}
	return false, nil
}

func (c *FileSettings) isSet(name string) bool {
	if raw, ok := c.Settings[name]; ok {
		if _, ok := raw.(bool); ok {
			return true
		}
		return false
	}
	return false
}
