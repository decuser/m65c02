package machine

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// KIM1Config describes motherboard choices; it never changes ROM code.
// The supported syntax is a strict, deliberately small TOML subset.
type KIM1Config struct {
	Model          string
	ClockHz        uint64
	ROM002, ROM003 string
	TTYSelect      bool
}

func LoadKIM1Config(path string) (KIM1Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return KIM1Config{}, err
	}
	defer f.Close()
	c := KIM1Config{ClockHz: 1000000}
	section := ""
	s := bufio.NewScanner(f)
	for line := 1; s.Scan(); line++ {
		v := strings.TrimSpace(strings.SplitN(s.Text(), "#", 2)[0])
		if v == "" {
			continue
		}
		if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
			section = strings.TrimSpace(v[1 : len(v)-1])
			continue
		}
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return c, fmt.Errorf("%s:%d: invalid assignment", path, line)
		}
		key := section + "." + strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])
		switch key {
		case "machine.model", "rom.chip_002", "rom.chip_003":
			if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
				return c, fmt.Errorf("%s:%d: expected quoted string", path, line)
			}
			str := raw[1 : len(raw)-1]
			switch key {
			case "machine.model":
				c.Model = str
			case "rom.chip_002":
				c.ROM002 = str
			case "rom.chip_003":
				c.ROM003 = str
			}
		case "machine.clock_hz":
			c.ClockHz, err = strconv.ParseUint(raw, 10, 64)
			if err != nil || c.ClockHz == 0 {
				return c, fmt.Errorf("%s:%d: invalid clock_hz", path, line)
			}
		case "jumpers.tty_select":
			c.TTYSelect, err = strconv.ParseBool(raw)
			if err != nil {
				return c, fmt.Errorf("%s:%d: invalid tty_select", path, line)
			}
		default:
			return c, fmt.Errorf("%s:%d: unknown setting %s", path, line, key)
		}
	}
	if err = s.Err(); err != nil {
		return c, err
	}
	if c.Model != "kim1" || c.ROM002 == "" || c.ROM003 == "" {
		return c, fmt.Errorf("%s: requires model=kim1 and both ROM paths", path)
	}
	return c, nil
}
