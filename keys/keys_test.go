package keys

import (
	"fmt"
	"testing"
)

func TestGenerationEmptyParamMap(t *testing.T) {
	emptyParams :=  make(map[string][]string)
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=0&dpi=0")
}

func TestGenerationWithFullParamMap(t *testing.T) {
	emptyParams :=  map[string][]string {
		"width": []string {"300"},
		"height": []string {"450"},
		"dpi": []string {"96"},
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=450&dpi=96")
}

func TestGenerationWithInvalidNegativeParams(t *testing.T) {
	emptyParams := map[string][]string {
		"width": []string {"-10"},
		"height": []string {"450"},
		"dpi": []string {"96"},
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=450&dpi=96")
}

func TestGenerationWithInvalidStringParams(t *testing.T) {
	emptyParams :=  map[string][]string {
		"width": []string {"foo-bar"},
		"height": []string {"450"},
		"dpi": []string {"96"},
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=450&dpi=96")
}

func TestGenerationWithParamMap(t *testing.T) {
	emptyParams :=  map[string][]string {
		"width": []string {"300"},
		"dpi": []string {"96"},
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=96")

	emptyParams =  map[string][]string {
		"dpi": []string {"96"},
		"width": []string {"300"},
	}
	key = Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=96")

	emptyParams =  map[string][]string {
		"height": []string {"0"},
		"width": []string {"300"},
	}
	key = Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=0")
}

func TestParseKey(t *testing.T) {
 	captures, err := Parse("foo.jpg?width=300&height=400&dpi=0")
	assertNoError(t, err)
	assertTrue(t, len(captures) == 4)
	assertEquals(t, captures[FILE_NAME], "foo.jpg")
	assertEquals(t, captures["width"], "300")
	assertEquals(t, captures["height"], "400")
	assertEquals(t, captures["dpi"], "0")
}

func BenchmarkParseKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Parse("foo.jpg?width=300&height=400&dpi=0")
	}
}

func assertEquals(t *testing.T, actual, expected string) {
	if expected == actual {
		t.Log("Assertion passed.")
	} else {
		t.Error(fmt.Sprintf("Assertion failed! Expected: %s, Actual: %s", expected, actual))
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Error(fmt.Sprintf("Assertion failed! Error: %v", err))
	} else {
		t.Log("Assertion passed.")
	}
}

func assertTrue(t *testing.T, actual bool) {
	if actual {
		t.Log("Assertion passed.")
	} else {
		t.Error(fmt.Sprintf("Assertion failed! Expected to be true, but is: %v", actual))
	}
}
