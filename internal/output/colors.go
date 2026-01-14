package output

var (
	ColorReset     = ansiString("\033[0m")
	ColorRed       = ansiString("\033[31m")
	ColorGreen     = ansiString("\033[32m")
	ColorBlue      = ansiString("\033[34m")
	ColorCyan      = ansiString("\033[36m")
	ColorMagenta   = ansiString("\033[35m")
	ColorBold      = ansiString("\033[2m")
	ColorDimYellow = ansiString("\033[2;33m")
)
