import {
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  Badge,
  Box,
  Text,
  Stat,
  StatLabel,
  StatNumber,
  SimpleGrid,
  Center,
  Spinner,
  Flex,
  Spacer,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Button,
  useDisclosure,
  useToast,
} from "@chakra-ui/react";
import { AccordionIcon } from "@chakra-ui/icons";
import { useNavigate } from "react-router-dom";

import { useRef } from "react";
import { ChevronDownIcon } from "@chakra-ui/icons";
import { Card, CardBody } from "@chakra-ui/react";
import { getQueue, deleteQueue } from "../api/queues";
import EnqueueMessageForm from "./EnqueueMessageForm";
import DequeueMessageForm from "./DequeueMessageForm";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

const QueueDetails = ({ queueName }: { queueName: string }) => {
  const { isOpen, onOpen, onClose } = useDisclosure();
  const cancelRef = useRef(null);
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: deleteQueue,
    onSuccess: () => {
      onClose();
      navigate(`/`);
      queryClient.invalidateQueries({ queryKey: ["queues"] });
      toast({
        title: "Qeueu deleted.",
        description: `The queue '${queueName}' has been deleted successfully.`,
        status: "success",
        duration: 9000,
        isClosable: true,
        position: "bottom-right",
      });
    },
  });

  const { isPending, data: queue } = useQuery({
    queryKey: ["queues"],
    queryFn: () => getQueue(queueName),
    refetchInterval: 1000,
  });

  if (isPending) {
    return (
      <Center h="90vh" color="white">
        <Spinner
          thickness="4px"
          speed="0.65s"
          emptyColor="gray.200"
          color="blue.500"
          size="xl"
        />
      </Center>
    );
  }

  if (!queue) {
    return <></>;
  }

  return (
    <>
      <Box minWidth="120px">
        <Flex>
          <Text fontSize="2xl" mb={4}>
            {queue.name} &nbsp;
            <Badge colorScheme={queue.type === "delayed" ? "teal" : "cyan"}>
              {queue.type}
            </Badge>
          </Text>
          <Spacer />
          <Menu>
            <MenuButton as={Button} rightIcon={<ChevronDownIcon />}>
              Actions
            </MenuButton>
            <MenuList>
              <MenuItem onClick={onOpen}>Delete Queue</MenuItem>
            </MenuList>
          </Menu>
        </Flex>

        <SimpleGrid columns={[2, null, 3]} spacing="40px">
          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Ready</StatLabel>
                <StatNumber>{queue.ready}</StatNumber>
              </Stat>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Unacked</StatLabel>
                <StatNumber>{queue.unacked}</StatNumber>
              </Stat>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Total</StatLabel>
                <StatNumber>{queue.total}</StatNumber>
              </Stat>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Enqueue Rate</StatLabel>
                <StatNumber>{queue.enqueue_rps.toFixed(1)}/s</StatNumber>
              </Stat>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Dequeue Rate</StatLabel>
                <StatNumber>{queue.dequeue_rps.toFixed(1)}/s</StatNumber>
              </Stat>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <Stat>
                <StatLabel>Acknowledge Rate</StatLabel>
                <StatNumber>{queue.ack_rps.toFixed(1)}/s</StatNumber>
              </Stat>
            </CardBody>
          </Card>
        </SimpleGrid>
        <Accordion allowMultiple mt={4}>
          <AccordionItem>
            <h2>
              <AccordionButton>
                <Box as="span" flex="1" textAlign="left">
                  Dequeue messages
                </Box>
                <AccordionIcon />
              </AccordionButton>
            </h2>
            <AccordionPanel pb={4}>
              <Box pt={4}>
                <DequeueMessageForm queueName={queueName} />
              </Box>
            </AccordionPanel>
          </AccordionItem>

          <AccordionItem>
            <h2>
              <AccordionButton>
                <Box as="span" flex="1" textAlign="left">
                  Enqueue messages
                </Box>
                <AccordionIcon />
              </AccordionButton>
            </h2>
            <AccordionPanel pb={4}>
              <Box pt={4}>
                <EnqueueMessageForm queueName={queueName} />
              </Box>
            </AccordionPanel>
          </AccordionItem>
        </Accordion>
      </Box>
      <AlertDialog
        isOpen={isOpen}
        leastDestructiveRef={cancelRef}
        onClose={onClose}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              Delete Queue
            </AlertDialogHeader>

            <AlertDialogBody>
              Are you sure? You can't undo this action afterwards.
            </AlertDialogBody>

            <AlertDialogFooter>
              <Button onClick={onClose}>Cancel</Button>
              <Button
                colorScheme="red"
                onClick={async () => {
                  mutation.mutate({ name: queueName });
                }}
                ml={3}
              >
                Delete
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </>
  );
};

export default QueueDetails;
