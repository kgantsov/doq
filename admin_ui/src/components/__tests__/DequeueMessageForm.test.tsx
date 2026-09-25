import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import DequeueMessageForm from "../DequeueMessageForm";
import { DequeueMessage, NackMessage } from "../../api/messages";

vi.mock("../../api/messages", () => ({
  DequeueMessage: vi.fn(),
  NackMessage: vi.fn(),
}));

describe("DequeueMessageForm", () => {
  beforeEach(() => {
    vi.mocked(DequeueMessage).mockReset();
    vi.mocked(NackMessage).mockReset();
  });

  it("renders an error message when the dequeue request fails", async () => {
    vi.mocked(DequeueMessage).mockRejectedValueOnce(
      new Error("Failed to dequeue message: Internal Server Error")
    );

    renderWithProviders(<DequeueMessageForm queueName="test-queue" />);

    await userEvent.click(screen.getByRole("button", { name: "Dequeue" }));

    expect(
      await screen.findByText("Failed to dequeue message")
    ).toBeInTheDocument();
  });

  it("renders the dequeued message on success", async () => {
    vi.mocked(DequeueMessage).mockResolvedValueOnce({
      id: "1",
      group: "default",
      priority: 0,
      content: "hello world",
      status: 0,
    });

    renderWithProviders(<DequeueMessageForm queueName="test-queue" />);

    await userEvent.click(screen.getByRole("button", { name: "Dequeue" }));

    expect(await screen.findByText("hello world")).toBeInTheDocument();
  });
});
