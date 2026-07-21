package event

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type Field struct {
	Key   string
	Value string
}

func F(key, value string) Field { return Field{Key: key, Value: value} }

type Logger struct {
	Out io.Writer
	Now func() time.Time
}

func (l Logger) Emit(eventName string, fields ...Field) error {
	now := time.Now
	if l.Now != nil {
		now = l.Now
	}
	all := append([]Field{{Key: "ts", Value: now().UTC().Format(time.RFC3339)}, {Key: "event", Value: eventName}}, fields...)
	var parts []string
	for _, field := range all {
		if !keyPattern.MatchString(field.Key) {
			return fmt.Errorf("invalid event key %q", field.Key)
		}
		if field.Value == "" || strings.ContainsAny(field.Value, " \t\r\n=") {
			return fmt.Errorf("invalid event value for %s", field.Key)
		}
		parts = append(parts, field.Key+"="+field.Value)
	}
	_, err := fmt.Fprintln(l.Out, strings.Join(parts, " "))
	return err
}
