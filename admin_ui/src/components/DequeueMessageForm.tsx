import {
  Field,
  Button,
  Switch,
  Text,
  Stack,
  Card,
  Box,
  Heading,
  Progress,
} from "@chakra-ui/react";
import { useState } from "react";
import { DequeueMessage, NackMessage, DequeueRequest } from "../api/messages";
import { Message } from "../types/messages";
import { useMutation } from "@tanstack/react-query";

const DequeueMessageForm = ({ queueName }: { queueName: string }) => {
  const [dequeuedMessage, setDequeuedMessage] = useState<null | Message>(null);
  const [dequeuedAt, setDequeuedAt] = useState("");
  const [ack, setAck] = useState(false);

  const mutation = useMutation({
    mutationFn: async ({ queueName, ack }: DequeueRequest) => {
      const data = await DequeueMessage({ queueName, ack });
      if (ack) return data;
      await NackMessage(queueName, data.id);
      return data;
    },
    onSuccess: (message) => {
      setDequeuedMessage(message);
      setDequeuedAt(new Date().toLocaleString());
    },
  });

  const handleAckChange = (checked: boolean) => setAck(checked);

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      queueName: queueName,
      ack: ack,
    });
  };

  return (
    <>
      <Card.Root>
        <Card.Body>
          <form onSubmit={handleSubmit}>
            <Field.Root>
              <Field.Label>Ack</Field.Label>
              <Switch.Root
                size="lg"
                onCheckedChange={(v) => handleAckChange(v.checked)}
                checked={ack}
              >
                <Switch.HiddenInput />
                <Switch.Control>
                  <Switch.Thumb />
                </Switch.Control>
                <Switch.Label />
              </Switch.Root>
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
              <Button colorPalette="teal" type="submit">
                Dequeue
              </Button>
            </Stack>
          </form>
        </Card.Body>
      </Card.Root>
      {mutation.isError
        ? () => <Text>Failed to dequeue message</Text>
        : dequeuedMessage?.content && (
            <Card.Root mt={4} background="#e0f4ff">
              <Card.Header>
                <Heading size="sm" textTransform="uppercase">
                  Message #{dequeuedMessage.id}
                </Heading>
              </Card.Header>
              <Card.Body>
                <Stack gap="4">
                  <Box>
                    <Heading size="xs" textTransform="uppercase">
                      Group
                    </Heading>
                    <Text pt="2" fontSize="sm">
                      {dequeuedMessage.group}
                    </Text>
                  </Box>
                  <Box>
                    <Heading size="xs" textTransform="uppercase">
                      Priority
                    </Heading>
                    <Text pt="2" fontSize="sm">
                      {dequeuedMessage.priority}
                    </Text>
                  </Box>
                  <Box>
                    <Heading size="xs" textTransform="uppercase">
                      Content
                    </Heading>
                    <Text pt="2" fontSize="sm">
                      {dequeuedMessage.content}
                    </Text>
                  </Box>
                  <Box>
                    <Heading size="xs" textTransform="uppercase">
                      Dequeued at
                    </Heading>
                    <Text pt="2" fontSize="sm">
                      {dequeuedAt}
                    </Text>
                  </Box>
                </Stack>
              </Card.Body>
            </Card.Root>
          )}
    </>
  );
};

export default DequeueMessageForm;
