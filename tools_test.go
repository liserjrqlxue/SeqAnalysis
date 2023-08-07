package main

import (
	"os"
	"testing"
)

// TestOpen is a test function that tests the Open function.
//
// This function tests the Open function by performing the following test cases:
//   - Test case 1: Opening an existing file using embedFS.
//   - Test case 2: Opening a non-existing file using embedFS.
//   - Test case 3: Opening a file using os.Open.
//
// Each test case validates the behavior of the Open function with different scenarios.
//
// Parameters:
//   - t: A pointer to the testing.T struct.
//
// Return type: None.
func TestOpen(t *testing.T) {
	// Test case 1: Opening an existing file using embedFS
	t.Run("Open existing file with embedFS", func(t *testing.T) {
		embedFS := etcEMFS      // initialize your embed.FS here with test data
		path := "etc/sheet.txt" // specify the path of an existing file in the embedFS
		_, err := Open(path, "", embedFS)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
	})

	// Test case 2: Opening a non-existing file using embedFS
	t.Run("Open non-existing file with embedFS", func(t *testing.T) {
		embedFS := etcEMFS    // initialize your embed.FS here with test data
		path := "etc/xxx.txt" // specify the path of a non-existing file in the embedFS
		_, err := Open(path, "", embedFS)
		if err == nil {
			t.Error("Expected an error, but got nil")
		}
	})

	// Test case 3: Opening a file using os.Open
	t.Run("Open file with os.Open", func(t *testing.T) {
		embedFS := etcEMFS    // initialize your embed.FS here with test data
		path := "etc/yyy.txt" // specify the path of an existing file
		var f, _ = os.Create(path)
		f.Close()
		_, err := Open(path, exPath, embedFS)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
	})
}
