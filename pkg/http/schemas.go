package http

type EnqueueInputBody struct {
	Priority int64  `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}

type EnqueueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Body      EnqueueInputBody
}

type EnqueueOutputBody struct {
	Status   string `json:"status" example:"ENQUEUED" doc:"Status of the enqueue operation"`
	ID       uint64 `json:"id" doc:"ID of the message"`
	Priority int64  `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}

type EnqueueOutput struct {
	Status int
	Body   EnqueueOutputBody
}

type DequeueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Ack       bool   `query:"ack" example:"true" doc:"Acknowledge the message"`
}

type DequeueOutputBody struct {
	Status   string `json:"status" example:"DEQUEUED" doc:"Status of the dequeue operation"`
	ID       uint64 `json:"id" doc:"ID of the message"`
	Priority int64  `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}

type DequeueOutput struct {
	Status int
	Body   DequeueOutputBody
}

type AckInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
}

type AckOutputBody struct {
	Status string `json:"status" example:"DEQUEUED" doc:"Status of the dequeue operation"`
	ID     uint64 `json:"id" doc:"ID of the message"`
}

type AckOutput struct {
	Status int
	Body   AckOutputBody
}

type UpdatePriorityInputBody struct {
	Priority int64 `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
}

type UpdatePriorityInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
	Body      UpdatePriorityInputBody
}

type UpdatePriorityOutputBody struct {
	Status   string `json:"status" example:"ENQUEUED" doc:"Status of the enqueue operation"`
	ID       uint64 `json:"id" doc:"ID of the message"`
	Priority int64  `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
}
type UpdatePriorityOutput struct {
	Status int
	Body   UpdatePriorityOutputBody
}
