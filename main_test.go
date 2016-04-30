package main

import (
	"testing"
)
func TestPush(t *testing.T) {
	cmd:=newPushCmd("http://","http://","123456")
	err:=cmd.CmdExec.Run()
	if nil==err{
		t.Fatal("test error")
	}
}