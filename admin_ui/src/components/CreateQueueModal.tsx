import {
  Field,
  Button,
  Input,
  Progress,
  Box,
  Select,
  Dialog,
  createListCollection,
} from "@chakra-ui/react";
import { toaster } from "./ui/toaster";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createQueue } from "../api/queues";

const CreateQueueModal = ({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) => {
  const [name, setName] = useState("");
  const [type, setType] = useState(["delayed"]);
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: createQueue,
    onSuccess: () => {
      toaster.create({
        title: "Qeueu Created.",
        description: `The queue with name ${name} has been created successfully.`,
        type: "success",
        duration: 9000,
      });
      queryClient.invalidateQueries({ queryKey: ["queues"] });
      setName("");
      setType(["delayed"]);
    },
  });

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setName(e.target.value);

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      name: name,
      type: type[0],
    });
    onClose();
  };

  const isError = name === "";

  const types = createListCollection({
    items: [
      { label: "Delayed", value: "delayed" },
      { label: "Fair", value: "fair" },
    ],
  });

  return (
    <Dialog.Root open={isOpen}>
      <Dialog.Backdrop />
      <Dialog.Positioner>
        <Dialog.Content>
          <Dialog.Header>
            <Dialog.Title>Create Queue</Dialog.Title>
          </Dialog.Header>
          <Dialog.CloseTrigger />
          <Dialog.Body>
            <form onSubmit={handleSubmit}>
              <Field.Root invalid={isError}>
                <Field.Label>Name</Field.Label>
                <Input
                  onChange={handleNameChange}
                  value={name}
                  autoFocus
                ></Input>
              </Field.Root>
              <Field.Root invalid={isError}>
                <Field.Label>Type</Field.Label>

                <Select.Root
                  size="sm"
                  collection={types}
                  defaultValue={["delayed"]}
                  onValueChange={(v) => setType(v.value)}
                >
                  <Select.HiddenSelect />
                  <Select.Label>Select framework</Select.Label>
                  <Select.Control>
                    <Select.Trigger>
                      <Select.ValueText placeholder="Select framework" />
                    </Select.Trigger>
                    <Select.IndicatorGroup>
                      <Select.Indicator />
                    </Select.IndicatorGroup>
                  </Select.Control>
                  <Select.Positioner>
                    <Select.Content>
                      {types.items.map((typeOption) => (
                        <Select.Item item={typeOption} key={typeOption.value}>
                          {typeOption.label}
                          <Select.ItemIndicator />
                        </Select.Item>
                      ))}
                    </Select.Content>
                  </Select.Positioner>
                </Select.Root>
              </Field.Root>
              <Box mt={4}>
                <Progress.Root
                  size="xs"
                  opacity={mutation.isPending ? 1 : 0}
                  value={null}
                >
                  <Progress.Track>
                    <Progress.Range />
                  </Progress.Track>
                </Progress.Root>
              </Box>
            </form>
          </Dialog.Body>

          <Dialog.Footer>
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button colorPalette="green" mr={3} onClick={handleSubmit}>
              Create
            </Button>
          </Dialog.Footer>
        </Dialog.Content>
      </Dialog.Positioner>
    </Dialog.Root>
  );
};

export default CreateQueueModal;
