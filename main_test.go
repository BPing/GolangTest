package main

import (
	"testing"
)

func TestPush(t *testing.T) {
	cmd := newPushCmd("http://", "http://", "123456")
	err := cmd.CmdExec.Run()
	if nil == err {
		t.Fatal("test error")
	}
}

func TestSign(t *testing.T) {
	key = "dw5445gg77gdg4a45"
	authUser = "4t"
	signFlag := "4t:20d9bef196a3a65651fec2c3a09128d2"
	param := map[string]string{"d":"bb"}
	if (!CheckSign(param, signFlag)) {
		t.Fatal(Sign(param))
	}
}