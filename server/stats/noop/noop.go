// Package noop provides a stats impl that does nothing.
// nolint:revive
package noop

import (
	"context"

	"github.com/charmbracelet/charm/server/stats"
)

// Stats is a stats implementation that does nothing.
type Stats struct{}

var _ stats.Stats = Stats{}

func (Stats) APILinkGen()                      {}
func (Stats) APILinkRequest()                  {}
func (Stats) APIUnlink()                       {}
func (Stats) APIAuth()                         {}
func (Stats) APIKeys()                         {}
func (Stats) LinkGen()                         {}
func (Stats) LinkRequest()                     {}
func (Stats) Keys()                            {}
func (Stats) ID()                              {}
func (Stats) JWT()                             {}
func (Stats) GetUserByID()                     {}
func (Stats) GetUser()                         {}
func (Stats) SetUserName()                     {}
func (Stats) GetNewsList()                     {}
func (Stats) GetNews()                         {}
func (Stats) PostNews()                        {}
func (Stats) FSFileRead(_ string, _ int64)     {}
func (Stats) FSFileWritten(_ string, _ int64)  {}
func (Stats) Start() error                     { return nil }
func (Stats) Close() error                     { return nil }
func (Stats) Shutdown(_ context.Context) error { return nil }
