package rotate

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestRotateKeepsWriting(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"
	rotatedPath := os.TempDir() + "/rotatable.log.1"

	f, err := NewRotatableFileWriter(logPath, 0777)
	if err != nil {
		t.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	for i := 0; i < 5; i++ {
		if i == 3 {
			err := os.Rename(logPath, rotatedPath)
			if err != nil {
				t.Fatalf("Error renaming file: %s", err)
			}
		}

		log.Println(i)
	}

	contents, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Unable read file '%s': %s", logPath, err)
	}
	// The re-opened file at the original path should only have the new lines in it
	expected := "3\n4\n"
	actual := string(contents)
	if actual != expected {
		t.Errorf("Got: %s, Expected: %s", actual, expected)
	}

	contents, err = ioutil.ReadFile(rotatedPath)
	if err != nil {
		t.Fatalf("Unable read file '%s': %s", rotatedPath, err)
	}
	// The rotated file at the new path should have the original lines in it
	expected = "0\n1\n2\n"
	actual = string(contents)
	if actual != expected {
		t.Errorf("Got: %s, Expected: %s", actual, expected)
	}

	os.Remove(logPath)
	os.Remove(rotatedPath)
}

func TestDeleteWritesNewFile(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"

	f, err := NewRotatableFileWriter(logPath, 0777)
	if err != nil {
		t.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	for i := 0; i < 5; i++ {
		if i == 3 {
			contents, err := ioutil.ReadFile(logPath)
			if err != nil {
				t.Fatalf("Unable read file '%s': %s", logPath, err)
			}
			// The file before deletion should have the original lines in it
			expected := "0\n1\n2\n"
			actual := string(contents)
			if actual != expected {
				t.Errorf("Got: %s, Expected: %s", actual, expected)
			}

			err = os.Remove(logPath)
			if err != nil {
				t.Fatalf("Error removing file: %s", err)
			}
		}

		log.Println(i)
	}

	contents, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Unable read file '%s': %s", logPath, err)
	}
	// The recreated file after a new inode is assigned should only have the new lines in it
	expected := "3\n4\n"
	actual := string(contents)
	if actual != expected {
		t.Errorf("Got: %s, Expected: %s", actual, expected)
	}

	os.Remove(logPath)
}
