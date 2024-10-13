import {
  FormControl,
  FormLabel,
  Button,
  Input,
  Textarea,
  Stack,
  useToast,
  Card,
  CardHeader,
  CardBody,
  Heading,
  NumberInput,
  NumberInputField,
  Progress,
  Box,
} from "@chakra-ui/react";
import { SetStateAction, useState } from "react";
import { EnqueueMessage } from "../api/messages";
import { useMutation } from "@tanstack/react-query";

const EnqueueMessageForm = ({ queueName }: { queueName: string }) => {
  const [content, setContent] = useState("");
  const [group, setGroup] = useState("default");
  const [priority, setPriority] = useState(100);

  const toast = useToast();

  const mutation = useMutation({
    mutationFn: EnqueueMessage,
    onSuccess: (message) => {
      toast({
        title: "Message enqueued.",
        description: `The message with ID ${message.id} has been enqueued successfully.`,
        status: "success",
        duration: 9000,
        isClosable: true,
      });
    },
  });

  const handleContentChange = (e: {
    target: { value: SetStateAction<string> };
  }) => setContent(e.target.value);

  const handleGroupChange = (e: {
    target: { value: SetStateAction<string> };
  }) => setGroup(e.target.value);

  const handlePriorityChange = (
    valueAsString: string,
    valueAsNumber: number
  ): void => {
    setPriority(valueAsNumber);
  };

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    // const message = await EnqueueMessage(queueName, group, priority, content);
    mutation.mutate({
      queueName: queueName,
      group: group,
      priority: priority,
      content: content,
    });
  };

  const isError = content === "";

  return (
    <Card>
      <CardHeader>
        <Heading size="sm" textTransform="uppercase">
          Enqueue message
        </Heading>
      </CardHeader>
      <CardBody>
        <form onSubmit={handleSubmit}>
          <FormControl isInvalid={isError}>
            <FormLabel>Group</FormLabel>
            <Input onChange={handleGroupChange} value={group}></Input>
          </FormControl>
          <FormControl isInvalid={isError}>
            <FormLabel>Priority</FormLabel>
            <NumberInput
              defaultValue={100}
              min={0}
              onChange={handlePriorityChange}
              value={priority}
            >
              <NumberInputField />
            </NumberInput>
          </FormControl>
          <FormControl isInvalid={isError}>
            <FormLabel>Message content</FormLabel>
            <Textarea onChange={handleContentChange} value={content}></Textarea>
          </FormControl>

          <Box mt={4}>
            <Progress
              size="xs"
              isIndeterminate
              opacity={mutation.isPending ? 1 : 0}
            />
          </Box>

          <Stack spacing={5} pt={4} direction="row" align="center">
            <Button colorScheme="teal" type="submit" disabled={isError}>
              Enqueue
            </Button>
          </Stack>
        </form>
      </CardBody>
    </Card>
  );
};

export default EnqueueMessageForm;
