import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import ServerList from "../ServerList";
import { Toaster } from "../ui/toaster";
import { getServers, leaveCluster, transferLeadership } from "../../api/servers";

vi.mock("../../api/servers", () => ({
  getServers: vi.fn(),
  leaveCluster: vi.fn(),
  transferLeadership: vi.fn(),
}));

const servers = {
  servers: [
    { id: "node-1", addr: "127.0.0.1:7000", is_leader: true, suffrage: "Voter" },
    { id: "node-2", addr: "127.0.0.1:7001", is_leader: false, suffrage: "Voter" },
  ],
};

describe("ServerList", () => {
  beforeEach(() => {
    vi.mocked(getServers).mockReset().mockResolvedValue(servers);
    vi.mocked(leaveCluster).mockReset();
    vi.mocked(transferLeadership).mockReset();
  });

  it("confirms before removing a server, then shows a success toast", async () => {
    vi.mocked(leaveCluster).mockResolvedValueOnce(undefined);

    renderWithProviders(
      <>
        <ServerList />
        <Toaster />
      </>
    );

    const leaveButtons = await screen.findAllByRole("button", {
      name: "Leave cluster",
    });

    // Dialog isn't open yet - mutation shouldn't be reachable.
    expect(screen.queryByRole("button", { name: "Leave" })).not.toBeInTheDocument();

    await userEvent.click(leaveButtons[1]); // node-2, the non-leader row
    await userEvent.click(await screen.findByRole("button", { name: "Leave" }));

    expect(leaveCluster).toHaveBeenCalledWith({ serverId: "node-2" });
    expect(
      await screen.findByText("Server left the cluster.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Leave cluster?")).not.toBeInTheDocument();
  });

  it("shows an error toast when leaving the cluster fails", async () => {
    vi.mocked(leaveCluster).mockRejectedValueOnce(new Error("not the leader"));

    renderWithProviders(
      <>
        <ServerList />
        <Toaster />
      </>
    );

    const leaveButtons = await screen.findAllByRole("button", {
      name: "Leave cluster",
    });
    await userEvent.click(leaveButtons[1]);
    await userEvent.click(await screen.findByRole("button", { name: "Leave" }));

    expect(
      await screen.findByText("Failed to leave the cluster.")
    ).toBeInTheDocument();
    expect(screen.getByText("not the leader")).toBeInTheDocument();
  });

  it("only offers leadership transfer on the leader row, and confirms before transferring", async () => {
    vi.mocked(transferLeadership).mockResolvedValueOnce(undefined);

    renderWithProviders(
      <>
        <ServerList />
        <Toaster />
      </>
    );

    const transferButtons = await screen.findAllByRole("button", {
      name: "Transfer leadership",
    });
    expect(transferButtons).toHaveLength(1); // only node-1 is leader

    await userEvent.click(transferButtons[0]);
    await userEvent.click(await screen.findByRole("button", { name: "Transfer" }));

    expect(transferLeadership).toHaveBeenCalledTimes(1);
    expect(
      await screen.findByText("Leadership transferred.")
    ).toBeInTheDocument();
  });
});
