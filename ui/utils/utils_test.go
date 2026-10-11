package utils

import (
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func Test_ModifySizeTableDriven(t *testing.T) {
	for _, test := range []struct {
		name           string
		width, height  SizeI
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "ratio scales and truncates",
			width:          SizeRatio(0.5),
			height:         SizeRatio(0.25),
			expectedWidth:  40,
			expectedHeight: 7,
		},
		{
			name:           "full ratio keeps size",
			width:          SizeRatio(1),
			height:         SizeRatio(1),
			expectedWidth:  80,
			expectedHeight: 30,
		},
		{
			name:           "fixed ignores terminal size",
			width:          SizeFixed(10),
			height:         SizeFixed(1),
			expectedWidth:  10,
			expectedHeight: 1,
		},
		{
			name:           "modifier adjusts by amount",
			width:          SizeModifier(4),
			height:         SizeModifier(-17),
			expectedWidth:  84,
			expectedHeight: 13,
		},
		{
			name:           "width and height are independent",
			width:          SizeRatio(1),
			height:         SizeModifier(-1),
			expectedWidth:  80,
			expectedHeight: 29,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			size := ModifySize(tea.WindowSizeMsg{Width: 80, Height: 30}, test.width, test.height)

			testutil.Equal(t, test.expectedWidth, size.Width)
			testutil.Equal(t, test.expectedHeight, size.Height)
		})
	}
}
