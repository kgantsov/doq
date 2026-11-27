import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import EnqueueMessageForm from "../EnqueueMessageForm";
import { EnqueueMessage } from "../../api/messages";
import { toaster } from "../ui/toaster";

vi.mock("../../api/messages", () => ({
  EnqueueMessage: vi.fn(),
}));

const enqueue = async () => {
  renderWithProviders(<EnqueueMessageForm queueName="test-queue" />);

  await userEvent.type(screen.getByRole("textbox", { name: "Message content" }), "hello");
  await userEvent.click(screen.getByRole("button", { name: "Enqueue" }));
};

describe("EnqueueMessageForm", () => {
  beforeEach(() => {
    vi.mocked(EnqueueMessage).mockReset();
    vi.spyOn(toaster, "create").mockClear();
  });

  it("shows a success toast when the message is enqueued", async () => {
    vi.mocked(EnqueueMessage).mockResolvedValueOnce({
      id: "1",
      group: "default",
      priority: 100,
      content: "hello",
      delivery_count: 0,
      status: "ENQUEUED",
    });

    await enqueue();

    await waitFor(() =>
      expect(toaster.create).toHaveBeenCalledWith(
        expect.objectContaining({ title: "Message enqueued.", type: "success" })
      )
    );
  });

  it("shows a warning toast when the id already exists", async () => {
    vi.mocked(EnqueueMessage).mockResolvedValueOnce({
      id: "7",
      group: "default",
      priority: 10,
      content: "original",
      delivery_count: 1,
      status: "ALREADY_EXISTS",
    });

    await enqueue();

    await waitFor(() =>
      expect(toaster.create).toHaveBeenCalledWith(
        expect.objectContaining({
          title: "Message not enqueued.",
          description: expect.stringContaining("ID 7 already exists"),
          type: "warning",
        })
      )
    );
    expect(toaster.create).not.toHaveBeenCalledWith(
      expect.objectContaining({ type: "success" })
    );
  });
});
