package settings

import (
	"math"
	"reflect"
	"testing"
)

func TestMetadataIntegerLimitFitsItsWireType(t *testing.T) {
	const safeJSONInteger = int64(9007199254740991)
	for _, tc := range []struct {
		value any
		upper int64
		want  int64
	}{
		{int32(0), safeJSONInteger, math.MaxInt32},
		{int32(0), 65535, 65535},
		{int64(0), safeJSONInteger, safeJSONInteger},
		{int(0), safeJSONInteger, min(safeJSONInteger, int64(math.MaxInt))},
	} {
		field := reflect.ValueOf(tc.value)
		got := settingIntegerLimit(field, tc.upper)
		if got != tc.want || field.OverflowInt(got) {
			t.Errorf("%s: limit %d is not its representable bound %d", field.Type(), got, tc.want)
		}
	}
}
