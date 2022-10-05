package termui

import (
	"fmt"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/util/colors"
)

type helpBar []struct {
	keys string
	text string
}

func (hb helpBar) Render(maxX int) string {
	var builder strings.Builder
	for _, entry := range hb {
		builder.WriteString(colors.BlueBg(fmt.Sprintf("[%s] %s", entry.keys, entry.text)))
		builder.WriteByte(' ')
	}

	l := termtext.Len(builder.String())
	if l < maxX {
		builder.WriteString(colors.BlueBg(strings.Repeat(" ", maxX-l)))
	}

	return builder.String()
}
