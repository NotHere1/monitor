package utils

import "fmt"
import "time"
import "testing"

func TestParseSection(t * testing.T) {
	if sec1 != "api" {
		t.Errorf("TestParseSection failed, Want: 'api'. Got: %s", sec1)
	} else if sec2 != "help" {
		t.Errorf("TestParseSection failed, Want: 'help'. Got: %s", sec2)
	} else if sec3 != "/" {
		t.Errorf("TestParseSection failed, Want: '/'. Got: %s", sec3)
	} else if sec4 != "/" {
		t.Errorf("TestParseSection failed, Want: '/'. Got: %s", sec4)		
	}
}

func TestParseEpochString(t *testing.T) {
	epoch, err := ParseEpochString("1549573860")
	if err != nil {
		 t.Errorf("ParseEpochString failed, got: %s", err)
	}
	year := epoch.Year()
	month := epoch.Month()
	day := epoch.Day()
	if year != 2019 || month != time.Month(2) || day != 7 {
		t.Errorf("ParseEpochString failed. Want %s. Got: %s", "2019-02-07", epoch)
	}
}

func TestAccuMap(t *testing.T) {
	mp := map[string]int{"a": 10, "b": 20, "c": 30}
	accu := map[string]int{"a": 1, "c":100}
	collect := AccuMap(mp, accu)
	if collect["a"] != 11 || collect["b"] != 20 || collect["c"] != 130 {
		t.Errorf("AccuMap failed. Want {a:11,b:20,c:130}. Got: %s", fmt.Sprint(collect))	
	}
}

