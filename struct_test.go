package main

import (
	"reflect"
	"testing"
)

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
