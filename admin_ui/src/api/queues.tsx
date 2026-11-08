import { Queue } from "../types/queues";
import { ApiError } from "./errors";

export const getQueues = async () => {
  const response = await fetch("/API/v1/queues");
  return await response.json();
};

export const getQueue = async (name: string): Promise<Queue> => {
  const response = await fetch(`/API/v1/queues/${name}`);

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to get a queue: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }

  const data = await response.json();

  return data;
};

export type CreateQueueRequest = {
  name: string;
  type: string;
  maxUnacked?: number;
  ackTimeout?: number;
  strategy?: string;
  deliveryLimit?: number;
};

export const createQueue = async ({
  name,
  type,
  maxUnacked = 0,
  ackTimeout = 1800,
  strategy = "weighted",
  deliveryLimit = 0,
}: CreateQueueRequest) => {
  let settings: {
    max_unacked?: number;
    strategy?: string;
    delivery_limit?: number;
    ack_timeout?: number;
  } = {};

  if (type === "fair") {
    settings = {
      max_unacked: maxUnacked,
      strategy: strategy,
    };
  }

  settings["ack_timeout"] = ackTimeout;
  settings["delivery_limit"] = deliveryLimit;

  const response = await fetch("/API/v1/queues", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name, type, settings }),
  });

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to create a queue: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }
};

export const deleteQueue = async ({ name }: { name: string }) => {
  const response = await fetch(`/API/v1/queues/${name}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to delete a queue: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }
};
