import {
  Field,
  Button,
  Input,
  Progress,
  Box,
  Select,
  Dialog,
  Fieldset,
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
  const [strategy, setStrategy] = useState(["weighted"]);
  const [maxUnacked, setMaxUnacked] = useState(0);
  const [ackTimeout, setAckTimeout] = useState(1800);
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: createQueue,
    onSuccess: () => {
      toaster.create({
        title: "Queue Created.",
        description: `The queue with name ${name} has been created successfully.`,
        type: "success",
        duration: 9000,
      });
      queryClient.invalidateQueries({ queryKey: ["queues"] });
      setName("");
      setType(["delayed"]);
      setStrategy(["weighted"]);
      setMaxUnacked(0);
      setAckTimeout(0);
    },
  });

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setName(e.target.value);

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      name: name,
      type: type[0],
      maxUnacked: maxUnacked,
      strategy: strategy[0],
      ackTimeout: ackTimeout,
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
  const strategies = createListCollection({
    items: [
      { label: "Weighted", value: "weighted" },
      { label: "Round Robin", value: "round_robin" },
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
                <Field.Label>Acknowledgement Timeout</Field.Label>
                <Input
                  onChange={(e) => setAckTimeout(Number(e.target.value))}
                  value={ackTimeout}
                  type="number"
                  autoFocus
                ></Input>
              </Field.Root>

              <Field.Root invalid={isError}>
                <Field.Label>Type</Field.Label>

                <Select.Root
                  size="sm"
                  collection={types}
                  defaultValue={type}
                  onValueChange={(v) => setType(v.value)}
                >
                  <Select.HiddenSelect />
                  <Select.Control>
                    <Select.Trigger>
                      <Select.ValueText placeholder="Select type" />
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
              {type[0] === "fair" ? (
                <Fieldset.Root mt={10}>
                  <Fieldset.Legend>Settings</Fieldset.Legend>

                  <Field.Root invalid={isError}>
                    <Field.Label>Strategy</Field.Label>
                    <Select.Root
                      size="sm"
                      collection={strategies}
                      defaultValue={strategy}
                      onValueChange={(v) => setStrategy(v.value)}
                    >
                      <Select.HiddenSelect />
                      <Select.Control>
                        <Select.Trigger>
                          <Select.ValueText placeholder="Select strategy" />
                        </Select.Trigger>
                        <Select.IndicatorGroup>
                          <Select.Indicator />
                        </Select.IndicatorGroup>
                      </Select.Control>
                      <Select.Positioner>
                        <Select.Content>
                          {strategies.items.map((strategyOption) => (
                            <Select.Item
                              item={strategyOption}
                              key={strategyOption.value}
                            >
                              {strategyOption.label}
                              <Select.ItemIndicator />
                            </Select.Item>
                          ))}
                        </Select.Content>
                      </Select.Positioner>
                    </Select.Root>
                  </Field.Root>

                  <Field.Root invalid={isError}>
                    <Field.Label>Max Unacked</Field.Label>
                    <Input
                      onChange={(e) => setMaxUnacked(Number(e.target.value))}
                      value={maxUnacked}
                      type="number"
                      autoFocus
                    ></Input>
                  </Field.Root>
                </Fieldset.Root>
              ) : (
                ""
              )}
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
