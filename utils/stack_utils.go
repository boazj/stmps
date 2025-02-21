package utils

import (
	"runtime"
	"strings"
)

func Funcinfo(skip int) (name, file string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if ok {
		name = runtime.FuncForPC(pc).Name()
		i := strings.Index(file, "stmps")
		if i > -1 {
			file = file[i:]
		}
		i = strings.Index(name, "stmps")
		if i > -1 {
			name = name[i:]
		}
	}
	return name, file, line
}

func Funcname(skip int) string {
	name, _, _ := Funcinfo(skip + 1)
	return name
}

func FuncnameOnly(skip int) string {
	name := Funcname(skip + 1)
	i := strings.LastIndex(name, ".")
	if i > -1 {
		return name[i+1:]
	} else {
		return name
	}
}
