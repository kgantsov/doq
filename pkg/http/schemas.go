package http

type EnqueueInputBody struct {
	Priority int    `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}

type EnqueueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Body      EnqueueInputBody
}

type EnqueueOutputBody struct {
	Status   string `json:"status" example:"ENQUEUED" doc:"Status of the enqueue operation"`
	ID       uint64 `json:"id" doc:"ID of the message"`
	Priority int    `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}
type EnqueueOutput struct {
	Status int
	Body   EnqueueOutputBody
}

type DequeueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
}

type DequeueOutputBody struct {
	Status   string `json:"status" example:"DEQUEUED" doc:"Status of the dequeue operation"`
	ID       uint64 `json:"id" doc:"ID of the message"`
	Priority int    `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
}

type DequeueOutput struct {
	Status int
	Body   DequeueOutputBody
}
