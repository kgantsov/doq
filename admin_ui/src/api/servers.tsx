import { ApiError } from "./errors";

export const getServers = async () => {
  const response = await fetch("/API/v1/cluster/servers");

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to get servers: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }

  return await response.json();
};

export const leaveCluster = async ({
  serverId: serverId,
}: {
  serverId: string;
}) => {
  const response = await fetch(`/API/v1/cluster/leave`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ id: serverId }),
  });

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to leave cluster: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }
};

export const transferLeadership = async () => {
  const response = await fetch(`/API/v1/cluster/transfer-leadership`, {
    method: "POST",
  });

  if (!response.ok) {
    let errorDetails;
    try {
      errorDetails = await response.json();
    } catch {
      errorDetails = null;
    }

    throw new ApiError(
      `Failed to transfer leadership: ${response.statusText}`,
      response.status,
      errorDetails
    );
  }
};
