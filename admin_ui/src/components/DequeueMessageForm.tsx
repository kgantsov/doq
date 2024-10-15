import {
  FormControl,
  FormLabel,
  Button,
  Switch,
  Text,
  Stack,
  Card,
  CardHeader,
  CardBody,
  Box,
  Heading,
  StackDivider,
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

  const handleAckChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setAck(e.target.checked);

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      queueName: queueName,
      ack: ack,
    });
  };

  return (
    <>
      <Card>
        <CardHeader>
          <Heading size="sm" textTransform="uppercase">
            Dequeue message
          </Heading>
        </CardHeader>
        <CardBody>
          <form onSubmit={handleSubmit}>
            <FormControl>
              <FormLabel>Ack</FormLabel>
              <Switch
                size="lg"
                onChange={handleAckChange}
                isChecked={ack}
              ></Switch>
            </FormControl>
            <Box mt={4}>
              <Progress
                size="xs"
                isIndeterminate
                opacity={mutation.isPending ? 1 : 0}
              />
            </Box>
            <Stack spacing={5} pt={4} direction="row" align="center">
              <Button colorScheme="teal" type="submit">
                Dequeue
              </Button>
            </Stack>
          </form>
        </CardBody>
      </Card>
      {mutation.isError
        ? () => <Text>Failed to dequeue message</Text>
        : dequeuedMessage?.content && (
            <Card mt={4} background="#e0f4ff">
              <CardHeader>
                <Heading size="sm" textTransform="uppercase">
                  Message #{dequeuedMessage.id}
                </Heading>
              </CardHeader>
              <CardBody>
                <Stack divider={<StackDivider />} spacing="4">
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
              </CardBody>
            </Card>
          )}
    </>
  );
};

export default DequeueMessageForm;
