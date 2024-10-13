import {
  Badge,
  Box,
  Text,
  Stat,
  StatLabel,
  StatNumber,
  SimpleGrid,
  Center,
  Spinner,
} from "@chakra-ui/react";
import { Card, CardBody } from "@chakra-ui/react";
import { getQueue } from "../api/queues";
import EnqueueMessageForm from "./EnqueueMessageForm";
import DequeueMessageForm from "./DequeueMessageForm";

import { useQuery } from "@tanstack/react-query";

const QueueDetails = ({ queueName }: { queueName: string }) => {
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
    <Box minWidth="120px">
      <Text fontSize="2xl" mb={4}>
        {queue.name} &nbsp;
        <Badge colorScheme={queue.type === "delayed" ? "teal" : "teal"}>
          {queue.type}
        </Badge>
      </Text>

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
              <StatNumber>{queue.enqueue_rps}/s</StatNumber>
            </Stat>
          </CardBody>
        </Card>

        <Card>
          <CardBody>
            <Stat>
              <StatLabel>Dequeue Rate</StatLabel>
              <StatNumber>{queue.dequeue_rps}/s</StatNumber>
            </Stat>
          </CardBody>
        </Card>

        <Card>
          <CardBody>
            <Stat>
              <StatLabel>Acknowledge Rate</StatLabel>
              <StatNumber>{queue.ack_rps}/s</StatNumber>
            </Stat>
          </CardBody>
        </Card>
      </SimpleGrid>

      <Box pt={4}>
        <DequeueMessageForm queueName={queueName} />
      </Box>
      <Box pt={4}>
        <EnqueueMessageForm queueName={queueName} />
      </Box>
    </Box>
  );
};

export default QueueDetails;
