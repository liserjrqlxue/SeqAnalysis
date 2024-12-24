package seqAnalysis

import (
	"os"
	"reflect"
	"testing"
)

func TestWriteStatsTxt(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "stats.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Create a SeqInfo object for testing
	info := &SeqInfo{
		Name:                 "Test",
		IndexSeq:             "ACGT",
		Seq:                  []byte("ATCG"),
		YieldCoefficient:     1.5,
		AverageYieldAccuracy: 0.9,
		Stats: map[string]int{
			"AllReadsNum":         100,
			"IndexReadsNum":       50,
			"AnalyzedReadsNum":    80,
			"RightReadsNum":       75,
			"ErrorReadsNum":       20,
			"Deletion":            10,
			"DeletionSingle":      5,
			"DeletionDiscrete2":   3,
			"DeletionContinuous2": 2,
			"DeletionDiscrete3":   1,
			"ErrorInsReadsNum":    4,
			"ErrorInsDelReadsNum": 2,
			"ErrorMutReadsNum":    6,
			"ErrorOtherReadsNum":  9,
		},
	}

	// Call the WriteStatsTxt function
	info.WriteStatsTxt(tempFile)

	// Read the content of the temporary file
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temporary file: %v", err)
	}

	// Assert that the content matches the expected value
	expectedContent := "Test\tACGT\tATCG\t4\t100\t50\t80\t75\t1.500000\t0.900000\t0.250000\t0.125000\t0.062500\t0.037500\t0.025000\t0.012500\t0.050000\t0.025000\t0.075000\t0.112500\n"
	if string(content) != expectedContent {
		t.Errorf("Unexpected content in the file.\nExpected: %s\nActual: %s", expectedContent, string(content))
	}
}

func TestByteFloatList_Len(t *testing.T) {
	// Test when the ByteFloatList is empty
	{
		l := ByteFloatList{}
		expected := 0

		if got := l.Len(); got != expected {
			t.Errorf("Len() = %d; want %d", got, expected)
		}
	}

	// Test when the ByteFloatList has one element
	{
		l := ByteFloatList{ByteFloat{'a', 1.23}}
		expected := 1

		if got := l.Len(); got != expected {
			t.Errorf("Len() = %d; want %d", got, expected)
		}
	}

	// Test when the ByteFloatList has multiple elements
	{
		l := ByteFloatList{
			ByteFloat{'a', 1.23},
			ByteFloat{'b', 2.34},
			ByteFloat{'c', 3.45},
		}
		expected := 3

		if got := l.Len(); got != expected {
			t.Errorf("Len() = %d; want %d", got, expected)
		}
	}
}

func TestByteFloatList_Less(t *testing.T) {
	// Test case 1: Comparing two elements with different values
	l := ByteFloatList{
		{Value: 1.5},
		{Value: 2.5},
	}
	if !l.Less(0, 1) {
		t.Errorf("Expected l[0] to be less than l[1]")
	}

	// Test case 2: Comparing two elements with the same value
	l = ByteFloatList{
		{Value: 3.5},
		{Value: 3.5},
	}
	if l.Less(0, 1) {
		t.Errorf("Expected l[0] to be equal to l[1]")
	}

	// Test case 3: Comparing two elements with negative values
	l = ByteFloatList{
		{Value: -1.5},
		{Value: -2.5},
	}
	if !l.Less(1, 0) {
		t.Errorf("Expected l[1] to be less than l[0]")
	}
}

func TestByteFloatList_Swap(t *testing.T) {
	// Test case 1: swapping elements at index 0 and index 1
	l := ByteFloatList{
		{Value: 1.1},
		{Value: 2.2},
		{Value: 3.3},
	}
	expected := ByteFloatList{
		{Value: 2.2},
		{Value: 1.1},
		{Value: 3.3},
	}
	l.Swap(0, 1)
	if !reflect.DeepEqual(l, expected) {
		t.Errorf("Expected %v, but got %v", expected, l)
	}

	// Test case 2: swapping elements at index 1 and index 2
	l = ByteFloatList{
		{Value: 1.1},
		{Value: 2.2},
		{Value: 3.3},
	}
	expected = ByteFloatList{
		{Value: 1.1},
		{Value: 3.3},
		{Value: 2.2},
	}
	l.Swap(1, 2)
	if !reflect.DeepEqual(l, expected) {
		t.Errorf("Expected %v, but got %v", expected, l)
	}

	// Test case 3: swapping elements at index 0 and index 2
	l = ByteFloatList{
		{Value: 1.1},
		{Value: 2.2},
		{Value: 3.3},
	}
	expected = ByteFloatList{
		{Value: 3.3},
		{Value: 2.2},
		{Value: 1.1},
	}
	l.Swap(0, 2)
	if !reflect.DeepEqual(l, expected) {
		t.Errorf("Expected %v, but got %v", expected, l)
	}
}

func TestRankByteFloatMap(t *testing.T) {
	data := map[byte]float64{
		'a': 1.0,
		'b': 2.0,
		'c': 3.0,
	}

	expected := ByteFloatList{
		{Key: 'c', Value: 3.0},
		{Key: 'b', Value: 2.0},
		{Key: 'a', Value: 1.0},
	}

	result := RankByteFloatMap(data)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}

	data = map[byte]float64{}

	expected = ByteFloatList{}

	result = RankByteFloatMap(data)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
