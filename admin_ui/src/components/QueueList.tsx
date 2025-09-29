import {
  Center,
  Spinner,
  Flex,
  Spacer,
  Menu,
  Button,
  Box,
  Heading,
  Input,
  InputGroup,
  CloseButton,
} from "@chakra-ui/react";
import { Search } from "lucide-react";
import { Badge } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import { Queue } from "../types/queues";
import { getQueues } from "../api/queues";
import { useQuery } from "@tanstack/react-query";
import CreateQueueModal from "./CreateQueueModal";
import { createColumnHelper } from "@tanstack/react-table";
import { DataTable } from "./DataTable";
import { useState } from "react";

import { Tooltip } from "./ui/tooltip";

const columnHelper = createColumnHelper<Queue>();

const columns = [
  columnHelper.accessor("name", {
    cell: (info) => {
      const name = info.getValue();
      return <RouterLink to={`/queues/${name}`}>{name}</RouterLink>;
    },
    header: "Name",
  }),
  columnHelper.accessor("type", {
    cell: (info) => {
      const type = info.getValue();
      return (
        <Tooltip content="The type of the queue">
          <Badge colorPalette={type === "delayed" ? "teal" : "cyan"}>
            {type}
          </Badge>
        </Tooltip>
      );
    },
    header: "Type",
  }),
  columnHelper.accessor("settings", {
    cell: (info) => {
      const settings = info.getValue();
      return (
        <Box margin={1} display="inline-flex" gap={1}>
          {settings?.strategy && (
            <Tooltip content="The strategy used by the queue">
              <Badge colorPalette={"blue"}>{settings?.strategy}</Badge>
            </Tooltip>
          )}
          {settings?.max_unacked && (
            <Tooltip content="The maximum number of unacknowledged messages">
              <Badge colorPalette={"purple"}>{settings?.max_unacked}</Badge>
            </Tooltip>
          )}
          {settings?.ack_timeout && (
            <Tooltip content="The acknowledgement timeout in seconds">
              <Badge colorPalette={"orange"}>{settings?.ack_timeout}</Badge>
            </Tooltip>
          )}
        </Box>
      );
    },
    header: "Settings",
  }),
  columnHelper.accessor("ready", {
    cell: (info) => info.getValue(),
    header: "ready",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("unacked", {
    cell: (info) => info.getValue(),
    header: "unacked",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("total", {
    cell: (info) => info.getValue(),
    header: "total",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("enqueue_rps", {
    cell: (info) => info.getValue().toFixed(1),
    header: "enqueue_rps",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("dequeue_rps", {
    cell: (info) => info.getValue().toFixed(1),
    header: "dequeue_rps",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("ack_rps", {
    cell: (info) => info.getValue().toFixed(1),
    header: "ack_rps",
    meta: {
      isNumeric: true,
    },
  }),
];

const QueueList = () => {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");

  const { isPending, data } = useQuery({
    queryKey: ["queues"],
    queryFn: getQueues,
    refetchInterval: 1000,
  });

  if (isPending) {
    return (
      <Center h="90vh" color="white" colorPalette="teal">
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

  const queues: Queue[] =
    data?.queues?.filter((queue: Queue) => queue.name.includes(query)) || [];

  return (
    <>
      <Box p={5}>
        <Flex>
          <Heading mb={5}>Queues</Heading>
          <Spacer />
          <Menu.Root
            onSelect={(item) => {
              if (item.value === "create-queue") {
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
                <Menu.Item value="create-queue">Create Queue</Menu.Item>
              </Menu.Content>
            </Menu.Positioner>
          </Menu.Root>
        </Flex>
        <InputGroup
          width={"500px"}
          mr={4}
          mb={4}
          startElement={<Search pointerEvents="none" />}
          endElement={
            <CloseButton
              h="1.75rem"
              color="gray.300"
              size="sm"
              onClick={() => setQuery("")}
            ></CloseButton>
          }
        >
          <Input
            type="query"
            placeholder="Search"
            value={query}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setQuery(e.target.value)
            }
          />
        </InputGroup>
        <DataTable columns={columns} data={queues} />
      </Box>
      <CreateQueueModal
        isOpen={isOpen}
        onClose={() => {
          setIsOpen(false);
        }}
      />
    </>
  );
};

export default QueueList;
