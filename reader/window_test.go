package reader

import (
	"fmt"
	"strings"
	"testing"
)

// pushN - pushes lines "0".."n-1" so each line's text equals its line number.
func pushN(lb *LiveBuffer, n int) {
	for i := range n {
		lb.Push(fmt.Sprint(i))
	}
}

func Test_WindowFollowing(t *testing.T) {
	for _, test := range []struct {
		name     string
		capacity int
		height   int
		pushed   int
		expected string
	}{
		{name: "empty", capacity: 10, height: 3, pushed: 0, expected: ""},
		{name: "fewer lines than height", capacity: 10, height: 3, pushed: 2, expected: "0\n1"},
		{name: "shows newest lines", capacity: 10, height: 3, pushed: 7, expected: "4\n5\n6"},
		{name: "keeps following after wrap", capacity: 4, height: 3, pushed: 9, expected: "6\n7\n8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lb := NewLiveBuffer(test.capacity)
			w := NewWindow(lb)
			w.SetHeight(test.height)
			pushN(lb, test.pushed)

			Equal(t, true, w.Following())
			Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowPinned(t *testing.T) {
	for _, test := range []struct {
		name      string
		pushed    int
		moves     []int
		moved     int // return value of the last move
		pushAfter int
		following bool
		expected  string
	}{
		{name: "scroll up pins", pushed: 10, moves: []int{-2}, moved: -2, following: false, expected: "5\n6\n7"},
		{name: "pinned ignores new lines", pushed: 10, moves: []int{-2}, moved: -2, pushAfter: 5, following: false, expected: "5\n6\n7"},
		{name: "scroll down while following is a no-op", pushed: 10, moves: []int{3}, moved: 0, following: true, expected: "7\n8\n9"},
		{name: "scroll up stops at oldest", pushed: 10, moves: []int{-50}, moved: -7, following: false, expected: "0\n1\n2"},
		{name: "scroll down within pinned", pushed: 10, moves: []int{-5, 2}, moved: 2, following: false, expected: "4\n5\n6"},
		{name: "scroll down to newest re-follows", pushed: 10, moves: []int{-5, 9}, moved: 5, pushAfter: 2, following: true, expected: "9\n10\n11"},
		{name: "fewer lines than height cannot scroll", pushed: 2, moves: []int{-1}, moved: 0, following: true, expected: "0\n1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lb := NewLiveBuffer(20)
			w := NewWindow(lb)
			w.SetHeight(3)
			pushN(lb, test.pushed)

			moved := 0
			for _, n := range test.moves {
				moved = w.MoveBy(n)
			}
			for i := range test.pushAfter {
				lb.Push(fmt.Sprint(test.pushed + i))
			}

			Equal(t, test.moved, moved)
			Equal(t, test.following, w.Following())
			Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowClamp(t *testing.T) {
	for _, test := range []struct {
		name      string
		moves     []int // applied after the eviction
		moved     int
		following bool
		expected  string
	}{
		{name: "evicted window shifts to oldest held", following: false, expected: "2\n3"},
		{name: "next move starts from clamped position", moves: []int{1}, moved: 1, following: false, expected: "3\n4"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lb := NewLiveBuffer(5)
			w := NewWindow(lb)
			w.SetHeight(2)
			pushN(lb, 5) // holds 0..4
			w.MoveBy(-3) // pinned on 0,1
			lb.Push("5") // evicts 0
			lb.Push("6") // evicts 1

			moved := 0
			for _, n := range test.moves {
				moved = w.MoveBy(n)
			}

			Equal(t, test.moved, moved)
			Equal(t, test.following, w.Following())
			Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowMoveToEnd(t *testing.T) {
	lb := NewLiveBuffer(20)
	w := NewWindow(lb)
	w.SetHeight(3)
	pushN(lb, 10)
	w.MoveBy(-4)

	w.MoveToEnd()
	lb.Push("10")

	Equal(t, true, w.Following())
	Equal(t, "8\n9\n10", strings.Join(w.Lines(), "\n"))
}
