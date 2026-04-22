package errors

import (
	"strconv"
	"strings"
)

// uncaughtPanic marks an error as originating from a recovered panic.
type uncaughtPanic struct{ message string }

func (p uncaughtPanic) Error() string {
	return p.message
}

// ParsePanic allows you to get an *Error from the output of a Go program
// that panicked. This is particularly useful with
// https://github.com/mitchellh/panicwrap.
const maxPanicInput = 1 << 20
const maxPanicFrames = 256

func ParsePanic(text string) (*Error, error) {
	if len(text) > maxPanicInput {
		return nil, Errorf("errors.panicParser: input too large (%d bytes)", len(text))
	}
	lines := strings.Split(text, "\n")

	state := "start"

	var message string
	stack := make([]StackFrame, 0, 8)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if state == "start" {
			if strings.HasPrefix(line, "panic: ") {
				message = strings.TrimPrefix(line, "panic: ")
				state = "seek"
			} else {
				return nil, Errorf("errors.panicParser: Invalid line (no prefix): %s", line)
			}

		} else if state == "seek" {
			if strings.HasPrefix(line, "goroutine ") && strings.HasSuffix(line, "[running]:") {
				state = "parsing"
			}

		} else if state == "parsing" {
			if line == "" {
				state = "done"
				break
			}
			createdBy := false
			if strings.HasPrefix(line, "created by ") {
				line = strings.TrimPrefix(line, "created by ")
				createdBy = true
			}

			i++

			if i >= len(lines) {
				return nil, Errorf("errors.panicParser: Invalid line (unpaired): %s", line)
			}

			frame, err := parsePanicFrame(line, lines[i], createdBy)
			if err != nil {
				return nil, err
			}

			stack = append(stack, *frame)
			if len(stack) >= maxPanicFrames {
				state = "done"
				break
			}
			if createdBy {
				state = "done"
				break
			}
		}
	}

	if state == "done" || state == "parsing" {
		return &Error{Err: uncaughtPanic{message}, data: &errorData{frames: stack}}, nil
	}
	return nil, Errorf("could not parse panic: %v", text)
}

// parsePanicFrame parses two lines from a panic stack trace into a StackFrame.
//
// The lines look like:
//
//	main.(*foo).destruct(0xc208067e98)
//	        /0/go/src/github.com/bugsnag/bugsnag-go/pan/main.go:22 +0x151
func parsePanicFrame(name string, line string, createdBy bool) (*StackFrame, error) {
	idx := strings.LastIndex(name, "(")
	if idx == -1 && !createdBy {
		return nil, Errorf("errors.panicParser: Invalid line (no call): %s", name)
	}
	if idx != -1 {
		name = name[:idx]
	}
	pkg := ""

	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.ReplaceAll(name, "·", ".")

	if !strings.HasPrefix(line, "\t") {
		return nil, Errorf("errors.panicParser: Invalid line (no tab): %s", line)
	}

	idx = strings.LastIndex(line, ":")
	if idx == -1 {
		return nil, Errorf("errors.panicParser: Invalid line (no line number): %s", line)
	}
	file := line[1:idx]

	number := line[idx+1:]
	if idx = strings.Index(number, " +"); idx > -1 {
		number = number[:idx]
	}

	lno, err := strconv.ParseInt(number, 10, 32)
	if err != nil {
		return nil, Errorf("errors.panicParser: Invalid line (bad line number): %s", line)
	}

	return &StackFrame{
		File:       file,
		LineNumber: int(lno),
		Package:    pkg,
		Name:       name,
	}, nil
}
