import {
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  Link,
  Center,
  Spinner,
  Flex,
  Spacer,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Button,
  Box,
  Heading,
  useDisclosure,
} from "@chakra-ui/react";
import { ChevronDownIcon } from "@chakra-ui/icons";
import { Badge } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom"; // For routing
import { Queue } from "../types/queues";
import { getQueues } from "../api/queues";
import { useQuery } from "@tanstack/react-query";
import CreateQueueModal from "./CreateQueueModal";

const QueueList = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();

  const { isPending, data } = useQuery({
    queryKey: ["queues"],
    queryFn: getQueues,
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

  const queues: Queue[] = data?.queues || [];

  return (
    <>
      <Box p={5}>
        <Flex>
          <Heading mb={5}>Queues</Heading>
          <Spacer />
          <Menu>
            <MenuButton as={Button} rightIcon={<ChevronDownIcon />}>
              Actions
            </MenuButton>
            <MenuList>
              <MenuItem onClick={onOpen}>Create Queue</MenuItem>
            </MenuList>
          </Menu>
        </Flex>
        <TableContainer>
          <Table variant="simple">
            <Thead>
              <Tr>
                <Th>Name</Th>
                <Th>Type</Th>
                <Th>Ready</Th>
                <Th>Unacked</Th>
                <Th>Total</Th>
                <Th>Enqueue RPS</Th>
                <Th>Dequeue RPS</Th>
                <Th>Ack RPS</Th>
              </Tr>
            </Thead>
            <Tbody>
              {queues.map((queue, index) => (
                <Tr key={index}>
                  <Td>
                    <Link as={RouterLink} to={`/queues/${queue.name}`}>
                      {queue.name}
                    </Link>
                  </Td>
                  <Td>
                    <Badge
                      colorScheme={queue.type === "delayed" ? "teal" : "teal"}
                    >
                      {queue.type}
                    </Badge>
                  </Td>
                  <Td>{queue.ready}</Td>
                  <Td>{queue.unacked}</Td>
                  <Td>{queue.total}</Td>
                  <Td>{queue.enqueue_rps}/s</Td>
                  <Td>{queue.dequeue_rps}/s</Td>
                  <Td>{queue.ack_rps}/s</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </TableContainer>
      </Box>

      <CreateQueueModal isOpen={isOpen} onClose={onClose} />
    </>
  );
};

export default QueueList;
