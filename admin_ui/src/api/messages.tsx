import { Message } from "../types/messages";

export type EnqueueRequest = {
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
      group,
      priority,
      content,
    }),
  });
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
