export const getServers = async () => {
  const response = await fetch("/API/v1/cluster/servers");
  return await response.json();
};

export const leaveCluster = async ({
  serverId: serverId,
}: {
  serverId: string;
}) => {
  await fetch(`/API/v1/cluster/leave`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ id: serverId }),
  });
};
