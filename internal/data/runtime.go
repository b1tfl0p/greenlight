package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

// MarshalJSON implements the [encoding/json.Marshaler] interface.
// The runtime is a quoted string in the format "<runtime> mins".
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}
