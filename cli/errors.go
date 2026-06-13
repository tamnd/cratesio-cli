package cli

import (
	"errors"

	"github.com/tamnd/cratesio-cli/cratesio"
)

func isNotFound(err error) bool {
	return errors.Is(err, cratesio.ErrNotFound)
}
