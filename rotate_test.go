/*
 * Copyright (c) 2016, Psiphon Inc.
 * All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package rotate

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRotateKeepsWriting(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"
	rotatedPath := os.TempDir() + "/rotatable.log.1"
	defer os.Remove(logPath)
	defer os.Remove(rotatedPath)

	f, err := NewRotatableFileWriter(logPath, 0, true, 0666)
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

	f, err := NewRotatableFileWriter(logPath, 0, true, 0666)
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

func TestOtherCreatesNextFile(t *testing.T) {
	testCreateNextFile(t, false)
}

func TestBothCreateNextFile(t *testing.T) {
	testCreateNextFile(t, true)
}

func testCreateNextFile(t *testing.T, selfCreateFile bool) {
	logPath := os.TempDir() + "/rotatable.log"
	rotatedPath := os.TempDir() + "/rotatable.log.1"
	defer os.Remove(logPath)
	defer os.Remove(rotatedPath)

	// The log manager creates the file.

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		t.Fatalf("Error creating file: %s", err)
	}
	logFile.Close()

	// RotatableFileWriter will also attempt to create the file when
	// selfCreateFile is true.

	f, err := NewRotatableFileWriter(logPath, 2, selfCreateFile, 0666)
	if err != nil {
		t.Fatalf("Unable to set log output: %s", err)
	}

	// A write here should succeed.

	_, err = f.Write([]byte("0\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Rotate the file, creating the next file but leaving it temporarily
	// inaccessible. Writes will fail with "permission denied" (unless the test
	// is run as root).

	err = os.Rename(logPath, rotatedPath)
	if err != nil {
		t.Fatalf("Error renaming file: %s", err)
	}
	logFile, err = os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0466)
	if err != nil {
		t.Fatalf("Error creating file: %s", err)
	}
	logFile.Close()

	// Start trying another write; it should fail and retry.

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := f.Write([]byte("1\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
	}()

	// After a delay, make the file accessible.

	time.Sleep(1 * time.Millisecond)
	err = os.Chmod(logPath, 0666)
	if err != nil {
		t.Fatalf("Error modifying file: %s", err)
	}

	wg.Wait()

	// Verify file contents.

	contents, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Unable read file '%s': %s", logPath, err)
	}
	expected := "1\n"
	actual := string(contents)
	if actual != expected {
		t.Errorf("Got: %s, Expected: %s", actual, expected)
	}
}

func TestNormalWrite(t *testing.T) {
	logPath := os.TempDir() + "/rotatable.log"
	defer os.Remove(logPath)

	f, err := NewRotatableFileWriter(logPath, 0, true, 0666)
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

	f, err := NewRotatableFileWriter(logPath, 0, true, 0666)
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

	f, err := NewRotatableFileWriter(logPath, 0, true, 0666)
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
