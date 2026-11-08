import { Message } from "../types/messages";
import { ApiError } from "./errors";

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
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to enqueue message: ${response.statusText}`,
      response.status,
      errorDetails
    );
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

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to dequeue message: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }

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

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to nack message: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }

  return await response.json();
};
