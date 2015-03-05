package keys

import (
	"fmt"
	"testing"
)

func TestGenerationEmptyParamMap(t *testing.T) {
	emptyParams :=  make(map[string]string)
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=0&dpi=0")
}

func TestGenerationWithFullParamMap(t *testing.T) {
	emptyParams :=  map[string]string {
		"width": "300",
		"height": "450",
		"dpi": "96",
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=450&dpi=96")
}

func TestGenerationWithInvalidNegativeParams(t *testing.T) {
	emptyParams := map[string]string {
		"width": "-10",
		"height": "450",
		"dpi": "96",
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=450&dpi=96")
}

func TestGenerationWithInvalidStringParams(t *testing.T) {
	emptyParams :=  map[string]string {
		"width": "foo-bar",
		"height": "450",
		"dpi": "96",
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=0&height=450&dpi=96")
}

func TestGenerationWithParamMap(t *testing.T) {
	emptyParams :=  map[string]string {
		"width": "300",
		"dpi": "96",
	}
	key := Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=96")

	emptyParams =  map[string]string {
		"dpi": "96",
		"width": "300",
	}
	key = Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=96")

	emptyParams =  map[string]string {
		"height": "0",
		"width": "300",
	}
	key = Generate("test.jpg", emptyParams)
	assertEquals(t, key, "test.jpg?width=300&height=0&dpi=0")
}

func assertEquals(t *testing.T, actual, expected string) {
	if expected == actual {
		t.Log("Assertion passed.")
	} else {
		t.Error(fmt.Sprintf("Assertion failed! Expected: %s, Actual: %s", expected, actual))
	}
}
