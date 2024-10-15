import {
  FormControl,
  FormLabel,
  Button,
  Input,
  useToast,
  Progress,
  Box,
  Select,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
} from "@chakra-ui/react";
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
  const [type, setType] = useState("delayed");
  const queryClient = useQueryClient();

  const toast = useToast();

  const mutation = useMutation({
    mutationFn: createQueue,
    onSuccess: () => {
      toast({
        title: "Qeueu Created.",
        description: `The queue with name ${name} has been created successfully.`,
        status: "success",
        duration: 9000,
        isClosable: true,
        position: "bottom-right",
      });
      queryClient.invalidateQueries({ queryKey: ["queues"] });
      setName("");
      setType("delayed");
    },
  });

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setName(e.target.value);

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();

    mutation.mutate({
      name: name,
      type: type,
    });
    onClose();
  };

  const isError = name === "";

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>Create Queue</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <form onSubmit={handleSubmit}>
            <FormControl isInvalid={isError}>
              <FormLabel>Name</FormLabel>
              <Input onChange={handleNameChange} value={name} autoFocus></Input>
            </FormControl>
            <FormControl isInvalid={isError}>
              <FormLabel>Type</FormLabel>
              <Select
                defaultValue={"delayed"}
                onChange={(e) => setType(e.target.value)}
              >
                <option value="delayed">Delayed</option>
                <option value="fair">Fair</option>
              </Select>
            </FormControl>

            <Box mt={4}>
              <Progress
                size="xs"
                isIndeterminate
                opacity={mutation.isPending ? 1 : 0}
              />
            </Box>
          </form>
        </ModalBody>

        <ModalFooter>
          <Button colorScheme="green" mr={3} onClick={handleSubmit}>
            Create
          </Button>
          <Button onClick={onClose}>Cancel</Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default CreateQueueModal;
