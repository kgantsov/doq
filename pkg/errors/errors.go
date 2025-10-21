package errors

import "fmt"

var ErrQueueNotFound = fmt.Errorf("queue not found")
var ErrEmptyQueue = fmt.Errorf("queue is empty")
var ErrMessageNotFound = fmt.Errorf("message not found")

var ErrInvalidStrategy = fmt.Errorf("invalid strategy")
var ErrInvalidAckTimeout = fmt.Errorf("invalid ack timeout")
var ErrInvalidMaxUnacked = fmt.Errorf("invalid max unacked")
var ErrInvalidQueueSettings = fmt.Errorf("invalid queue settings")
