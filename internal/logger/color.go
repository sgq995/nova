package logger

// func reset(ss ...string) string {
// 	var b strings.Builder
// 	for _, s := range ss {
// 		b.WriteString(s)
// 	}
// 	b.WriteString("\033[0m")
// 	return b.String()
// }

func bold(s string) string {
	return "\033[1m" + s + "\033[0m"
}

func red(s string) string {
	return "\033[31m" + s + "\033[0m"
}

// func green(s string) string {
// 	return "\033[32m" + s + "\033[0m"
// }

func yellow(s string) string {
	return "\033[33m" + s + "\033[0m"
}

func blue(s string) string {
	return "\033[34m" + s + "\033[0m"
}

// func magenta(s string) string {
// 	return "\033[35m" + s + "\033[0m"
// }

func cyan(s string) string {
	return "\033[36m" + s + "\033[0m"
}
