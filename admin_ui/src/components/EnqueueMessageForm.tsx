import {
  Field,
  Button,
  Input,
  Textarea,
  Stack,
  Card,
  NumberInput,
  Progress,
  Box,
} from "@chakra-ui/react";
import { useState } from "react";
import { EnqueueMessage } from "../api/messages";
import { useMutation } from "@tanstack/react-query";
import { toaster } from "./ui/toaster";

const EnqueueMessageForm = ({ queueName }: { queueName: string }) => {
  const [content, setContent] = useState("");
  const [group, setGroup] = useState("default");
  const [priority, setPriority] = useState(100);

  const mutation = useMutation({
    mutationFn: EnqueueMessage,
    onSuccess: (message) => {
      toaster.create({
        title: "Message enqueued.",
        description: `The message with ID ${message.id} has been enqueued successfully.`,
        type: "success",
        duration: 9000,
      });
    },
  });

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) =>
    setContent(e.target.value);

  const handleGroupChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setGroup(e.target.value);

  const handlePriorityChange = ({
    valueAsNumber,
  }: {
    value: string;
    valueAsNumber: number;
  }): void => {
    setPriority(valueAsNumber);
  };

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      queueName: queueName,
      group: group,
      priority: priority,
      content: content,
    });
  };

  const isError = content === "";

  return (
    <Card.Root>
      <Card.Body>
        <form onSubmit={handleSubmit}>
          <Field.Root invalid={isError}>
            <Field.Label>Group</Field.Label>
            <Input onChange={handleGroupChange} value={group}></Input>
          </Field.Root>
          <Field.Root invalid={isError}>
            <Field.Label>Priority</Field.Label>
            <NumberInput.Root
              defaultValue={"100"}
              min={0}
              onValueChange={handlePriorityChange}
              value={priority.toString()}
            >
              <NumberInput.Input />
            </NumberInput.Root>
          </Field.Root>
          <Field.Root invalid={isError}>
            <Field.Label>Message content</Field.Label>
            <Textarea onChange={handleContentChange} value={content}></Textarea>
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

          <Stack gap={5} pt={4} direction="row" align="center">
            <Button colorPalette="teal" type="submit" disabled={isError}>
              Enqueue
            </Button>
          </Stack>
        </form>
      </Card.Body>
    </Card.Root>
  );
};

export default EnqueueMessageForm;
