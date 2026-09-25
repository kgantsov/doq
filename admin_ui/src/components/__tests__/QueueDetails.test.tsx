import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Provider } from "../ui/provider";
import QueueDetails from "../QueueDetails";
import { Toaster } from "../ui/toaster";
import { getQueue, deleteQueue } from "../../api/queues";

vi.mock("../../api/queues", () => ({
  getQueue: vi.fn(),
  deleteQueue: vi.fn(),
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>(
    "react-router-dom"
  );
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const queue = {
  name: "test-queue",
  type: "delayed",
  enqueue_rps: 1.2,
  dequeue_rps: 1.1,
  ack_rps: 1.0,
  nack_rps: 0,
  ready: 3,
  unacked: 1,
  total: 4,
};

const renderQueueDetails = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <Provider>
        <MemoryRouter>
          <QueueDetails queueName="test-queue" />
          <Toaster />
        </MemoryRouter>
      </Provider>
    </QueryClientProvider>
  );
};

describe("QueueDetails delete flow", () => {
  beforeEach(() => {
    vi.mocked(getQueue).mockReset().mockResolvedValue(queue);
    vi.mocked(deleteQueue).mockReset();
    mockNavigate.mockReset();
  });

  it("requires confirmation and does not call the API until confirmed", async () => {
    renderQueueDetails();

    await userEvent.click(await screen.findByRole("button", { name: "Actions" }));
    await userEvent.click(await screen.findByText("Delete Queue"));

    expect(await screen.findByText("Delete Queue")).toBeInTheDocument();
    expect(
      screen.getByText("Are you sure? You can't undo this action afterwards.")
    ).toBeInTheDocument();
    expect(deleteQueue).not.toHaveBeenCalled();
  });

  it("deletes the queue, navigates home, and shows a success toast on confirm", async () => {
    vi.mocked(deleteQueue).mockResolvedValueOnce(undefined);
    renderQueueDetails();

    await userEvent.click(await screen.findByRole("button", { name: "Actions" }));
    await userEvent.click(await screen.findByText("Delete Queue"));
    await userEvent.click(await screen.findByRole("button", { name: "Delete" }));

    expect(deleteQueue).toHaveBeenCalledWith({ name: "test-queue" });
    expect(await screen.findByText("Queue deleted.")).toBeInTheDocument();
    expect(mockNavigate).toHaveBeenCalledWith("/");
  });

  it("keeps the user on the page and shows an error toast when deletion fails", async () => {
    vi.mocked(deleteQueue).mockRejectedValueOnce(new Error("queue is not empty"));
    renderQueueDetails();

    await userEvent.click(await screen.findByRole("button", { name: "Actions" }));
    await userEvent.click(await screen.findByText("Delete Queue"));
    await userEvent.click(await screen.findByRole("button", { name: "Delete" }));

    expect(
      await screen.findByText("Failed to delete queue.")
    ).toBeInTheDocument();
    expect(screen.getByText("queue is not empty")).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });
});
