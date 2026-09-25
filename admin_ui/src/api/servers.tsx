const parseErrorMessage = async (response: Response) => {
  let message = `Error ${response.status}`;

  try {
    const body = await response.json();
    message = body?.errors?.[0]?.message || body?.detail || message;
  } catch (err) {
    console.error(err);
  }

  return message;
};

export const getServers = async () => {
  const response = await fetch("/API/v1/cluster/servers");
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
    throw new Error(await parseErrorMessage(response));
  }
};

export const transferLeadership = async () => {
  const response = await fetch(`/API/v1/cluster/transfer-leadership`, {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error(await parseErrorMessage(response));
  }
};
