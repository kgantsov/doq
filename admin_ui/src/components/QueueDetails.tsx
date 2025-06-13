import {
  Accordion,
  Badge,
  Box,
  Text,
  Stat,
  SimpleGrid,
  Center,
  Spinner,
  Flex,
  Spacer,
  Menu,
  Button,
  Span,
  Dialog,
} from "@chakra-ui/react";
import { useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { toaster } from "./ui/toaster";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { useState } from "react";
import { Card } from "@chakra-ui/react";

import { getQueue, deleteQueue } from "../api/queues";
import { QueueStatsBuffer, QueueStats } from "../stats";
import EnqueueMessageForm from "./EnqueueMessageForm";
import DequeueMessageForm from "./DequeueMessageForm";
import SparkLine from "./SparkLine";
import { Tooltip } from "./ui/tooltip";

const QueueDetails = ({ queueName }: { queueName: string }) => {
  const [isOpen, setIsOpen] = useState(false);
  const navigate = useNavigate();
  const statsBuffer = useRef(new QueueStatsBuffer(600));

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: deleteQueue,
    onSuccess: () => {
      setIsOpen(true);
      navigate(`/`);
      queryClient.invalidateQueries({ queryKey: ["queues"] });
      toaster.create({
        title: "Queue deleted.",
        description: `The queue '${queueName}' has been deleted successfully.`,
        type: "success",
        duration: 9000,
      });
    },
  });

  const {
    isPending,
    dataUpdatedAt,
    data: queue,
  } = useQuery({
    queryKey: ["queue"],
    queryFn: () => getQueue(queueName),
    refetchInterval: 1000,
  });

  useEffect(() => {
    if (queue) {
      statsBuffer.current.append({
        name: queue.name,
        type: queue.type,
        enqueue_rps: queue.enqueue_rps,
        dequeue_rps: queue.dequeue_rps,
        ack_rps: queue.ack_rps,
        nack_rps: queue.nack_rps,
        ready: queue.ready,
        unacked: queue.unacked,
        total: queue.total,
      } as QueueStats);
    }
  }, [dataUpdatedAt]);

  // Clean up buffer on unmount or remount
  useEffect(() => {
    statsBuffer.current.clear();

    return () => {
      statsBuffer.current.clear();
    };
  }, []);

  if (isPending) {
    return (
      <Center h="90vh" color="white">
        <Spinner
          borderWidth="4px"
          animationDuration="0.65s"
          color="teal.500"
          colorPalette="teal"
          size="xl"
        />
      </Center>
    );
  }

  if (!queue) {
    return <></>;
  }

  const chartData = statsBuffer.current.getPoints();

  const accordionContent = [
    {
      value: "dequeue",
      title: "Dequeue messages",
      content: <DequeueMessageForm queueName={queueName} />,
    },
    {
      value: "enqueue",
      title: "Enqueue messages",
      content: <EnqueueMessageForm queueName={queueName} />,
    },
  ];

  return (
    <>
      <Box minWidth="120px">
        <Flex>
          <Box fontSize="2xl" flexDirection="row" mb={4}>
            <Text fontSize="2xl">{queue.name}</Text>
            <Box margin={1} display="inline-flex" gap={1}>
              <Tooltip content="The type of the queue">
                <Badge
                  colorPalette={queue.type === "delayed" ? "teal" : "cyan"}
                >
                  Type: {queue.type}
                </Badge>
              </Tooltip>

              {queue.settings?.strategy && (
                <Tooltip content="The strategy used by the queue">
                  <Badge colorPalette={"blue"}>
                    Strategy: {queue.settings?.strategy}
                  </Badge>
                </Tooltip>
              )}
              {queue.settings?.max_unacked && (
                <Tooltip content="The maximum number of unacknowledged messages">
                  <Badge colorPalette={"purple"}>
                    Max Unacked: {queue.settings?.max_unacked}
                  </Badge>
                </Tooltip>
              )}
            </Box>
          </Box>
          <Spacer />
          <Menu.Root
            onSelect={(item) => {
              if (item.value === "delete-queue") {
                setIsOpen(true);
              }
            }}
          >
            <Menu.Trigger asChild>
              <Button variant="outline" size="sm">
                Actions
              </Button>
            </Menu.Trigger>
            <Menu.Positioner>
              <Menu.Content>
                <Menu.Item value="delete-queue">Delete Queue</Menu.Item>
              </Menu.Content>
            </Menu.Positioner>
          </Menu.Root>
        </Flex>

        <SimpleGrid columns={[2, null, 3]} gap="40px">
          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Ready</Stat.Label>
                <Stat.ValueText>{queue.ready}</Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.ready,
                };
              })}
            />
          </Card.Root>

          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Unacked</Stat.Label>
                <Stat.ValueText>{queue.unacked}</Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.unacked,
                };
              })}
            />
          </Card.Root>

          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Total</Stat.Label>
                <Stat.ValueText>{queue.total}</Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.total,
                };
              })}
            />
          </Card.Root>

          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Enqueue Rate</Stat.Label>
                <Stat.ValueText>
                  {queue.enqueue_rps.toFixed(1)}/s
                </Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.enqueue_rps,
                };
              })}
            />
          </Card.Root>

          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Dequeue Rate</Stat.Label>
                <Stat.ValueText>
                  {queue.dequeue_rps.toFixed(1)}/s
                </Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.dequeue_rps,
                };
              })}
            />
          </Card.Root>

          <Card.Root>
            <Card.Body>
              <Stat.Root>
                <Stat.Label>Acknowledge Rate</Stat.Label>
                <Stat.ValueText>{queue.ack_rps.toFixed(1)}/s</Stat.ValueText>
              </Stat.Root>
            </Card.Body>
            <SparkLine
              data={chartData?.map((item) => {
                return {
                  value: item.ack_rps,
                };
              })}
            />
          </Card.Root>
        </SimpleGrid>

        <Accordion.Root collapsible>
          {accordionContent.map((item, index) => (
            <Accordion.Item key={index} value={item.value}>
              <Accordion.ItemTrigger>
                <Span flex="1">{item.title}</Span>
                <Accordion.ItemIndicator />
              </Accordion.ItemTrigger>
              <Accordion.ItemContent>
                <Accordion.ItemBody>{item.content}</Accordion.ItemBody>
              </Accordion.ItemContent>
            </Accordion.Item>
          ))}
        </Accordion.Root>
      </Box>

      <Dialog.Root open={isOpen}>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Content>
            <Dialog.Header fontSize="lg" fontWeight="bold">
              Delete Queue
            </Dialog.Header>

            <Dialog.Body>
              Are you sure? You can't undo this action afterwards.
            </Dialog.Body>

            <Dialog.Footer>
              <Button variant="outline" onClick={() => setIsOpen(false)}>
                Cancel
              </Button>
              <Button
                colorPalette="red"
                onClick={async () => {
                  mutation.mutate({ name: queueName });
                }}
                ml={3}
              >
                Delete
              </Button>
            </Dialog.Footer>
          </Dialog.Content>
        </Dialog.Positioner>
      </Dialog.Root>
    </>
  );
};

export default QueueDetails;
