package main

import "testing"

func TestReplaceParam(t *testing.T) {
	u := &User{}
	u.param = map[string]string{
		"GAME_SN": "12345",
	}

	{
		dest := u.replaceParam("/x/game/start/[GAME_SN]")
		if dest != "/x/game/start/12345" {
			t.Error("replace err", dest)
		}
	}
	{
		dest := u.replaceParam("/x/game/start/[GAME_SN]/abcde")
		if dest != "/x/game/start/12345/abcde" {
			t.Error("replace err", dest)
		}
	}

}
