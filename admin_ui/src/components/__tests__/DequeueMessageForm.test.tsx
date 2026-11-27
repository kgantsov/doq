import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import DequeueMessageForm from "../DequeueMessageForm";
import { DequeueMessage, NackMessage } from "../../api/messages";
import { ApiError } from "../../api/errors";

vi.mock("../../api/messages", () => ({
  DequeueMessage: vi.fn(),
  NackMessage: vi.fn(),
}));

describe("DequeueMessageForm", () => {
  beforeEach(() => {
    vi.mocked(DequeueMessage).mockReset();
    vi.mocked(NackMessage).mockReset();
  });

  it("renders the server's problem details when the dequeue request fails", async () => {
    vi.mocked(DequeueMessage).mockRejectedValueOnce(
      new ApiError("Failed to dequeue message: Internal Server Error", 500, {
        status: 500,
        detail: "Failed to dequeue a message from a queue",
        errors: [{ message: "store unavailable" }],
      })
    );

    renderWithProviders(<DequeueMessageForm queueName="test-queue" />);

    await userEvent.click(screen.getByRole("button", { name: "Dequeue" }));

    expect(
      await screen.findByText("Failed to dequeue a message from a queue")
    ).toBeInTheDocument();
    expect(screen.getByText("store unavailable")).toBeInTheDocument();
  });

  it("falls back to the error message when there are no problem details", async () => {
    vi.mocked(DequeueMessage).mockRejectedValueOnce(
      new Error("Failed to dequeue message: Internal Server Error")
    );

    renderWithProviders(<DequeueMessageForm queueName="test-queue" />);

    await userEvent.click(screen.getByRole("button", { name: "Dequeue" }));

    expect(await screen.findByText("Error")).toBeInTheDocument();
    expect(
      screen.getByText("Failed to dequeue message: Internal Server Error")
    ).toBeInTheDocument();
  });

  it("renders the dequeued message on success", async () => {
    vi.mocked(DequeueMessage).mockResolvedValueOnce({
      id: "1",
      group: "default",
      priority: 0,
      content: "hello world",
      status: "DEQUEUED",
      delivery_count: 1,
    });

    renderWithProviders(<DequeueMessageForm queueName="test-queue" />);

    await userEvent.click(screen.getByRole("button", { name: "Dequeue" }));

    expect(await screen.findByText("hello world")).toBeInTheDocument();
  });
});
