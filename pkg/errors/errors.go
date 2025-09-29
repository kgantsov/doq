package errors

import "fmt"

var ErrQueueNotFound = fmt.Errorf("queue not found")
var ErrEmptyQueue = fmt.Errorf("queue is empty")
var ErrMessageNotFound = fmt.Errorf("message not found")

var ErrInvalidAckTimeout = fmt.Errorf("invalid ack timeout")
