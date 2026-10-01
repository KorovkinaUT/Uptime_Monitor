package domain

import "errors"

var (
	ErrMonitorNotFound     = errors.New("monitor not found")
	ErrInvalidMonitor      = errors.New("invalid monitor")
	ErrCheckAlreadyRunning = errors.New("check already running")
	ErrInvalidLimit        = errors.New("invalid limit")
)
