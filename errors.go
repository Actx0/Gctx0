// Copyright 2026 Actx0. All rights reserved.
// License can be found in the LICENSE file.

package gctx0

import (
	"fmt"
)

// APIError is returned when the Actx0 API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("actx0 api error: status=%d body=%v", e.StatusCode, e.Body)
}
