// Copyright 2023 The STMPS Authors
// Copyright 2023 Drew Weymouth and contributors, zackslash
// SPDX-License-Identifier: GPL-3.0-only

//go:build !darwin

package remote

import (
	"errors"

	"github.com/spezifisch/stmps/logger"
)

func RegisterMPMediaHandler(_ ControlledPlayer, _ logger.Logger) error {
	// MPMediaHandler only supports macOS.
	return errors.New("unsupported platform")
}
