package cli

import "github.com/rs/zerolog/log"

var logger = log.With().Str("module", "application").Logger()
