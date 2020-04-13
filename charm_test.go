package charm

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestNameValidation(t *testing.T) {
	if ValidateName("") {
		t.Error("validated the empty string, which should have failed")
	}
	if !ValidateName("a") {
		t.Error("failed validating the single character 'a', which should have passed")
	}
	if !ValidateName("A") {
		t.Error("failed validating the single character 'A', which should have passed")
	}
	if ValidateName("épicerie") {
		t.Error("validated a string with an 'é', which should have failed")
	}
	if ValidateName("straße") {
		t.Error("validated a string with an 'ß', which should have failed")
	}
	if ValidateName("mr.green") {
		t.Error("validated a string with a period, which should have failed")
	}
	if ValidateName("mister green") {
		t.Error("validated a string with a space, which should have failed")
	}
	if ValidateName("茶") {
		t.Error("validated the string '茶', which should have failed")
	}
	if ValidateName("😀") {
		t.Error("validated an emoji, which should have failed")
	}
	if !ValidateName(strings.Repeat("x", 50)) {
		t.Error("falied validating a 50-character-string, which should have passed")
	}
	if ValidateName(strings.Repeat("x", 51)) {
		t.Error("validated a 51-character-string, which should have failed")
	}
}
