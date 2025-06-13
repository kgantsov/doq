import { Queue } from "../types/queues";

export const getQueues = async () => {
  const response = await fetch("/API/v1/queues");
  return await response.json();
};

export const getQueue = async (name: string): Promise<Queue> => {
  const response = await fetch(`/API/v1/queues/${name}`);
  const data = await response.json();

  return data;
};

export const createQueue = async ({
  name,
  type,
  maxUnacked = 0,
  strategy = "weighted",
}: {
  name: string;
  type: string;
  maxUnacked?: number;
  strategy?: string;
}) => {
  let settings = {};

  if (type === "fair") {
    settings = {
      max_unacked: maxUnacked,
      strategy: strategy,
    };
  }
  await fetch("/API/v1/queues", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name, type, settings }),
  });
};

export const deleteQueue = async ({ name }: { name: string }) => {
  await fetch(`/API/v1/queues/${name}`, {
    method: "DELETE",
  });
};
