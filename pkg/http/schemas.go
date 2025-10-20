package http

import "github.com/danielgtaylor/huma/v2"

type JoinInput struct {
	Body struct {
		ID   string `json:"id" example:"node-0" doc:"ID of a node"`
		Addr string `json:"addr" example:"localhost:12001" doc:"IP address and a port of a service"`
	}
}

type JoinOutputBody struct {
	ID   string `json:"id" example:"node-0" doc:"ID of a node"`
	Addr string `json:"addr" example:"localhost:12001" doc:"IP address and a port of a service"`
}

type JoinOutput struct {
	Body JoinOutputBody
}

type LeaveInput struct {
	Body struct {
		ID string `json:"id" example:"node-0" doc:"ID of a node"`
	}
}

type LeaveOutputBody struct {
	ID string `json:"id" example:"node-0" doc:"ID of a node"`
}

type LeaveOutput struct {
	Body LeaveOutputBody
}

type ServersInput struct {
}

type Server struct {
	Id       string `json:"id" example:"node-0" doc:"ID of a node"`
	Addr     string `json:"addr" example:"localhost:12001" doc:"IP address and a port of a service"`
	IsLeader bool   `json:"is_leader" example:"true" doc:"Indicates if the server is leader or not"`
	Suffrage string `json:"suffrage" example:"full" doc:"Determines whether the server gets a vote"`
}

type ServersOutputBody struct {
	Servers []Server `json:"servers" doc:"List of servers"`
}

type ServersOutput struct {
	Body ServersOutputBody
}

type BackupInput struct {
	Body struct {
		Since uint64 `json:"since" example:"0" doc:"Minimum version of the log to include in the backup"`
	}
}

type RestoreInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"application/octet-stream" required:"true"`
	}]
}

type RestoreOutputBody struct {
	Status string `json:"status" example:"SUCCEDED" doc:"Status of the restore operation"`
}

type RestoreOutput struct {
	Status int
	Body   RestoreOutputBody
}

type QueueSettings struct {
	Strategy   string `json:"strategy,omitempty" enum:"round_robin,weighted" example:"round_robin" doc:"Strategy for message distribution among consumers"`
	MaxUnacked int    `json:"max_unacked,omitempty" minimum:"0" example:"7" doc:"Maximum expected unacknowledged messages in the queue"`
	AckTimeout uint32 `json:"ack_timeout,omitempty" minimum:"60" example:"600" doc:"Time in seconds to wait for an acknowledgment before re-enqueuing the message"`
}

type CreateQueueInputBody struct {
	Name     string        `json:"name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Type     string        `json:"type" enum:"delayed,fair" example:"fair" doc:"Type of the queue"`
	Settings QueueSettings `json:"settings" doc:"Configuration of the queue"`
}

type CreateQueueInput struct {
	Body CreateQueueInputBody
}

type CreateQueueOutputBody struct {
	Status   string        `json:"status" example:"CREATED" doc:"Status of the create operation"`
	Name     string        `json:"name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Type     string        `json:"type" enum:"delayed,fair" example:"fair" doc:"Type of the queue"`
	Settings QueueSettings `json:"settings" doc:"Configuration of the queue"`
}

type CreateQueueOutput struct {
	Status int
	Body   CreateQueueOutputBody
}

type UpdateQueueInputBody struct {
	Settings QueueSettings `json:"settings" doc:"Configuration of the queue"`
}

type UpdateQueueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Body      UpdateQueueInputBody
}

type UpdateQueueOutputBody struct {
	Status   string        `json:"status" example:"UPDATED" doc:"Status of the create operation"`
	Name     string        `json:"name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Settings QueueSettings `json:"settings" doc:"Configuration of the queue"`
}

type UpdateQueueOutput struct {
	Status int
	Body   UpdateQueueOutputBody
}

type DeleteQueueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
}

type DeleteQueueOutputBody struct {
	Status string `json:"status" example:"DELETED" doc:"Status of the delete operation"`
}

type DeleteQueueOutput struct {
	Status int
	Body   DeleteQueueOutputBody
}

type EnqueueInputBody struct {
	ID       string            `json:"id,omitempty" doc:"ID of the message"`
	Group    string            `json:"group,omitempty" default:"default" example:"customer-1" doc:"Group of the message"`
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string            `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
}

type EnqueueInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Body      EnqueueInputBody
}

type EnqueueOutputBody struct {
	Status   string            `json:"status" example:"ENQUEUED" doc:"Status of the enqueue operation"`
	ID       string            `json:"id" doc:"ID of the message"`
	Group    string            `json:"group,omitempty" default:"default" example:"customer-1" doc:"Group of the message"`
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string            `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
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
	Status   string            `json:"status" example:"DEQUEUED" doc:"Status of the dequeue operation"`
	ID       string            `json:"id" doc:"ID of the message"`
	Group    string            `json:"group,omitempty" default:"default" example:"customer-1" doc:"Group of the message"`
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string            `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
}

type DequeueOutput struct {
	Status int
	Body   DequeueOutputBody
}

type GetInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
}

type GetOutputBody struct {
	Status   string            `json:"status" example:"DEQUEUED" doc:"Status of the dequeue operation"`
	ID       string            `json:"id" doc:"ID of the message"`
	Group    string            `json:"group,omitempty" default:"default" example:"customer-1" doc:"Group of the message"`
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Content  string            `json:"content" example:"{\"user_id\": 1}" doc:"Content of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
}

type GetOutput struct {
	Status int
	Body   GetOutputBody
}

type DeleteInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
}

type DeleteOutput struct {
	Status int
}

type AckInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
}

type AckOutputBody struct {
	Status string `json:"status" example:"ACKNOWLEDGED" doc:"Status of the dequeue operation"`
	ID     string `json:"id" doc:"ID of the message"`
}

type AckOutput struct {
	Status int
	Body   AckOutputBody
}

type NackInputBody struct {
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
}

type NackInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	ID        uint64 `path:"id" example:"123" doc:"ID of the message"`
	Body      NackInputBody
}

type NackOutputBody struct {
	Status   string            `json:"status" example:"UNACKNOWLEDGED" doc:"Status of the dequeue operation"`
	ID       string            `json:"id" doc:"ID of the message"`
	Priority int64             `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
	Metadata map[string]string `json:"metadata,omitempty" example:"{\"retry\": \"0\"}" doc:"Metadata of the message"`
}

type NackOutput struct {
	Status int
	Body   NackOutputBody
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
	ID       string `json:"id" doc:"ID of the message"`
	Priority int64  `json:"priority" minimum:"0" example:"60" doc:"Priority of the message"`
}
type UpdatePriorityOutput struct {
	Status int
	Body   UpdatePriorityOutputBody
}

type QueueInfoInput struct {
	QueueName string `path:"queue_name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
}

type QueueInfoOutputBody struct {
	Name       string        `json:"name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Type       string        `json:"type" enum:"delayed,fair" example:"fair" doc:"Type of the queue"`
	Settings   QueueSettings `json:"settings" doc:"Configuration of the queue"`
	EnqueueRPS float64       `json:"enqueue_rps" doc:"Rate of enqueued messages per second"`
	DequeueRPS float64       `json:"dequeue_rps" doc:"Rate of dequeued messages per second"`
	AckRPS     float64       `json:"ack_rps" doc:"Rate of acknowledged messages per second"`
	NackRPS    float64       `json:"nack_rps" doc:"Rate of unacknowledged messages per second"`
	Ready      int64         `json:"ready" doc:"Number of ready messages"`
	Unacked    int64         `json:"unacked" doc:"Number of unacknowledged messages"`
	Total      int64         `json:"total" doc:"Total number of messages"`
}

type QueueInfoOutput struct {
	Status int
	Body   QueueInfoOutputBody
}

type QueuesInput struct {
}

type QueueOutput struct {
	Name       string        `json:"name" maxLength:"1024" example:"user_indexing_queue" doc:"Name of the queue"`
	Type       string        `json:"type" enum:"delayed,fair" example:"fair" doc:"Type of the queue"`
	Settings   QueueSettings `json:"settings" doc:"Configuration of the queue"`
	EnqueueRPS float64       `json:"enqueue_rps" doc:"Rate of enqueued messages per second"`
	DequeueRPS float64       `json:"dequeue_rps" doc:"Rate of dequeued messages per second"`
	AckRPS     float64       `json:"ack_rps" doc:"Rate of acknowledged messages per second"`
	NackRPS    float64       `json:"nack_rps" doc:"Rate of unacknowledged messages per second"`
	Ready      int64         `json:"ready" doc:"Number of ready messages"`
	Unacked    int64         `json:"unacked" doc:"Number of unacknowledged messages"`
	Total      int64         `json:"total" doc:"Total number of messages"`
}

type QueuesOutputBody struct {
	Queues []QueueOutput `json:"queues"`
}

type QueuesOutput struct {
	Status int
	Body   QueuesOutputBody
}

type GenerateIdInputBody struct {
	Number int `json:"number" minimum:"1" maximum:"1000" example:"1" doc:"Number of IDs to generate"`
}

type GenerateIdInput struct {
	Body GenerateIdInputBody
}

type GenerateIdOutputBody struct {
	IDs []string `json:"ids" doc:"List of generated IDs"`
}

type GenerateIdOutput struct {
	Status int
	Body   GenerateIdOutputBody
}
