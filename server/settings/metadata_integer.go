package settings

import "reflect"

func settingIntegerLimit(field reflect.Value, upper int64) int64 {
	// JSON decodes into the actual Go field before settings validation. An int
	// field on a 32-bit target cannot accept the 64-bit registry's rate ceiling.
	if bits := field.Type().Bits(); bits < 64 {
		return min(upper, (int64(1)<<(bits-1))-1)
	}
	return upper
}
