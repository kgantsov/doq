import { Message } from "../types/messages";

export type EnqueueRequest = {
  id: string;
  queueName: string;
  group: string;
  priority: number;
  content: string;
};

export type DequeueRequest = {
  queueName: string;
  ack: boolean;
};

export const EnqueueMessage = async ({
  id,
  queueName,
  group,
  priority,
  content,
}: EnqueueRequest): Promise<Message> => {
  const response = await fetch(`/API/v1/queues/${queueName}/messages`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json, application/problem+json",
    },
    body: JSON.stringify({
      id,
      group,
      priority,
      content,
    }),
  });
  if (!response.ok) {
    throw new Error(`Failed to enqueue message: ${response.statusText}`);
  }
  return await response.json();
};

export const DequeueMessage = async ({
  queueName,
  ack,
}: DequeueRequest): Promise<Message> => {
  const response = await fetch(
    `/API/v1/queues/${queueName}/messages?ack=${ack}`,
    {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json, application/problem+json",
      },
    }
  );
  return await response.json();
};

export const NackMessage = async (queueName: string, id: string) => {
  const response = await fetch(
    `/API/v1/queues/${queueName}/messages/${id}/nack`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json, application/problem+json",
      },
      body: JSON.stringify({
        priority: 0,
      }),
    }
  );
  return await response.json();
};
