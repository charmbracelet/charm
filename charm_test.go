package charm

import (
	"strings"
	"testing"
)

func TestNameValidation(t *testing.T) {
	if validateName("") {
		t.Error("validated the empty string, which should have failed")
	}
	if !validateName("a") {
		t.Error("failed validating the single character 'a', which should have passed")
	}
	if !validateName("A") {
		t.Error("failed validating the single character 'A', which should have passed")
	}
	if validateName("épicerie") {
		t.Error("validated a string with an 'é', which should have failed")
	}
	if validateName("straße") {
		t.Error("validated a string with an 'ß', which should have failed")
	}
	if validateName("mr.green") {
		t.Error("validated a string with a period, which should have failed")
	}
	if validateName("mister green") {
		t.Error("validated a string with a space, which should have failed")
	}
	if validateName("茶") {
		t.Error("validated the string '茶', which should have failed")
	}
	if validateName("😀") {
		t.Error("validated an emoji, which should have failed")
	}
	if !validateName(strings.Repeat("x", 64)) {
		t.Error("falied validating a 64-character-string, which should have passed")
	}
	if validateName(strings.Repeat("x", 65)) {
		t.Error("validated a 65-character-string, which should have failed")
	}
}
