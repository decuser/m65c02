//go:build linux
// +build linux

package machine

import (
	"bytes"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestKIM1PTYOriginalMonitor(t *testing.T) {
	p002, p003 := os.Getenv("M65C02_KIM_ROM002"), os.Getenv("M65C02_KIM_ROM003")
	if p002 == "" || p003 == "" {
		t.Skip("supply both original KIM-1 ROM paths")
	}
	k, err := NewKIM1FromFiles(p002, p003)
	if err != nil {
		t.Fatal(err)
	}
	k.TTYSelect = true
	s := k.ConnectSerial(9091)
	port, err := NewPTY()
	if err != nil {
		t.Skipf("PTY unavailable: %v", err)
	}
	defer port.Close()
	client, err := os.OpenFile(port.SlavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	stty := exec.Command("stty", "raw", "-echo")
	stty.Stdin = client
	if err = stty.Run(); err != nil {
		t.Fatal(err)
	}
	if err = syscall.SetNonblock(int(client.Fd()), true); err != nil {
		t.Fatal(err)
	}
	clientFD := int(client.Fd())
	masterFD := int(port.Master.Fd())
	if err = syscall.SetNonblock(clientFD, true); err != nil {
		t.Fatal(err)
	}
	if err = syscall.SetNonblock(masterFD, true); err != nil {
		t.Fatal(err)
	}
	if err = k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	k.ReleaseReset()
	var pending, received []byte
	readBuf := make([]byte, 128)
	for k.Bus.Cycles < 13000000 {
		if k.Bus.Cycles == 2000 {
			s.Queue([]byte{0x7f})
		}
		if k.Bus.Cycles == 3300000 {
			if _, err = client.Write([]byte("1234 ")); err != nil {
				t.Fatal(err)
			}
		}
		if k.Bus.Cycles%1000 == 0 {
			n, readErr := syscall.Read(masterFD, readBuf)
			if n > 0 {
				pending = append(pending, readBuf[:n]...)
			}
			if readErr != nil && readErr != syscall.EAGAIN {
				t.Fatal(readErr)
			}
			n, readErr = syscall.Read(clientFD, readBuf)
			if n > 0 {
				received = append(received, readBuf[:n]...)
			}
			if readErr != nil && readErr != syscall.EAGAIN {
				t.Fatal(readErr)
			}
		}
		if len(pending) > 0 && s.RXIdle() {
			s.Queue(pending[:1])
			pending = pending[1:]
		}
		if err = k.Clock(); err != nil {
			t.Fatal(err)
		}
		s.DrainEdges()
		for _, b := range s.DrainDecoded() {
			_, _ = port.Master.Write([]byte{b})
		}
	}
	if !bytes.Contains(received, []byte("KIM")) || !bytes.Contains(received, []byte("1234 FF")) {
		t.Fatalf("PTY did not carry monitor greeting and reply: %q", received)
	}
	t.Logf("PTY client received %q", received)
}

func TestPTYTransfersBytes(t *testing.T) {
	p, err := NewPTY()
	if err != nil {
		t.Skipf("PTY not available: %v", err)
	}
	defer p.Close()
	slave, err := os.OpenFile(p.SlavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	if err = syscall.SetNonblock(int(slave.Fd()), true); err != nil {
		t.Fatal(err)
	}
	if _, err = slave.Write([]byte{'Q', '\n'}); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	deadline := time.Now().Add(time.Second)
	for {
		n, readErr := syscall.Read(int(p.Master.Fd()), buf)
		if n > 0 {
			if buf[0] != 'Q' {
				t.Fatalf("got %q", buf[:n])
			}
			break
		}
		if readErr != nil && readErr != syscall.EAGAIN {
			t.Fatal(readErr)
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout reading PTY")
		}
		time.Sleep(time.Millisecond)
	}
}
