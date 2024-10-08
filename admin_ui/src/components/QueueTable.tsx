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
} from "@chakra-ui/react";
import { Badge } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom"; // For routing
import { Queue } from "../types/queues";
import { getQueues } from "../api/queues";
import { useQuery } from "@tanstack/react-query";

const QueueTable = () => {
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
                <Badge colorScheme={queue.type === "delayed" ? "teal" : "teal"}>
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
  );
};

export default QueueTable;
