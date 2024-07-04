// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package services

import (
	"context"

	"github.com/ledgerwatch/erigon/cmd/devnet/devnet"
	"github.com/ledgerwatch/erigon/cmd/devnet/services/accounts"
	"github.com/ledgerwatch/erigon/cmd/devnet/services/polygon"
)

type ctxKey int

const (
	ckFaucet ctxKey = iota
)

func Faucet(ctx context.Context) *accounts.Faucet {
	if network := devnet.CurrentNetwork(ctx); network != nil {
		for _, service := range network.Services {
			if faucet, ok := service.(*accounts.Faucet); ok {
				return faucet
			}
		}
	}

	return nil
}

func Heimdall(ctx context.Context) *polygon.Heimdall {
	if network := devnet.CurrentNetwork(ctx); network != nil {
		for _, service := range network.Services {
			if heimdall, ok := service.(*polygon.Heimdall); ok {
				return heimdall
			}
		}
	}

	return nil
}

func ProofGenerator(ctx context.Context) *polygon.ProofGenerator {
	if network := devnet.CurrentNetwork(ctx); network != nil {
		for _, service := range network.Services {
			if proofGenerator, ok := service.(*polygon.ProofGenerator); ok {
				return proofGenerator
			}
		}
	}

	return nil
}
