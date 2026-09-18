package printer

import (
	"fmt"

	"github.com/fatih/color"
)

// ColorPrinter formats human-readable CLI messages.
type ColorPrinter struct {
	color bool
}

func NewColorPrinter(enabled bool) *ColorPrinter {
	return &ColorPrinter{color: enabled}
}

func (p *ColorPrinter) Success(format string, args ...any) string {
	return p.format(color.FgGreen, format, args...)
}

func (p *ColorPrinter) Error(format string, args ...any) string {
	return p.format(color.FgRed, format, args...)
}

func (p *ColorPrinter) Warning(format string, args ...any) string {
	return p.format(color.FgYellow, format, args...)
}

func (p *ColorPrinter) Info(format string, args ...any) string {
	return p.format(color.FgBlue, format, args...)
}

func (p *ColorPrinter) Debug(format string, args ...any) string {
	return p.format(color.FgCyan, format, args...)
}

func (p *ColorPrinter) format(attribute color.Attribute, format string, args ...any) string {
	if !p.color {
		return fmt.Sprintf(format, args...)
	}
	return color.New(attribute).Sprintf(format, args...)
}
