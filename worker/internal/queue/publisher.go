package queue

import sharedqueue "vedio/shared/queue"

// Publisher is an alias to the shared publisher implementation.
type Publisher = sharedqueue.Publisher

// NewPublisher creates a new publisher using the shared implementation.
var NewPublisher = sharedqueue.NewPublisher
