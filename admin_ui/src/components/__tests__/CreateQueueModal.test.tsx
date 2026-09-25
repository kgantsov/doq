import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/render";
import CreateQueueModal from "../CreateQueueModal";
import { Toaster } from "../ui/toaster";
import { createQueue } from "../../api/queues";

vi.mock("../../api/queues", () => ({
  createQueue: vi.fn(),
}));

describe("CreateQueueModal", () => {
  beforeEach(() => {
    vi.mocked(createQueue).mockReset();
  });

  it("keeps the modal open and shows an error toast when creation fails", async () => {
    vi.mocked(createQueue).mockRejectedValueOnce(new Error("boom"));
    const onClose = vi.fn();

    renderWithProviders(
      <>
        <CreateQueueModal isOpen={true} onClose={onClose} />
        <Toaster />
      </>
    );

    await userEvent.type(screen.getByLabelText("Name"), "my-queue");
    await userEvent.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("Failed to create queue.")).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes the modal and shows a success toast when creation succeeds", async () => {
    vi.mocked(createQueue).mockResolvedValueOnce(undefined);
    const onClose = vi.fn();

    renderWithProviders(
      <>
        <CreateQueueModal isOpen={true} onClose={onClose} />
        <Toaster />
      </>
    );

    await userEvent.type(screen.getByLabelText("Name"), "my-queue");
    await userEvent.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Queue Created.")).toBeInTheDocument();
  });
});
