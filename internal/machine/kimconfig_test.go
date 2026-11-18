package machine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKIM1Config(t *testing.T) {
	p := filepath.Join(t.TempDir(), "kim1.toml")
	data := []byte("[machine]\nmodel = \"kim1\"\nclock_hz = 1000000\n[rom]\nchip_002 = \"002.bin\"\nchip_003 = \"003.bin\"\n[jumpers]\ntty_select = true\n")
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadKIM1Config(p)
	if err != nil || !c.TTYSelect || c.ROM002 != "002.bin" || c.ClockHz != 1000000 {
		t.Fatalf("config=%+v err=%v", c, err)
	}
	if err := os.WriteFile(p, append(data, []byte("unknown = true\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKIM1Config(p); err == nil {
		t.Fatal("accepted unknown setting")
	}
}
