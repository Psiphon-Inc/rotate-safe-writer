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
	defer os.Remove(logPath)
	defer os.Remove(rotatedPath)

	f, err := NewRotatableFileWriter(logPath, 0666)
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

}

func TestDeleteWritesNewFile(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"
	defer os.Remove(logPath)

	f, err := NewRotatableFileWriter(logPath, 0666)
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
}

func TestNormalWrite(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"
	defer os.Remove(logPath)

	f, err := NewRotatableFileWriter(logPath, 0666)
	if err != nil {
		t.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	for i := 0; i < 5; i++ {
		log.Println(i)
	}

	contents, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Unable read file '%s': %s", logPath, err)
	}
	// The file should have all 5 lines in it
	expected := "0\n1\n2\n3\n4\n"
	actual := string(contents)
	if actual != expected {
		t.Errorf("Got: %s, Expected: %s", actual, expected)
	}
}

func benchmarkStandardFileLogger(b *testing.B) {
	logPath := os.TempDir() + "/rotatable.log"
	defer os.Remove(logPath)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		b.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		log.Println(n)
	}
}

func BenchmarkStandardFileLogger1(b *testing.B)       { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger10(b *testing.B)      { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger100(b *testing.B)     { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger1000(b *testing.B)    { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger10000(b *testing.B)   { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger100000(b *testing.B)  { benchmarkStandardFileLogger(b) }
func BenchmarkStandardFileLogger1000000(b *testing.B) { benchmarkStandardFileLogger(b) }

func benchmarkRotatableWriterLogger(b *testing.B) {
	logPath := os.TempDir() + "/rotatable.log"
	defer os.Remove(logPath)

	f, err := NewRotatableFileWriter(logPath, 0666)
	if err != nil {
		b.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		log.Println(n)
	}
}

func BenchmarkRotatableWriterLogger1(b *testing.B)       { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger10(b *testing.B)      { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger100(b *testing.B)     { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger1000(b *testing.B)    { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger10000(b *testing.B)   { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger100000(b *testing.B)  { benchmarkRotatableWriterLogger(b) }
func BenchmarkRotatableWriterLogger1000000(b *testing.B) { benchmarkRotatableWriterLogger(b) }

func benchmarkRotatableWriterLoggerWithSingleRotation(b *testing.B) {
	logPath := os.TempDir() + "/rotatable.log"
	rotatedPath := os.TempDir() + "/rotatable.log.1"
	defer os.Remove(logPath)
	defer os.Remove(rotatedPath)

	f, err := NewRotatableFileWriter(logPath, 0666)
	if err != nil {
		b.Fatalf("Unable to set log output: %s", err)
	}

	log.SetFlags(0) // disables all formatting
	log.SetOutput(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if n == (b.N / 2) {
			err := os.Rename(logPath, rotatedPath)
			if err != nil {
				b.Fatalf("Error renaming file: %s", err)
			}
		}
		log.Println(n)
	}
}

func BenchmarkRotatableWriterLoggerWithSingleRotation1(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation10(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation100(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation1000(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation10000(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation100000(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
func BenchmarkRotatableWriterLoggerWithSingleRotation1000000(b *testing.B) {
	benchmarkRotatableWriterLoggerWithSingleRotation(b)
}
