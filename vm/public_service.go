// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	log "github.com/inconshreveable/log15"

	"github.com/ava-labs/quarkvm/chain"
	"github.com/ava-labs/quarkvm/parser"
)

var (
	ErrPoWFailed      = errors.New("PoW failed")
	ErrInvalidEmptyTx = errors.New("invalid empty transaction")
)

type PublicService struct {
	vm *VM
}

type PingReply struct {
	Success bool `serialize:"true" json:"success"`
}

func (svc *PublicService) Ping(_ *http.Request, _ *struct{}, reply *PingReply) (err error) {
	log.Info("ping")
	reply.Success = true
	return nil
}

type GenesisReply struct {
	Genesis *chain.Genesis `serialize:"true" json:"genesis"`
}

func (svc *PublicService) Genesis(_ *http.Request, _ *struct{}, reply *GenesisReply) (err error) {
	reply.Genesis = svc.vm.Genesis()
	return nil
}

type IssueTxArgs struct {
	Tx []byte `serialize:"true" json:"tx"`
}

type IssueTxReply struct {
	TxID    ids.ID `serialize:"true" json:"txId"`
	Success bool   `serialize:"true" json:"success"`
}

func (svc *PublicService) IssueTx(_ *http.Request, args *IssueTxArgs, reply *IssueTxReply) error {
	tx := new(chain.Transaction)
	if _, err := chain.Unmarshal(args.Tx, tx); err != nil {
		return err
	}

	// otherwise, unexported tx.id field is empty
	if err := tx.Init(svc.vm.genesis); err != nil {
		reply.Success = false
		return err
	}
	reply.TxID = tx.ID()

	errs := svc.vm.Submit(tx)
	reply.Success = len(errs) == 0
	if reply.Success {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("%v", errs)
}

type CheckTxArgs struct {
	TxID ids.ID `serialize:"true" json:"txId"`
}

type CheckTxReply struct {
	Confirmed bool `serialize:"true" json:"confirmed"`
}

func (svc *PublicService) CheckTx(_ *http.Request, args *CheckTxArgs, reply *CheckTxReply) error {
	has, err := chain.HasTransaction(svc.vm.db, args.TxID)
	if err != nil {
		return err
	}
	reply.Confirmed = has
	return nil
}

type LastAcceptedReply struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

func (svc *PublicService) LastAccepted(_ *http.Request, _ *struct{}, reply *LastAcceptedReply) error {
	reply.BlockID = svc.vm.lastAccepted.ID()
	return nil
}

type ValidBlockIDArgs struct {
	BlockID ids.ID `serialize:"true" json:"blockId"`
}

type ValidBlockIDReply struct {
	Valid bool `serialize:"true" json:"valid"`
}

func (svc *PublicService) ValidBlockID(_ *http.Request, args *ValidBlockIDArgs, reply *ValidBlockIDReply) error {
	valid, err := svc.vm.ValidBlockID(args.BlockID)
	if err != nil {
		return err
	}
	reply.Valid = valid
	return nil
}

type DifficultyEstimateArgs struct{}

type DifficultyEstimateReply struct {
	Difficulty uint64 `serialize:"true" json:"difficulty"`
	Cost       uint64 `serialize:"true" json:"cost"`
}

func (svc *PublicService) DifficultyEstimate(
	_ *http.Request,
	_ *DifficultyEstimateArgs,
	reply *DifficultyEstimateReply,
) error {
	diff, cost, err := svc.vm.DifficultyEstimate()
	if err != nil {
		return err
	}
	reply.Difficulty = diff
	reply.Cost = cost
	return nil
}

type PrefixInfoArgs struct {
	Prefix []byte `serialize:"true" json:"prefix"`
}

type PrefixInfoReply struct {
	Info *chain.PrefixInfo `serialize:"true" json:"info"`
}

func (svc *PublicService) PrefixInfo(_ *http.Request, args *PrefixInfoArgs, reply *PrefixInfoReply) error {
	i, _, err := chain.GetPrefixInfo(svc.vm.db, args.Prefix)
	if err != nil {
		return err
	}
	reply.Info = i
	return nil
}

type RangeArgs struct {
	// Prefix is the namespace for the "PrefixInfo"
	// whose owner can write and read value for the
	// specific key space.
	// Assume the client pre-processes the inputs so that
	// all prefix must have the delimiter '/' as suffix.
	Prefix []byte `serialize:"true" json:"prefix"`

	// Key is parsed from the given input, with its prefix removed.
	// Optional for claim/lifeline transactions.
	// Non-empty to claim a key-value pair.
	Key []byte `serialize:"true" json:"key"`

	// RangeEnd is optional, and only non-empty for range query transactions.
	RangeEnd []byte `serialize:"true" json:"rangeEnd"`

	// Limit limits the number of key-value pairs in the response.
	Limit uint32 `serialize:"true" json:"limit"`
}

type RangeReply struct {
	KeyValues []chain.KeyValue `serialize:"true" json:"keyValues"`
}

func (svc *PublicService) Range(_ *http.Request, args *RangeArgs, reply *RangeReply) (err error) {
	log.Debug("range query", "key", string(args.Key), "rangeEnd", string(args.RangeEnd))
	opts := make([]chain.OpOption, 0)
	if len(args.RangeEnd) > 0 {
		opts = append(opts, chain.WithRangeEnd(args.RangeEnd))
	}
	if args.Limit > 0 {
		opts = append(opts, chain.WithRangeLimit(args.Limit))
	}
	kvs, err := chain.Range(svc.vm.db, args.Prefix, args.Key, opts...)
	if err != nil {
		return err
	}
	reply.KeyValues = kvs
	return nil
}

type ResolveArgs struct {
	Path string `serialize:"true" json:"path"`
}

type ResolveReply struct {
	Exists bool   `serialize:"true" json:"exists"`
	Value  []byte `serialize:"true" json:"value"`
}

func (svc *PublicService) Resolve(_ *http.Request, args *ResolveArgs, reply *ResolveReply) error {
	pfx, key, _, err := parser.ParsePrefixKey(
		[]byte(args.Path),
		parser.WithCheckPrefix(),
		parser.WithCheckKey(),
	)
	if err != nil {
		return err
	}

	v, exists, err := chain.GetValue(svc.vm.db, pfx, key)
	reply.Exists = exists
	reply.Value = v
	return err
}
