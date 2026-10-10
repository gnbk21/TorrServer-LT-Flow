package log

import "io"

// Guard the sink as well as our convenience functions: the native alert pump
// and inherited components also write through Go's standard logger.
type redactingWriter struct{ destination io.Writer }

func (w redactingWriter) Write(input []byte) (int, error) {
	output := []byte(RedactSecrets(string(input)))
	n, err := w.destination.Write(output)
	if err != nil {
		return 0, err
	}
	if n != len(output) {
		return 0, io.ErrShortWrite
	}
	// io.Writer's count describes the caller's input, not the redacted length.
	return len(input), nil
}
